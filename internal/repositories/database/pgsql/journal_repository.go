package pgsql

import (
	"context"
	"errors"
	"fmt"

	"github.com/SscSPs/money_managemet_app/internal/core/domain"
	portsrepo "github.com/SscSPs/money_managemet_app/internal/core/ports/repositories"
	"github.com/SscSPs/money_managemet_app/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"github.com/SscSPs/money_managemet_app/internal/apperrors"
)

type PgxJournalRepository struct {
	pool *pgxpool.Pool
}

// NewPgxJournalRepository creates a new repository for journal and transaction data.
func NewPgxJournalRepository(pool *pgxpool.Pool) portsrepo.JournalRepository {
	return &PgxJournalRepository{pool: pool}
}

// Ensure PgxJournalRepository implements portsrepo.JournalRepository
var _ portsrepo.JournalRepository = (*PgxJournalRepository)(nil)

// --- Mapping Helpers ---
func toModelJournal(d domain.Journal) models.Journal {
	return models.Journal{
		JournalID:    d.JournalID,
		WorkplaceID:  d.WorkplaceID,
		JournalDate:  d.JournalDate,
		Description:  d.Description,
		CurrencyCode: d.CurrencyCode,
		Status:       models.JournalStatus(d.Status),
		AuditFields: models.AuditFields{
			CreatedAt:     d.CreatedAt,
			CreatedBy:     d.CreatedBy,
			LastUpdatedAt: d.LastUpdatedAt,
			LastUpdatedBy: d.LastUpdatedBy,
		},
	}
}

func toDomainJournal(m models.Journal) domain.Journal {
	return domain.Journal{
		JournalID:    m.JournalID,
		WorkplaceID:  m.WorkplaceID,
		JournalDate:  m.JournalDate,
		Description:  m.Description,
		CurrencyCode: m.CurrencyCode,
		Status:       domain.JournalStatus(m.Status),
		AuditFields: domain.AuditFields{
			CreatedAt:     m.CreatedAt,
			CreatedBy:     m.CreatedBy,
			LastUpdatedAt: m.LastUpdatedAt,
			LastUpdatedBy: m.LastUpdatedBy,
		},
	}
}

func toModelTransaction(d domain.Transaction) models.Transaction {
	return models.Transaction{
		TransactionID:   d.TransactionID,
		JournalID:       d.JournalID,
		AccountID:       d.AccountID,
		Amount:          d.Amount,
		TransactionType: models.TransactionType(d.TransactionType),
		CurrencyCode:    d.CurrencyCode,
		Notes:           d.Notes,
		AuditFields: models.AuditFields{
			CreatedAt:     d.CreatedAt,
			CreatedBy:     d.CreatedBy,
			LastUpdatedAt: d.LastUpdatedAt,
			LastUpdatedBy: d.LastUpdatedBy,
		},
	}
}

func toDomainTransaction(m models.Transaction) domain.Transaction {
	return domain.Transaction{
		TransactionID:   m.TransactionID,
		JournalID:       m.JournalID,
		AccountID:       m.AccountID,
		Amount:          m.Amount,
		TransactionType: domain.TransactionType(m.TransactionType),
		CurrencyCode:    m.CurrencyCode,
		Notes:           m.Notes,
		AuditFields: domain.AuditFields{
			CreatedAt:     m.CreatedAt,
			CreatedBy:     m.CreatedBy,
			LastUpdatedAt: m.LastUpdatedAt,
			LastUpdatedBy: m.LastUpdatedBy,
		},
	}
}

func toDomainTransactionSlice(ms []models.Transaction) []domain.Transaction {
	ds := make([]domain.Transaction, len(ms))
	for i, m := range ms {
		ds[i] = toDomainTransaction(m)
	}
	return ds
}

// --- End Mapping Helpers ---

// SaveJournal saves a journal and its associated transactions within a DB transaction.
func (r *PgxJournalRepository) SaveJournal(ctx context.Context, journal domain.Journal, transactions []domain.Transaction) error {
	// Start a database transaction
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	// Defer rollback in case of error
	defer func() {
		_ = tx.Rollback(ctx) // Ignore rollback error
	}()

	// Convert domain journal to model for insertion
	modelJournal := toModelJournal(journal)

	// 1. Insert the Journal entry
	journalQuery := `
		INSERT INTO journals (journal_id, workplace_id, journal_date, description, currency_code, status, created_at, created_by, last_updated_at, last_updated_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10);
	`
	_, err = tx.Exec(ctx, journalQuery,
		modelJournal.JournalID,
		modelJournal.WorkplaceID,
		modelJournal.JournalDate,
		modelJournal.Description,
		modelJournal.CurrencyCode,
		modelJournal.Status,
		modelJournal.CreatedAt,
		modelJournal.CreatedBy,
		modelJournal.LastUpdatedAt,
		modelJournal.LastUpdatedBy,
	)
	if err != nil {
		return fmt.Errorf("failed to insert journal %s: %w", modelJournal.JournalID, err)
	}

	// 2. Insert all Transaction entries
	// Use pgx batching for potential performance improvement with many transactions
	batch := &pgx.Batch{}
	txnQuery := `
		INSERT INTO transactions (transaction_id, journal_id, account_id, amount, transaction_type, currency_code, notes, created_at, created_by, last_updated_at, last_updated_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11);
	`
	for _, txn := range transactions {
		modelTxn := toModelTransaction(txn)
		batch.Queue(txnQuery,
			modelTxn.TransactionID,
			modelTxn.JournalID,
			modelTxn.AccountID,
			modelTxn.Amount,
			modelTxn.TransactionType,
			modelTxn.CurrencyCode,
			modelTxn.Notes,
			modelTxn.CreatedAt,
			modelTxn.CreatedBy,
			modelTxn.LastUpdatedAt,
			modelTxn.LastUpdatedBy,
		)
	}

	br := tx.SendBatch(ctx, batch)
	// Close the batch results, checking for errors during execution
	if err := br.Close(); err != nil {
		return fmt.Errorf("failed to execute transaction batch for journal %s: %w", modelJournal.JournalID, err)
	}

	// If all inserts were successful, commit the transaction
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction for journal %s: %w", modelJournal.JournalID, err)
	}

	return nil
}

// FindJournalByID retrieves a journal by its ID.
func (r *PgxJournalRepository) FindJournalByID(ctx context.Context, journalID string) (*domain.Journal, error) {
	query := `
		SELECT journal_id, workplace_id, journal_date, description, currency_code, status, created_at, created_by, last_updated_at, last_updated_by
		FROM journals
		WHERE journal_id = $1;
	`
	var modelJournal models.Journal
	err := r.pool.QueryRow(ctx, query, journalID).Scan(
		&modelJournal.JournalID,
		&modelJournal.WorkplaceID,
		&modelJournal.JournalDate,
		&modelJournal.Description,
		&modelJournal.CurrencyCode,
		&modelJournal.Status,
		&modelJournal.CreatedAt,
		&modelJournal.CreatedBy,
		&modelJournal.LastUpdatedAt,
		&modelJournal.LastUpdatedBy,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Map db not found error to application specific error
			return nil, apperrors.ErrNotFound
		}
		// Wrap other potential errors
		return nil, fmt.Errorf("failed to find journal by ID %s: %w", journalID, err)
	}

	domainJournal := toDomainJournal(modelJournal)
	return &domainJournal, nil
}

// FindTransactionsByJournalID retrieves all transactions associated with a specific journal.
func (r *PgxJournalRepository) FindTransactionsByJournalID(ctx context.Context, journalID string) ([]domain.Transaction, error) {
	query := `
		SELECT transaction_id, journal_id, account_id, amount, transaction_type, currency_code, notes, created_at, created_by, last_updated_at, last_updated_by
		FROM transactions
		WHERE journal_id = $1
		ORDER BY created_at; -- Or potentially transaction_id for deterministic order
	`
	rows, err := r.pool.Query(ctx, query, journalID)
	if err != nil {
		return nil, fmt.Errorf("failed to query transactions for journal %s: %w", journalID, err)
	}
	defer rows.Close()

	modelTransactions := []models.Transaction{}
	for rows.Next() {
		var modelTxn models.Transaction
		// Need to scan decimal correctly
		var amount decimal.Decimal

		if err := rows.Scan(
			&modelTxn.TransactionID,
			&modelTxn.JournalID,
			&modelTxn.AccountID,
			&amount,
			&modelTxn.TransactionType,
			&modelTxn.CurrencyCode,
			&modelTxn.Notes,
			&modelTxn.CreatedAt,
			&modelTxn.CreatedBy,
			&modelTxn.LastUpdatedAt,
			&modelTxn.LastUpdatedBy,
		); err != nil {
			return nil, fmt.Errorf("failed to scan transaction row for journal %s: %w", journalID, err)
		}
		modelTxn.Amount = amount // Assign scanned decimal
		modelTransactions = append(modelTransactions, modelTxn)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating transaction rows for journal %s: %w", journalID, err)
	}

	return toDomainTransactionSlice(modelTransactions), nil
}

// FindTransactionsByAccountID retrieves all transactions associated with a specific account.
func (r *PgxJournalRepository) FindTransactionsByAccountID(ctx context.Context, workplaceID, accountID string) ([]domain.Transaction, error) {
	// First, ensure the account itself belongs to the given workplace.
	// This check might be better placed in the service layer.
	// Optional check: _, err := r.pool.Exec(ctx, "SELECT 1 FROM accounts WHERE account_id = $1 AND workplace_id = $2", accountID, workplaceID)

	query := `
        SELECT t.transaction_id, t.journal_id, t.account_id, t.amount, t.transaction_type, t.currency_code, t.notes, t.created_at, t.created_by, t.last_updated_at, t.last_updated_by
        FROM transactions t
        JOIN journals j ON t.journal_id = j.journal_id
        WHERE t.account_id = $1 AND j.workplace_id = $2
        ORDER BY j.journal_date DESC, t.created_at DESC;
    ` // Join with journals to filter by workplace
	rows, err := r.pool.Query(ctx, query, accountID, workplaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to query transactions for account %s in workplace %s: %w", accountID, workplaceID, err)
	}
	defer rows.Close()

	modelTransactions := []models.Transaction{}
	for rows.Next() {
		var modelTxn models.Transaction
		var amount decimal.Decimal
		if err := rows.Scan(
			&modelTxn.TransactionID,
			&modelTxn.JournalID,
			&modelTxn.AccountID,
			&amount,
			&modelTxn.TransactionType,
			&modelTxn.CurrencyCode,
			&modelTxn.Notes,
			&modelTxn.CreatedAt,
			&modelTxn.CreatedBy,
			&modelTxn.LastUpdatedAt,
			&modelTxn.LastUpdatedBy,
		); err != nil {
			return nil, fmt.Errorf("failed to scan transaction row for account %s: %w", accountID, err)
		}
		modelTxn.Amount = amount
		modelTransactions = append(modelTransactions, modelTxn)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating transaction rows for account %s: %w", accountID, err)
	}

	return toDomainTransactionSlice(modelTransactions), nil
}

// ListJournalsByWorkplace retrieves a paginated list of journals for a specific workplace.
func (r *PgxJournalRepository) ListJournalsByWorkplace(ctx context.Context, workplaceID string, limit int, offset int) ([]domain.Journal, error) {
	// Default limit and offset handling
	if limit <= 0 {
		limit = 20 // Or a configurable default
	}
	if offset < 0 {
		offset = 0
	}

	query := `
		SELECT journal_id, workplace_id, journal_date, description, currency_code, status, created_at, created_by, last_updated_at, last_updated_by
		FROM journals
		WHERE workplace_id = $1
		ORDER BY journal_date DESC, created_at DESC -- Order by date, then creation time
		LIMIT $2 OFFSET $3;
	`

	rows, err := r.pool.Query(ctx, query, workplaceID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query journals for workplace %s: %w", workplaceID, err)
	}
	defer rows.Close()

	modelJournals := []models.Journal{}
	for rows.Next() {
		var modelJournal models.Journal
		err := rows.Scan(
			&modelJournal.JournalID,
			&modelJournal.WorkplaceID,
			&modelJournal.JournalDate,
			&modelJournal.Description,
			&modelJournal.CurrencyCode,
			&modelJournal.Status,
			&modelJournal.CreatedAt,
			&modelJournal.CreatedBy,
			&modelJournal.LastUpdatedAt,
			&modelJournal.LastUpdatedBy,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan journal row for workplace %s: %w", workplaceID, err)
		}
		modelJournals = append(modelJournals, modelJournal)
	}

	if rows.Err() != nil {
		return nil, fmt.Errorf("error iterating journal rows for workplace %s: %w", workplaceID, rows.Err())
	}

	// Convert models to domain objects
	domainJournals := make([]domain.Journal, len(modelJournals))
	for i, mj := range modelJournals {
		domainJournals[i] = toDomainJournal(mj) // Assuming toDomainJournal exists
	}

	return domainJournals, nil
}

// FindTransactionsByJournalIDs retrieves all transactions for a given list of journal IDs.
// It returns a map where keys are journal IDs and values are slices of transactions.
func (r *PgxJournalRepository) FindTransactionsByJournalIDs(ctx context.Context, journalIDs []string) (map[string][]domain.Transaction, error) {
	if len(journalIDs) == 0 {
		return map[string][]domain.Transaction{}, nil
	}

	query := `
		SELECT transaction_id, journal_id, account_id, amount, transaction_type, currency_code, notes, created_at, created_by, last_updated_at, last_updated_by
		FROM transactions
		WHERE journal_id = ANY($1)
		ORDER BY journal_id, created_at; -- Order by journal_id for grouping, then by time
	`

	rows, err := r.pool.Query(ctx, query, journalIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to query transactions for journal IDs: %w", err)
	}
	defer rows.Close()

	transactionsMap := make(map[string][]domain.Transaction)
	for rows.Next() {
		var modelTxn models.Transaction
		var amount decimal.Decimal

		if err := rows.Scan(
			&modelTxn.TransactionID,
			&modelTxn.JournalID,
			&modelTxn.AccountID,
			&amount,
			&modelTxn.TransactionType,
			&modelTxn.CurrencyCode,
			&modelTxn.Notes,
			&modelTxn.CreatedAt,
			&modelTxn.CreatedBy,
			&modelTxn.LastUpdatedAt,
			&modelTxn.LastUpdatedBy,
		); err != nil {
			return nil, fmt.Errorf("failed to scan transaction row during batch fetch: %w", err)
		}
		modelTxn.Amount = amount

		domainTxn := toDomainTransaction(modelTxn) // Assuming toDomainTransaction exists
		transactionsMap[domainTxn.JournalID] = append(transactionsMap[domainTxn.JournalID], domainTxn)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating transaction rows during batch fetch: %w", err)
	}

	// Ensure even journals with no transactions have an entry (empty slice)
	for _, jid := range journalIDs {
		if _, exists := transactionsMap[jid]; !exists {
			transactionsMap[jid] = []domain.Transaction{}
		}
	}

	return transactionsMap, nil
}

// TODO: Implement UpdateJournalStatus for M4 (Reversals).

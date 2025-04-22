package pgsql

import (
	"context"
	"errors"
	"fmt"

	"github.com/SscSPs/money_managemet_app/internal/apperrors"
	"github.com/SscSPs/money_managemet_app/internal/core/ports"
	"github.com/SscSPs/money_managemet_app/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

type PgxJournalRepository struct {
	pool *pgxpool.Pool
}

// NewPgxJournalRepository creates a new repository for journal and transaction data.
func NewPgxJournalRepository(pool *pgxpool.Pool) ports.JournalRepository {
	return &PgxJournalRepository{pool: pool}
}

// SaveJournal saves a journal and its associated transactions within a DB transaction.
func (r *PgxJournalRepository) SaveJournal(ctx context.Context, journal models.Journal, transactions []models.Transaction) error {
	// Start a database transaction
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	// Defer rollback in case of error
	defer func() {
		_ = tx.Rollback(ctx) // Ignore rollback error
	}()

	// 1. Insert the Journal entry
	journalQuery := `
		INSERT INTO journals (journal_id, journal_date, description, currency_code, status, created_at, created_by, last_updated_at, last_updated_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9);
	`
	_, err = tx.Exec(ctx, journalQuery,
		journal.JournalID,
		journal.JournalDate,
		journal.Description,
		journal.CurrencyCode,
		journal.Status,
		journal.CreatedAt,
		journal.CreatedBy,
		journal.LastUpdatedAt,
		journal.LastUpdatedBy,
	)
	if err != nil {
		return fmt.Errorf("failed to insert journal %s: %w", journal.JournalID, err)
	}

	// 2. Insert all Transaction entries
	// Use pgx batching for potential performance improvement with many transactions
	batch := &pgx.Batch{}
	txnQuery := `
		INSERT INTO transactions (transaction_id, journal_id, account_id, amount, transaction_type, currency_code, notes, created_at, created_by, last_updated_at, last_updated_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11);
	`
	for _, txn := range transactions {
		batch.Queue(txnQuery,
			txn.TransactionID,
			txn.JournalID, // Already populated by service
			txn.AccountID,
			txn.Amount, // Assumes DB driver handles decimal.Decimal correctly
			txn.TransactionType,
			txn.CurrencyCode, // Already populated by service
			txn.Notes,
			txn.CreatedAt,
			txn.CreatedBy,
			txn.LastUpdatedAt,
			txn.LastUpdatedBy,
		)
	}

	br := tx.SendBatch(ctx, batch)
	// Close the batch results, checking for errors during execution
	if err := br.Close(); err != nil {
		return fmt.Errorf("failed to execute transaction batch for journal %s: %w", journal.JournalID, err)
	}

	// If all inserts were successful, commit the transaction
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction for journal %s: %w", journal.JournalID, err)
	}

	return nil
}

// FindJournalByID retrieves a journal by its ID.
func (r *PgxJournalRepository) FindJournalByID(ctx context.Context, journalID string) (*models.Journal, error) {
	query := `
		SELECT journal_id, journal_date, description, currency_code, status, created_at, created_by, last_updated_at, last_updated_by
		FROM journals
		WHERE journal_id = $1;
	`
	var journal models.Journal
	err := r.pool.QueryRow(ctx, query, journalID).Scan(
		&journal.JournalID,
		&journal.JournalDate,
		&journal.Description,
		&journal.CurrencyCode,
		&journal.Status,
		&journal.CreatedAt,
		&journal.CreatedBy,
		&journal.LastUpdatedAt,
		&journal.LastUpdatedBy,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Map db not found error to application specific error
			return nil, apperrors.ErrNotFound
		}
		// Wrap other potential errors
		return nil, fmt.Errorf("failed to find journal by ID %s: %w", journalID, err)
	}

	return &journal, nil
}

// FindTransactionsByJournalID retrieves all transactions associated with a specific journal.
func (r *PgxJournalRepository) FindTransactionsByJournalID(ctx context.Context, journalID string) ([]models.Transaction, error) {
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

	transactions := []models.Transaction{}
	for rows.Next() {
		var txn models.Transaction
		// Need to scan decimal correctly
		var amount decimal.Decimal

		if err := rows.Scan(
			&txn.TransactionID,
			&txn.JournalID,
			&txn.AccountID,
			&amount,
			&txn.TransactionType,
			&txn.CurrencyCode,
			&txn.Notes,
			&txn.CreatedAt,
			&txn.CreatedBy,
			&txn.LastUpdatedAt,
			&txn.LastUpdatedBy,
		); err != nil {
			return nil, fmt.Errorf("failed to scan transaction row for journal %s: %w", journalID, err)
		}
		txn.Amount = amount // Assign scanned decimal
		transactions = append(transactions, txn)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating transaction rows for journal %s: %w", journalID, err)
	}

	return transactions, nil
}

// FindTransactionsByAccountID retrieves all transactions associated with a specific account.
func (r *PgxJournalRepository) FindTransactionsByAccountID(ctx context.Context, accountID string) ([]models.Transaction, error) {
	query := `
		SELECT transaction_id, journal_id, account_id, amount, transaction_type, currency_code, notes, created_at, created_by, last_updated_at, last_updated_by
		FROM transactions
		WHERE account_id = $1
		ORDER BY created_at; -- Consistent ordering
	`
	rows, err := r.pool.Query(ctx, query, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to query transactions for account %s: %w", accountID, err)
	}
	defer rows.Close()

	transactions := []models.Transaction{}
	for rows.Next() {
		var txn models.Transaction
		var amount decimal.Decimal // Use decimal for scanning

		if err := rows.Scan(
			&txn.TransactionID,
			&txn.JournalID,
			&txn.AccountID,
			&amount, // Scan into decimal variable
			&txn.TransactionType,
			&txn.CurrencyCode,
			&txn.Notes,
			&txn.CreatedAt,
			&txn.CreatedBy,
			&txn.LastUpdatedAt,
			&txn.LastUpdatedBy,
		); err != nil {
			return nil, fmt.Errorf("failed to scan transaction row for account %s: %w", accountID, err)
		}
		txn.Amount = amount // Assign scanned decimal to model field
		transactions = append(transactions, txn)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating transaction rows for account %s: %w", accountID, err)
	}

	// Return empty slice if no transactions found, not nil
	if transactions == nil {
		return []models.Transaction{}, nil
	}

	return transactions, nil
}

// TODO: Implement UpdateJournalStatus for M4 (Reversals).

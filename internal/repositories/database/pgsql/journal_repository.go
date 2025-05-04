package pgsql

import (
	"context"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/SscSPs/money_managemet_app/internal/apperrors"
	"github.com/SscSPs/money_managemet_app/internal/core/domain"
	portsrepo "github.com/SscSPs/money_managemet_app/internal/core/ports/repositories"
	"github.com/SscSPs/money_managemet_app/internal/models" // Assuming journal_id is UUID
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

type PgxJournalRepository struct {
	pool        *pgxpool.Pool
	accountRepo portsrepo.AccountRepository
}

// newPgxJournalRepository creates a new repository for journal and transaction data.
func newPgxJournalRepository(pool *pgxpool.Pool, accountRepo portsrepo.AccountRepository) portsrepo.JournalRepository {
	return &PgxJournalRepository{
		pool:        pool,
		accountRepo: accountRepo,
	}
}

// Ensure PgxJournalRepository implements portsrepo.JournalRepository
var _ portsrepo.JournalRepository = (*PgxJournalRepository)(nil)

// --- Mapping Helpers ---
func toModelJournal(d domain.Journal) models.Journal {
	return models.Journal{
		JournalID:          d.JournalID,
		WorkplaceID:        d.WorkplaceID,
		JournalDate:        d.JournalDate,
		Description:        d.Description,
		CurrencyCode:       d.CurrencyCode,
		Status:             models.JournalStatus(d.Status),
		OriginalJournalID:  d.OriginalJournalID,
		ReversingJournalID: d.ReversingJournalID,
		Amount:             d.Amount,
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
		JournalID:          m.JournalID,
		WorkplaceID:        m.WorkplaceID,
		JournalDate:        m.JournalDate,
		Description:        m.Description,
		CurrencyCode:       m.CurrencyCode,
		Status:             domain.JournalStatus(m.Status),
		OriginalJournalID:  m.OriginalJournalID,
		ReversingJournalID: m.ReversingJournalID,
		Amount:             m.Amount,
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
		// RunningBalance is set during save
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
		RunningBalance: m.RunningBalance,
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

// getSignedAmountInternal calculates the signed amount for a transaction based on account type.
// This is an internal helper duplicating the logic from journalService to be used within the transaction boundary.
// NOTE: Consider refactoring this logic into a shared utility or finding a way to use the service's method
// without creating circular dependencies or needing service instances here.
func getSignedAmountInternal(txn domain.Transaction, accountType domain.AccountType) (decimal.Decimal, error) {
	signedAmount := txn.Amount
	isDebit := txn.TransactionType == domain.Debit

	switch accountType {
	case domain.Asset, domain.Expense:
		if !isDebit { // Credit to Asset/Expense
			signedAmount = signedAmount.Neg()
		}
	case domain.Liability, domain.Equity, domain.Income:
		if isDebit { // Debit to Liability/Equity/Income
			signedAmount = signedAmount.Neg()
		}
	default:
		return decimal.Zero, fmt.Errorf("unknown account type '%s' encountered for account ID %s", accountType, txn.AccountID)
	}
	return signedAmount, nil
}

// SaveJournal saves a journal, updates account balances, and saves associated transactions within a DB transaction.
func (r *PgxJournalRepository) SaveJournal(ctx context.Context, journal domain.Journal, transactions []domain.Transaction, balanceChanges map[string]decimal.Decimal) error {
	// Use the injected account repository dependency
	accountRepo := r.accountRepo

	// Start a database transaction
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	// Defer rollback in case of error
	defer func() {
		_ = tx.Rollback(ctx) // Ignore rollback error
	}()

	now := journal.CreatedAt // Use consistent time from journal
	userID := journal.CreatedBy

	// 1. Insert the Journal entry using the transaction tx
	modelJournal := toModelJournal(journal)
	journalQuery := `
		INSERT INTO journals (
			journal_id, workplace_id, journal_date, description, currency_code, status, 
			original_journal_id, reversing_journal_id, amount,
			created_at, created_by, last_updated_at, last_updated_by
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13); -- Update placeholders
	`
	_, err = tx.Exec(ctx, journalQuery,
		modelJournal.JournalID,
		modelJournal.WorkplaceID,
		modelJournal.JournalDate,
		modelJournal.Description,
		modelJournal.CurrencyCode,
		modelJournal.Status,
		modelJournal.OriginalJournalID,
		modelJournal.ReversingJournalID,
		modelJournal.Amount,
		modelJournal.CreatedAt,
		modelJournal.CreatedBy,
		modelJournal.LastUpdatedAt,
		modelJournal.LastUpdatedBy,
	)
	if err != nil {
		return fmt.Errorf("failed to insert journal %s: %w", modelJournal.JournalID, err)
	}

	// 2. Lock accounts and get current balances
	accountIDs := make([]string, 0, len(balanceChanges))
	for accID := range balanceChanges {
		accountIDs = append(accountIDs, accID)
	}

	lockedAccounts, err := accountRepo.FindAccountsByIDsForUpdate(ctx, tx, accountIDs)
	if err != nil {
		// Error includes ErrNotFound if any account is missing
		return fmt.Errorf("failed to lock accounts for update: %w", err)
	}

	// 3. Update account balances using the transaction tx
	if err := accountRepo.UpdateAccountBalancesInTx(ctx, tx, balanceChanges, userID, now); err != nil {
		return fmt.Errorf("failed to update account balances: %w", err)
	}

	// 4. Prepare and Insert Transaction entries with calculated running balances
	batch := &pgx.Batch{}
	txnQuery := `
		INSERT INTO transactions (transaction_id, journal_id, account_id, amount, transaction_type, currency_code, notes, created_at, created_by, last_updated_at, last_updated_by, running_balance)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12);
	`
	// Keep track of running balance calculation per account within this journal context
	currentRunningBalances := make(map[string]decimal.Decimal)
	for accID, lockedAcc := range lockedAccounts {
		currentRunningBalances[accID] = lockedAcc.Balance // Start with the balance *before* this journal's changes
	}

	// Sort transactions deterministically (e.g., by creation order or ID) if needed for consistent running balance calc within the journal
	// Sort by TransactionID for deterministic order
	sort.Slice(transactions, func(i, j int) bool {
		return transactions[i].TransactionID < transactions[j].TransactionID
	})

	// For now, we process in the order received.
	for _, txn := range transactions {
		modelTxn := toModelTransaction(txn)
		modelTxn.CreatedAt = now
		modelTxn.LastUpdatedAt = now
		modelTxn.CreatedBy = userID
		modelTxn.LastUpdatedBy = userID

		// Calculate running balance for this specific transaction line
		accountID := txn.AccountID
		lockedAccount, ok := lockedAccounts[accountID]
		if !ok {
			// This should not happen due to the locking step finding all accounts
			return fmt.Errorf("internal error: locked account %s not found during transaction processing", accountID)
		}

		signedAmount, err := getSignedAmountInternal(txn, lockedAccount.AccountType)
		if err != nil {
			return fmt.Errorf("failed to calculate signed amount for transaction %s: %w", txn.TransactionID, err)
		}

		// Calculate the running balance *after* this transaction
		// Uses the balance fetched *before* the bulk update, plus the effect of this single line
		newRunningBalance := currentRunningBalances[accountID].Add(signedAmount)
		modelTxn.RunningBalance = newRunningBalance
		currentRunningBalances[accountID] = newRunningBalance // Update the running balance for the next txn affecting this account *in this journal*

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
			modelTxn.RunningBalance, // Store the calculated running balance
		)
	}

	// 5. Send the batch of transaction inserts
	br := tx.SendBatch(ctx, batch)
	err = br.Close() // Important: Close the batch results to check for errors in each command
	if err != nil {
		return fmt.Errorf("failed to execute transaction batch for journal %s: %w", modelJournal.JournalID, err)
	}

	// 5. If all inserts/updates were successful, commit the transaction
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction for journal %s: %w", modelJournal.JournalID, err)
	}

	return nil
}

// FindJournalByID retrieves a journal by its ID.
func (r *PgxJournalRepository) FindJournalByID(ctx context.Context, journalID string) (*domain.Journal, error) {
	query := `
		SELECT journal_id, workplace_id, journal_date, description, currency_code, status, 
		       original_journal_id, reversing_journal_id, amount,
		       created_at, created_by, last_updated_at, last_updated_by
		FROM journals
		WHERE journal_id = $1;
	`
	var modelJournal models.Journal
	var originalID sql.NullString  // Use sql.NullString for nullable text
	var reversingID sql.NullString // Use sql.NullString for nullable text

	err := r.pool.QueryRow(ctx, query, journalID).Scan(
		&modelJournal.JournalID,
		&modelJournal.WorkplaceID,
		&modelJournal.JournalDate,
		&modelJournal.Description,
		&modelJournal.CurrencyCode,
		&modelJournal.Status,
		&originalID,  // Scan into NullString
		&reversingID, // Scan into NullString
		&modelJournal.Amount,
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

	// Manually assign scanned nullable strings to model pointers before conversion
	if originalID.Valid {
		modelJournal.OriginalJournalID = &originalID.String
	}
	if reversingID.Valid {
		modelJournal.ReversingJournalID = &reversingID.String
	}

	domainJournal := toDomainJournal(modelJournal)
	return &domainJournal, nil
}

// FindTransactionsByJournalID retrieves all transactions associated with a specific journal.
func (r *PgxJournalRepository) FindTransactionsByJournalID(ctx context.Context, journalID string) ([]domain.Transaction, error) {
	query := `
		SELECT transaction_id, journal_id, account_id, amount, transaction_type, currency_code, notes, created_at, created_by, last_updated_at, last_updated_by, running_balance
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
		var t models.Transaction
		err := rows.Scan(
			&t.TransactionID,
			&t.JournalID,
			&t.AccountID,
			&t.Amount,
			&t.TransactionType,
			&t.CurrencyCode,
			&t.Notes,
			&t.CreatedAt,
			&t.CreatedBy,
			&t.LastUpdatedAt,
			&t.LastUpdatedBy,
			&t.RunningBalance, // Scan the running balance
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan transaction row for journal %s: %w", journalID, err)
		}
		transactions = append(transactions, t)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating transaction rows for journal %s: %w", journalID, err)
	}

	return toDomainTransactionSlice(transactions), nil
}

// FindTransactionsByAccountID retrieves all transactions associated with a specific account.
func (r *PgxJournalRepository) FindTransactionsByAccountID(ctx context.Context, workplaceID, accountID string) ([]domain.Transaction, error) {
	// First, ensure the account itself belongs to the given workplace.
	// This check might be better placed in the service layer.
	// Optional check: _, err := r.pool.Exec(ctx, "SELECT 1 FROM accounts WHERE account_id = $1 AND workplace_id = $2", accountID, workplaceID)

	query := `
        SELECT t.transaction_id, t.journal_id, t.account_id, t.amount, t.transaction_type, t.currency_code, t.notes, t.created_at, t.created_by, t.last_updated_at, t.last_updated_by, t.running_balance
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

	transactions := []models.Transaction{}
	for rows.Next() {
		var t models.Transaction
		err := rows.Scan(
			&t.TransactionID,
			&t.JournalID,
			&t.AccountID,
			&t.Amount,
			&t.TransactionType,
			&t.CurrencyCode,
			&t.Notes,
			&t.CreatedAt,
			&t.CreatedBy,
			&t.LastUpdatedAt,
			&t.LastUpdatedBy,
			&t.RunningBalance, // Scan the running balance
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan transaction row for account %s: %w", accountID, err)
		}
		transactions = append(transactions, t)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating transaction rows for account %s: %w", accountID, err)
	}

	return toDomainTransactionSlice(transactions), nil
}

// encodeTransactionToken creates a base64 encoded token from a transaction's journal date and creation time.
func encodeTransactionToken(journalDate time.Time, createdAt time.Time) string {
	return encodeToken(journalDate, createdAt)
}

// decodeTransactionToken parses the base64 encoded token back into journal date and creation time.
func decodeTransactionToken(token string) (time.Time, time.Time, error) {
	return decodeToken(token)
}

// ListTransactionsByAccountID retrieves a paginated list of transactions for a specific account using token-based pagination.
// It returns the transactions, a token for the next page, and an error.
func (r *PgxJournalRepository) ListTransactionsByAccountID(ctx context.Context, workplaceID, accountID string, limit int, nextToken *string) ([]domain.Transaction, *string, error) {
	// Default limit handling
	if limit <= 0 {
		limit = 20 // Or a configurable default
	}
	// We fetch one extra item to determine if there's a next page.
	fetchLimit := limit + 1

	// Base query
	baseQuery := `
		SELECT t.transaction_id, t.journal_id, t.account_id, t.amount, t.transaction_type, t.currency_code, t.notes, 
		       t.created_at, t.created_by, t.last_updated_at, t.last_updated_by, t.running_balance, j.journal_date
		FROM transactions t
		JOIN journals j ON t.journal_id = j.journal_id
		WHERE t.account_id = $1 AND j.workplace_id = $2
	`
	// Ordering is crucial and must be stable
	// We use journal_date DESC, and created_at DESC as a tie-breaker.
	orderByClause := `ORDER BY j.journal_date DESC, t.created_at DESC`

	var rows pgx.Rows
	var err error
	args := []interface{}{accountID, workplaceID}

	if nextToken != nil && *nextToken != "" {
		// Decode the token to get the cursor values
		lastJournalDate, lastCreatedAt, decodeErr := decodeTransactionToken(*nextToken)
		if decodeErr != nil {
			// Consider logging the error but return a user-friendly error
			return nil, nil, fmt.Errorf("invalid nextToken: %w", decodeErr) // Return specific error for bad token
		}

		// Add cursor condition to WHERE clause
		// Tuple comparison is concise and efficient in Postgres
		cursorClause := `AND (j.journal_date, t.created_at) < ($3, $4)`
		args = append(args, lastJournalDate, lastCreatedAt)

		// Construct the full query with cursor
		query := fmt.Sprintf("%s %s %s LIMIT $%d;", baseQuery, cursorClause, orderByClause, len(args)+1)
		args = append(args, fetchLimit)

		rows, err = r.pool.Query(ctx, query, args...)
	} else {
		// First page request (no token)
		query := fmt.Sprintf("%s %s LIMIT $%d;", baseQuery, orderByClause, len(args)+1)
		args = append(args, fetchLimit)
		rows, err = r.pool.Query(ctx, query, args...)
	}

	// Error handling for query execution
	if err != nil {
		return nil, nil, fmt.Errorf("failed to query transactions for account %s in workplace %s: %w", accountID, workplaceID, err)
	}
	defer rows.Close()

	// Scan results
	transactions := make([]struct {
		transaction models.Transaction
		journalDate time.Time
	}, 0, fetchLimit)

	for rows.Next() {
		var t models.Transaction
		var journalDate time.Time
		err := rows.Scan(
			&t.TransactionID,
			&t.JournalID,
			&t.AccountID,
			&t.Amount,
			&t.TransactionType,
			&t.CurrencyCode,
			&t.Notes,
			&t.CreatedAt,
			&t.CreatedBy,
			&t.LastUpdatedAt,
			&t.LastUpdatedBy,
			&t.RunningBalance,
			&journalDate,
		)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to scan transaction row for account %s: %w", accountID, err)
		}
		transactions = append(transactions, struct {
			transaction models.Transaction
			journalDate time.Time
		}{t, journalDate})
	}

	// Check for errors during row iteration
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("error iterating transaction rows for account %s: %w", accountID, err)
	}

	// Determine the next token
	var nextTokenVal *string
	var results []models.Transaction
	if len(transactions) > limit {
		// There is a next page
		lastTxn := transactions[limit-1] // The *actual* last item of the *current* page
		// The token points to the *last item included* in this response page.
		// The next query will start *after* this item.
		token := encodeTransactionToken(lastTxn.journalDate, lastTxn.transaction.CreatedAt)
		nextTokenVal = &token

		// Extract just the transactions for the result (without the extra one)
		results = make([]models.Transaction, limit)
		for i := 0; i < limit; i++ {
			results[i] = transactions[i].transaction
		}
	} else {
		// No next page
		results = make([]models.Transaction, len(transactions))
		for i, t := range transactions {
			results[i] = t.transaction
		}
	}

	// Convert models to domain objects
	return toDomainTransactionSlice(results), nextTokenVal, nil
}

// --- Token Pagination Helpers ---

const timeFormat = time.RFC3339Nano // Use a precise time format

// encodeToken creates a base64 encoded token from journal date and creation time.
func encodeToken(journalDate time.Time, createdAt time.Time) string {
	tokenStr := fmt.Sprintf("%s|%s", journalDate.Format(timeFormat), createdAt.Format(timeFormat))
	return base64.StdEncoding.EncodeToString([]byte(tokenStr))
}

// decodeToken parses the base64 encoded token back into journal date and creation time.
func decodeToken(token string) (time.Time, time.Time, error) {
	decodedBytes, err := base64.StdEncoding.DecodeString(token)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid pagination token format (base64 decode): %w", err)
	}
	tokenStr := string(decodedBytes)
	parts := strings.SplitN(tokenStr, "|", 2)
	if len(parts) != 2 {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid pagination token format (split)")
	}

	journalDate, err := time.Parse(timeFormat, parts[0])
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid pagination token format (journal date parse): %w", err)
	}

	createdAt, err := time.Parse(timeFormat, parts[1])
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid pagination token format (created_at parse): %w", err)
	}

	return journalDate, createdAt, nil
}

// --- End Token Pagination Helpers ---

// ListJournalsByWorkplace retrieves a paginated list of journals for a specific workplace using token-based pagination.
// It returns the list of journals, a token for the next page (if any), and an error.
func (r *PgxJournalRepository) ListJournalsByWorkplace(ctx context.Context, workplaceID string, limit int, nextToken *string, includeReversals bool) ([]domain.Journal, *string, error) {
	// Default limit handling
	if limit <= 0 {
		limit = 20 // Or a configurable default
	}
	// We fetch one extra item to determine if there's a next page.
	fetchLimit := limit + 1

	// Base query
	baseQuery := `
		SELECT journal_id, workplace_id, journal_date, description, currency_code, status, 
		       original_journal_id, reversing_journal_id, amount,
		       created_at, created_by, last_updated_at, last_updated_by
		FROM journals
	`
	// Filtering criteria - conditionally include/exclude reversed and reversing journals
	filterClause := `WHERE workplace_id = $1`
	if !includeReversals {
		filterClause += ` AND status != 'REVERSED' AND reversing_journal_id IS NULL AND original_journal_id IS NULL`
	}

	// Ordering is crucial and must be stable
	// We use journal_date DESC, and created_at DESC as a tie-breaker.
	// Ensure journal_id is sortable (UUIDs can be compared lexicographically in Postgres).
	orderByClause := `ORDER BY journal_date DESC, created_at DESC`

	var rows pgx.Rows
	var err error
	args := []interface{}{workplaceID}

	if nextToken != nil && *nextToken != "" {
		// Decode the token to get the cursor values
		lastDate, lastCreatedAt, decodeErr := decodeToken(*nextToken)
		if decodeErr != nil {
			// Consider logging the error but return a user-friendly error
			// Or potentially treat as if no token was provided if lenient
			return nil, nil, fmt.Errorf("invalid nextToken: %w", decodeErr) // Return specific error for bad token
		}

		// Add cursor condition to WHERE clause
		// Tuple comparison is concise and efficient in Postgres
		cursorClause := `AND (journal_date, created_at) < ($2, $3)`
		args = append(args, lastDate, lastCreatedAt)

		// Construct the full query with cursor
		query := fmt.Sprintf("%s %s %s %s LIMIT $%d;", baseQuery, filterClause, cursorClause, orderByClause, len(args)+1)
		args = append(args, fetchLimit)

		rows, err = r.pool.Query(ctx, query, args...)

	} else {
		// First page request (no token)
		query := fmt.Sprintf("%s %s %s LIMIT $%d;", baseQuery, filterClause, orderByClause, len(args)+1)
		args = append(args, fetchLimit)
		rows, err = r.pool.Query(ctx, query, args...)
	}

	// Error handling for query execution
	if err != nil {
		// Check for specific DB errors if needed
		return nil, nil, fmt.Errorf("failed to query journals for workplace %s: %w", workplaceID, err)
	}
	defer rows.Close()

	// Scan results
	modelJournals := make([]models.Journal, 0, fetchLimit) // Allocate slice with capacity
	for rows.Next() {
		var m models.Journal
		var originalID sql.NullString
		var reversingID sql.NullString

		scanErr := rows.Scan(
			&m.JournalID,
			&m.WorkplaceID,
			&m.JournalDate,
			&m.Description,
			&m.CurrencyCode,
			&m.Status,
			&originalID,
			&reversingID,
			&m.Amount,
			&m.CreatedAt,
			&m.CreatedBy,
			&m.LastUpdatedAt,
			&m.LastUpdatedBy,
		)
		if scanErr != nil {
			// Log detailed error
			return nil, nil, fmt.Errorf("failed to scan journal row for workplace %s: %w", workplaceID, scanErr)
		}

		if originalID.Valid {
			m.OriginalJournalID = &originalID.String
		}
		if reversingID.Valid {
			m.ReversingJournalID = &reversingID.String
		}
		modelJournals = append(modelJournals, m)
	}

	// Check for errors during row iteration
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("error iterating journal rows for workplace %s: %w", workplaceID, err)
	}

	// Determine the next token
	var nextTokenVal *string
	results := modelJournals
	if len(modelJournals) > limit {
		// There is a next page
		lastJournal := modelJournals[limit-1] // The *actual* last item of the *current* page
		// The token points to the *last item included* in this response page.
		// The next query will start *after* this item.
		newToken := encodeToken(lastJournal.JournalDate, lastJournal.CreatedAt)
		nextTokenVal = &newToken
		// Trim the extra item fetched
		results = modelJournals[:limit]
	}
	// If len(modelJournals) <= limit, it means we fetched limit or fewer items,
	// so there are no more items after this page. nextTokenVal remains nil.

	// Convert models to domain objects
	domainJournals := make([]domain.Journal, len(results))
	for i, m := range results {
		domainJournals[i] = toDomainJournal(m)
	}

	return domainJournals, nextTokenVal, nil
}

// FindTransactionsByJournalIDs retrieves all transactions for a given list of journal IDs.
// It returns a map where keys are journal IDs and values are slices of transactions.
func (r *PgxJournalRepository) FindTransactionsByJournalIDs(ctx context.Context, journalIDs []string) (map[string][]domain.Transaction, error) {
	if len(journalIDs) == 0 {
		return map[string][]domain.Transaction{}, nil
	}

	query := `
		SELECT transaction_id, journal_id, account_id, amount, transaction_type, currency_code, notes, created_at, created_by, last_updated_at, last_updated_by, running_balance
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
		var runningBalancePtr *decimal.Decimal // Use pointer for nullable column

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
			&runningBalancePtr, // Scan into pointer
		); err != nil {
			return nil, fmt.Errorf("failed to scan transaction row during batch fetch: %w", err)
		}
		modelTxn.Amount = amount
		if runningBalancePtr != nil {
			modelTxn.RunningBalance = *runningBalancePtr // Assign dereferenced value if not null
		} else {
			modelTxn.RunningBalance = decimal.Zero // Assign default value if null
		}

		domainTxn := toDomainTransaction(modelTxn) // Includes RunningBalance now
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

// UpdateJournalStatusAndLinks updates the status and reversal links for a journal.
func (r *PgxJournalRepository) UpdateJournalStatusAndLinks(ctx context.Context, journalID string, status domain.JournalStatus, reversingJournalID *string, originalJournalID *string, updatedByUserID string, updatedAt time.Time) error {
	query := `
		UPDATE journals
		SET status = $2,
		    reversing_journal_id = $3,
		    original_journal_id = $4,
		    last_updated_at = $5,
		    last_updated_by = $6
		WHERE journal_id = $1;
	`

	cmdTag, err := r.pool.Exec(ctx, query,
		journalID,
		status,
		reversingJournalID,
		originalJournalID,
		updatedAt,
		updatedByUserID,
	)

	if err != nil {
		return fmt.Errorf("failed to update journal status/links for %s: %w", journalID, err)
	}

	if cmdTag.RowsAffected() == 0 {
		// Journal with the given ID was not found
		return fmt.Errorf("%w: journal %s not found for update", apperrors.ErrNotFound, journalID)
	}

	return nil
}

// UpdateJournal updates non-transaction details of a journal entry.
func (r *PgxJournalRepository) UpdateJournal(ctx context.Context, journal domain.Journal) error {
	modelJournal := toModelJournal(journal)

	query := `
		UPDATE journals
		SET journal_date = $2,
		    description = $3,
		    last_updated_at = $4,
		    last_updated_by = $5
		WHERE journal_id = $1;
	`
	// Note: Status, CurrencyCode, OriginalJournalID, ReversingJournalID are not updated here.
	// Status updates should go through UpdateJournalStatusAndLinks or similar specific methods.

	cmdTag, err := r.pool.Exec(ctx, query,
		modelJournal.JournalID,
		modelJournal.JournalDate,
		modelJournal.Description,
		modelJournal.LastUpdatedAt,
		modelJournal.LastUpdatedBy,
	)

	if err != nil {
		return fmt.Errorf("failed to execute update journal %s: %w", modelJournal.JournalID, err)
	}

	if cmdTag.RowsAffected() == 0 {
		// Journal with the given ID was not found
		return fmt.Errorf("%w: journal %s not found for update", apperrors.ErrNotFound, modelJournal.JournalID)
	}

	return nil
}

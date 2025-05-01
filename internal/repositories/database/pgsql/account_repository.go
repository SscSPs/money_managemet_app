package pgsql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/SscSPs/money_managemet_app/internal/apperrors"
	"github.com/SscSPs/money_managemet_app/internal/core/domain"
	portsrepo "github.com/SscSPs/money_managemet_app/internal/core/ports/repositories"
	"github.com/SscSPs/money_managemet_app/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
	// Import pgx specifically for error handling like ErrNoRows if needed
	// "github.com/jackc/pgx/v5"
)

type PgxAccountRepository struct {
	pool *pgxpool.Pool
}

// newPgxAccountRepository creates a new repository for account data.
func newPgxAccountRepository(pool *pgxpool.Pool) portsrepo.AccountRepository {
	return &PgxAccountRepository{pool: pool}
}

// Ensure PgxAccountRepository implements portsrepo.AccountRepository
var _ portsrepo.AccountRepository = (*PgxAccountRepository)(nil)

// Helper to convert domain.Account to models.Account for DB storage
func toModelAccount(d domain.Account) models.Account {
	return models.Account{
		AccountID:       d.AccountID,
		WorkplaceID:     d.WorkplaceID,
		Name:            d.Name,
		AccountType:     models.AccountType(d.AccountType),
		CurrencyCode:    d.CurrencyCode,
		ParentAccountID: d.ParentAccountID,
		Description:     d.Description,
		IsActive:        d.IsActive,
		AuditFields: models.AuditFields{
			CreatedAt:     d.CreatedAt,
			CreatedBy:     d.CreatedBy,
			LastUpdatedAt: d.LastUpdatedAt,
			LastUpdatedBy: d.LastUpdatedBy,
		},
		Balance: d.Balance,
	}
}

// Helper to convert models.Account from DB to domain.Account
func toDomainAccount(m models.Account) domain.Account {
	return domain.Account{
		AccountID:       m.AccountID,
		WorkplaceID:     m.WorkplaceID,
		Name:            m.Name,
		AccountType:     domain.AccountType(m.AccountType),
		CurrencyCode:    m.CurrencyCode,
		ParentAccountID: m.ParentAccountID,
		Description:     m.Description,
		IsActive:        m.IsActive,
		AuditFields: domain.AuditFields{
			CreatedAt:     m.CreatedAt,
			CreatedBy:     m.CreatedBy,
			LastUpdatedAt: m.LastUpdatedAt,
			LastUpdatedBy: m.LastUpdatedBy,
		},
		Balance: m.Balance,
	}
}

// SaveAccount inserts a new account.
// Note: Update/Inactivate logic will be added in later milestones/methods.
func (r *PgxAccountRepository) SaveAccount(ctx context.Context, account domain.Account) error {
	modelAcc := toModelAccount(account)

	query := `
		INSERT INTO accounts (account_id, workplace_id, name, account_type, currency_code, parent_account_id, description, is_active, created_at, created_by, last_updated_at, last_updated_by, balance)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13);
	`
	// Use sql.NullString for potentially NULL parent_account_id
	var parentID sql.NullString
	if modelAcc.ParentAccountID != "" {
		parentID = sql.NullString{String: modelAcc.ParentAccountID, Valid: true}
	}

	_, err := r.pool.Exec(ctx, query,
		modelAcc.AccountID,
		modelAcc.WorkplaceID,
		modelAcc.Name,
		modelAcc.AccountType,
		modelAcc.CurrencyCode,
		parentID, // Pass sql.NullString
		modelAcc.Description,
		modelAcc.IsActive,
		modelAcc.CreatedAt,
		modelAcc.CreatedBy,
		modelAcc.LastUpdatedAt,
		modelAcc.CreatedBy, // Corrected: Should use CreatedBy here too
		modelAcc.Balance,
	)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == "23505" { // Unique violation
				// Treat unique violation as a validation error
				return fmt.Errorf("%w: account with ID %s already exists", apperrors.ErrDuplicate, modelAcc.AccountID)
			}
		}
		return fmt.Errorf("failed to save account %s: %w", modelAcc.AccountID, err)
	}
	return nil
}

// FindAccountByID retrieves an account by its ID.
func (r *PgxAccountRepository) FindAccountByID(ctx context.Context, accountID string) (*domain.Account, error) {
	query := `
		SELECT account_id, workplace_id, name, account_type, currency_code, parent_account_id, description, is_active, created_at, created_by, last_updated_at, last_updated_by, balance
		FROM accounts
		WHERE account_id = $1;
	`
	var modelAcc models.Account
	var parentID sql.NullString // Use sql.NullString for scanning
	var balance decimal.Decimal

	err := r.pool.QueryRow(ctx, query, accountID).Scan(
		&modelAcc.AccountID,
		&modelAcc.WorkplaceID,
		&modelAcc.Name,
		&modelAcc.AccountType,
		&modelAcc.CurrencyCode,
		&parentID, // Scan into sql.NullString
		&modelAcc.Description,
		&modelAcc.IsActive,
		&modelAcc.CreatedAt,
		&modelAcc.CreatedBy,
		&modelAcc.LastUpdatedAt,
		&modelAcc.LastUpdatedBy,
		&balance,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.ErrNotFound
		}
		return nil, fmt.Errorf("failed to find account by ID %s: %w", accountID, err)
	}

	if parentID.Valid {
		modelAcc.ParentAccountID = parentID.String
	} else {
		modelAcc.ParentAccountID = ""
	}

	modelAcc.Balance = balance
	domainAcc := toDomainAccount(modelAcc)
	return &domainAcc, nil
}

// TODO: Implement methods for UpdateAccount (for editing/inactivating) and ListAccounts in M2.

// FindAccountsByIDs retrieves multiple accounts by their IDs.
func (r *PgxAccountRepository) FindAccountsByIDs(ctx context.Context, accountIDs []string) (map[string]domain.Account, error) {
	if len(accountIDs) == 0 {
		return map[string]domain.Account{}, nil
	}

	query := `
		SELECT account_id, workplace_id, name, account_type, currency_code, parent_account_id, description, is_active, created_at, created_by, last_updated_at, last_updated_by, balance
		FROM accounts
		WHERE account_id = ANY($1);
	`

	rows, err := r.pool.Query(ctx, query, accountIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to query accounts by IDs: %w", err)
	}
	defer rows.Close()

	accountsMap := make(map[string]domain.Account)
	for rows.Next() {
		var modelAcc models.Account
		var parentID sql.NullString // Use sql.NullString
		var balance decimal.Decimal
		err := rows.Scan(
			&modelAcc.AccountID,
			&modelAcc.WorkplaceID,
			&modelAcc.Name,
			&modelAcc.AccountType,
			&modelAcc.CurrencyCode,
			&parentID, // Scan into sql.NullString
			&modelAcc.Description,
			&modelAcc.IsActive,
			&modelAcc.CreatedAt,
			&modelAcc.CreatedBy,
			&modelAcc.LastUpdatedAt,
			&modelAcc.LastUpdatedBy,
			&balance,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan account row during batch fetch: %w", err)
		}

		if parentID.Valid {
			modelAcc.ParentAccountID = parentID.String
		} else {
			modelAcc.ParentAccountID = ""
		}
		modelAcc.Balance = balance
		accountsMap[modelAcc.AccountID] = toDomainAccount(modelAcc)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating account rows during batch fetch: %w", err)
	}

	// It's possible not all requested IDs were found, the map will simply not contain them.
	// The caller (service) should check if all needed accounts were retrieved.
	return accountsMap, nil
}

// ListAccounts retrieves a paginated list of active accounts FOR A SPECIFIC WORKPLACE.
func (r *PgxAccountRepository) ListAccounts(ctx context.Context, workplaceID string, limit int, offset int) ([]domain.Account, error) {
	// Default limit if not specified or invalid
	if limit <= 0 {
		limit = 20 // Or a configurable default
	}
	// Ensure offset is non-negative
	if offset < 0 {
		offset = 0
	}

	query := `
		SELECT account_id, workplace_id, name, account_type, currency_code, parent_account_id, description, is_active, created_at, created_by, last_updated_at, last_updated_by, balance
		FROM accounts
		WHERE is_active = TRUE AND workplace_id = $1 -- Filter by workplace
		ORDER BY name
		LIMIT $2 OFFSET $3;
	`

	rows, err := r.pool.Query(ctx, query, workplaceID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query accounts for workplace %s: %w", workplaceID, err)
	}
	defer rows.Close()

	accounts := []domain.Account{}
	for rows.Next() {
		var modelAcc models.Account
		var parentID sql.NullString
		var balance decimal.Decimal
		err := rows.Scan(
			&modelAcc.AccountID,
			&modelAcc.WorkplaceID,
			&modelAcc.Name,
			&modelAcc.AccountType,
			&modelAcc.CurrencyCode,
			&parentID,
			&modelAcc.Description,
			&modelAcc.IsActive,
			&modelAcc.CreatedAt,
			&modelAcc.CreatedBy,
			&modelAcc.LastUpdatedAt,
			&modelAcc.LastUpdatedBy,
			&balance,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan account row for workplace %s: %w", workplaceID, err)
		}

		if parentID.Valid {
			modelAcc.ParentAccountID = parentID.String
		} else {
			modelAcc.ParentAccountID = ""
		}
		modelAcc.Balance = balance
		accounts = append(accounts, toDomainAccount(modelAcc))
	}

	if rows.Err() != nil {
		return nil, fmt.Errorf("error iterating account rows for workplace %s: %w", workplaceID, rows.Err())
	}

	return accounts, nil
}

// Helper to convert slice of models.Account to slice of domain.Account
func toDomainAccountSlice(ms []models.Account) []domain.Account {
	ds := make([]domain.Account, len(ms))
	for i, m := range ms {
		ds[i] = toDomainAccount(m)
	}
	return ds
}

// UpdateAccount updates an existing account in the database.
func (r *PgxAccountRepository) UpdateAccount(ctx context.Context, account domain.Account) error {
	modelAcc := toModelAccount(account) // Convert domain to model

	query := `
		UPDATE accounts
		SET name = $2, description = $3, is_active = $4, last_updated_at = $5, last_updated_by = $6
		WHERE account_id = $1;
	`
	// Note: We are not allowing updates to account_type, currency_code, parent_account_id, created_at, created_by here.

	cmdTag, err := r.pool.Exec(ctx, query,
		modelAcc.AccountID,
		modelAcc.Name,
		modelAcc.Description,
		modelAcc.IsActive,
		modelAcc.LastUpdatedAt,
		modelAcc.LastUpdatedBy,
	)

	if err != nil {
		return fmt.Errorf("failed to execute update account %s: %w", modelAcc.AccountID, err)
	}

	if cmdTag.RowsAffected() == 0 {
		// If no rows were affected, the account ID likely didn't exist.
		// This check might be redundant if the service layer already fetched the account.
		return apperrors.ErrNotFound
	}

	return nil
}

// DeactivateAccount marks an account as inactive.
func (r *PgxAccountRepository) DeactivateAccount(ctx context.Context, accountID string, userID string, now time.Time) error {
	query := `
		UPDATE accounts
		SET is_active = FALSE, last_updated_at = $2, last_updated_by = $3
		WHERE account_id = $1 AND is_active = TRUE;
	` // Only update if it was active

	cmdTag, err := r.pool.Exec(ctx, query, accountID, now, userID)
	if err != nil {
		return fmt.Errorf("failed to execute deactivate account %s: %w", accountID, err)
	}

	if cmdTag.RowsAffected() == 0 {
		// If no rows affected, it could be because the account doesn't exist OR it was already inactive.
		// We need to check which case it is.
		_, findErr := r.FindAccountByID(ctx, accountID)
		if errors.Is(findErr, apperrors.ErrNotFound) {
			// Account truly doesn't exist.
			return apperrors.ErrNotFound
		} else if findErr != nil {
			// Some other error finding the account, return that.
			return fmt.Errorf("failed to check account status after deactivation attempt for %s: %w", accountID, findErr)
		}
		// If FindAccountByID succeeded, it means the account exists but was already inactive.
		// Return a validation/conflict error to indicate this.
		return apperrors.ErrValidation // Or potentially apperrors.ErrConflict
	}

	return nil
}

// TODO: Implement methods for UpdateAccount (for editing fields)

// FindAccountsByIDsForUpdate retrieves multiple accounts by IDs and locks the rows for update.
// Must be called within a transaction.
func (r *PgxAccountRepository) FindAccountsByIDsForUpdate(ctx context.Context, tx pgx.Tx, accountIDs []string) (map[string]domain.Account, error) {
	if len(accountIDs) == 0 {
		return map[string]domain.Account{}, nil
	}

	query := `
		SELECT account_id, workplace_id, name, account_type, currency_code, parent_account_id, description, is_active, created_at, created_by, last_updated_at, last_updated_by, balance
		FROM accounts
		WHERE account_id = ANY($1)
		FOR UPDATE;
	`

	rows, err := tx.Query(ctx, query, accountIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to query accounts by IDs for update: %w", err)
	}
	defer rows.Close()

	accountsMap := make(map[string]domain.Account)
	for rows.Next() {
		var modelAcc models.Account
		var parentID sql.NullString // Use sql.NullString
		var balance decimal.Decimal
		err := rows.Scan(
			&modelAcc.AccountID,
			&modelAcc.WorkplaceID,
			&modelAcc.Name,
			&modelAcc.AccountType,
			&modelAcc.CurrencyCode,
			&parentID, // Scan into sql.NullString
			&modelAcc.Description,
			&modelAcc.IsActive,
			&modelAcc.CreatedAt,
			&modelAcc.CreatedBy,
			&modelAcc.LastUpdatedAt,
			&modelAcc.LastUpdatedBy,
			&balance,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan locked account row: %w", err)
		}

		if parentID.Valid {
			modelAcc.ParentAccountID = parentID.String
		} else {
			modelAcc.ParentAccountID = ""
		}
		modelAcc.Balance = balance
		accountsMap[modelAcc.AccountID] = toDomainAccount(modelAcc)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating locked account rows: %w", err)
	}

	// Check if all requested accounts were found and locked
	if len(accountsMap) != len(accountIDs) {
		// Identify missing accounts (could be due to deletion or incorrect IDs)
		missing := []string{}
		requested := make(map[string]bool)
		for _, id := range accountIDs {
			requested[id] = true
		}
		for id := range requested {
			if _, found := accountsMap[id]; !found {
				missing = append(missing, id)
			}
		}
		// Log the missing accounts for debugging
		slog.WarnContext(ctx, "Some accounts requested for update lock were not found", "missing_accounts", missing)
		// Return a specific error indicating not all accounts could be locked/found
		return nil, fmt.Errorf("%w: could not find or lock all requested accounts, missing: %v", apperrors.ErrNotFound, missing)
	}

	return accountsMap, nil
}

// UpdateAccountBalancesInTx updates balances for multiple accounts within a transaction.
func (r *PgxAccountRepository) UpdateAccountBalancesInTx(ctx context.Context, tx pgx.Tx, balanceChanges map[string]decimal.Decimal, userID string, now time.Time) error {
	if len(balanceChanges) == 0 {
		return nil // Nothing to update
	}

	// Prepare the update statement
	// Use COALESCE to handle potential NULL balances if the default wasn't set correctly
	query := `
		UPDATE accounts
		SET balance = COALESCE(balance, 0) + $2, last_updated_at = $3, last_updated_by = $4
		WHERE account_id = $1;
	`

	batch := &pgx.Batch{}
	accountIDs := make([]string, 0, len(balanceChanges))
	for accountID, delta := range balanceChanges {
		if !delta.IsZero() { // Only queue updates if there's a change
			batch.Queue(query, accountID, delta, now, userID)
			accountIDs = append(accountIDs, accountID)
		}
	}

	if batch.Len() == 0 {
		return nil // No non-zero changes
	}

	br := tx.SendBatch(ctx, batch)
	// Important: Check result for each update
	var batchErr error
	updatedCount := 0
	for i := 0; i < batch.Len(); i++ {
		ct, err := br.Exec()
		if err != nil {
			// Capture the first error encountered
			if batchErr == nil {
				batchErr = fmt.Errorf("failed to update balance for account %s: %w", accountIDs[i], err)
			}
		} else if ct.RowsAffected() == 0 {
			// Capture the first error if an account wasn't found (shouldn't happen if locked)
			if batchErr == nil {
				batchErr = fmt.Errorf("%w: account %s not found during balance update", apperrors.ErrNotFound, accountIDs[i])
			}
		} else {
			updatedCount++
		}
	}

	err := br.Close()
	if err != nil && batchErr == nil {
		batchErr = fmt.Errorf("failed to close balance update batch: %w", err)
	}

	if batchErr != nil {
		return batchErr // Return the captured error
	}

	// Optional check: Ensure all expected accounts were updated
	if updatedCount != batch.Len() {
		// This case might indicate a problem, but the individual errors should have been caught above.
		slog.WarnContext(ctx, "Mismatch between expected and actual account balance updates", "expected", batch.Len(), "actual", updatedCount)
		// Consider returning an error here if strict consistency is required.
	}

	return nil
}

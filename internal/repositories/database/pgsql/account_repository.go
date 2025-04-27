package pgsql

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/SscSPs/money_managemet_app/internal/apperrors"
	"github.com/SscSPs/money_managemet_app/internal/core/domain"
	portsrepo "github.com/SscSPs/money_managemet_app/internal/core/ports/repositories"
	"github.com/SscSPs/money_managemet_app/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	// Import pgx specifically for error handling like ErrNoRows if needed
	// "github.com/jackc/pgx/v5"
)

type PgxAccountRepository struct {
	pool *pgxpool.Pool
}

// NewPgxAccountRepository creates a new repository for account data.
func NewPgxAccountRepository(pool *pgxpool.Pool) portsrepo.AccountRepository {
	return &PgxAccountRepository{pool: pool}
}

// Ensure PgxAccountRepository implements portsrepo.AccountRepository
var _ portsrepo.AccountRepository = (*PgxAccountRepository)(nil)

// Helper to convert domain.Account to models.Account for DB storage
func toModelAccount(d domain.Account) models.Account {
	return models.Account{
		AccountID:       d.AccountID,
		Name:            d.Name,
		AccountType:     models.AccountType(d.AccountType), // Conversion
		CurrencyCode:    d.CurrencyCode,
		ParentAccountID: d.ParentAccountID,
		Description:     d.Description,
		IsActive:        d.IsActive,
		AuditFields: models.AuditFields{ // Conversion
			CreatedAt:     d.CreatedAt,
			CreatedBy:     d.CreatedBy,
			LastUpdatedAt: d.LastUpdatedAt,
			LastUpdatedBy: d.LastUpdatedBy,
		},
	}
}

// Helper to convert models.Account from DB to domain.Account
func toDomainAccount(m models.Account) domain.Account {
	return domain.Account{
		AccountID:       m.AccountID,
		Name:            m.Name,
		AccountType:     domain.AccountType(m.AccountType), // Conversion
		CurrencyCode:    m.CurrencyCode,
		ParentAccountID: m.ParentAccountID,
		Description:     m.Description,
		IsActive:        m.IsActive,
		AuditFields: domain.AuditFields{ // Conversion
			CreatedAt:     m.CreatedAt,
			CreatedBy:     m.CreatedBy,
			LastUpdatedAt: m.LastUpdatedAt,
			LastUpdatedBy: m.LastUpdatedBy,
		},
	}
}

// SaveAccount inserts a new account.
// Note: Update/Inactivate logic will be added in later milestones/methods.
func (r *PgxAccountRepository) SaveAccount(ctx context.Context, account domain.Account) error {
	// Convert domain object to model for DB interaction
	modelAcc := toModelAccount(account)
	creatorUserID := modelAcc.CreatedBy

	query := `
		INSERT INTO accounts (account_id, name, account_type, currency_code, parent_account_id, description, is_active, created_at, created_by, last_updated_at, last_updated_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11);
	`
	// Handle potential nil parentAccountID if DB requires NULL explicitly
	var parentID *string
	if modelAcc.ParentAccountID != "" {
		parentID = &modelAcc.ParentAccountID
	}

	_, err := r.pool.Exec(ctx, query,
		modelAcc.AccountID, // Assuming ID is generated beforehand or use DB default
		modelAcc.Name,
		modelAcc.AccountType,
		modelAcc.CurrencyCode,
		parentID,
		modelAcc.Description,
		modelAcc.IsActive,      // Should default to true
		modelAcc.CreatedAt,     // Use time from domain object
		creatorUserID,          // created_by
		modelAcc.LastUpdatedAt, // Use time from domain object
		creatorUserID,          // last_updated_by
	)

	if err != nil {
		// TODO: Check for specific errors like unique constraint violation
		return fmt.Errorf("failed to save account %s: %w", modelAcc.AccountID, err)
	}
	return nil
}

// FindAccountByID retrieves an account by its ID.
func (r *PgxAccountRepository) FindAccountByID(ctx context.Context, accountID string) (*domain.Account, error) {
	query := `
		SELECT account_id, name, account_type, currency_code, parent_account_id, description, is_active, created_at, created_by, last_updated_at, last_updated_by
		FROM accounts
		WHERE account_id = $1;
	`
	var modelAcc models.Account // Scan into model struct
	var parentID *string

	err := r.pool.QueryRow(ctx, query, accountID).Scan(
		&modelAcc.AccountID,
		&modelAcc.Name,
		&modelAcc.AccountType,
		&modelAcc.CurrencyCode,
		&parentID, // Scan into nullable pointer
		&modelAcc.Description,
		&modelAcc.IsActive,
		&modelAcc.CreatedAt,
		&modelAcc.CreatedBy,
		&modelAcc.LastUpdatedAt,
		&modelAcc.LastUpdatedBy,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Map db not found error to application specific error
			return nil, apperrors.ErrNotFound
		}
		// Wrap other potential errors
		return nil, fmt.Errorf("failed to find account by ID %s: %w", accountID, err)
	}

	if parentID != nil {
		modelAcc.ParentAccountID = *parentID
	} else {
		modelAcc.ParentAccountID = ""
	}

	// Convert model object to domain object before returning
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
		SELECT account_id, name, account_type, currency_code, parent_account_id, description, is_active, created_at, created_by, last_updated_at, last_updated_by
		FROM accounts
		WHERE account_id = ANY($1);
	` // Use ANY for array matching

	rows, err := r.pool.Query(ctx, query, accountIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to query accounts by IDs: %w", err)
	}
	defer rows.Close()

	accountsMap := make(map[string]domain.Account)
	for rows.Next() {
		var modelAcc models.Account
		var parentID *string
		err := rows.Scan(
			&modelAcc.AccountID,
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
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan account row during batch fetch: %w", err)
		}

		if parentID != nil {
			modelAcc.ParentAccountID = *parentID
		} else {
			modelAcc.ParentAccountID = ""
		}

		domainAcc := toDomainAccount(modelAcc)
		accountsMap[domainAcc.AccountID] = domainAcc
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating account rows during batch fetch: %w", err)
	}

	// It's possible not all requested IDs were found, the map will simply not contain them.
	// The caller (service) should check if all needed accounts were retrieved.
	return accountsMap, nil
}

// ListAccounts retrieves a paginated list of active accounts.
func (r *PgxAccountRepository) ListAccounts(ctx context.Context, limit int, offset int) ([]domain.Account, error) {
	// Default limit if not specified or invalid
	if limit <= 0 {
		limit = 20 // Or a configurable default
	}
	// Ensure offset is non-negative
	if offset < 0 {
		offset = 0
	}

	query := `
		SELECT account_id, name, account_type, currency_code, parent_account_id, description, is_active, created_at, created_by, last_updated_at, last_updated_by
		FROM accounts
		WHERE is_active = TRUE -- Only list active accounts by default
		ORDER BY name -- Or account_type, name; Or created_at
		LIMIT $1 OFFSET $2;
	`

	rows, err := r.pool.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query accounts: %w", err)
	}
	defer rows.Close()

	modelAccounts := []models.Account{}
	for rows.Next() {
		var modelAcc models.Account
		var parentID *string
		err := rows.Scan(
			&modelAcc.AccountID,
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
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan account row during list: %w", err)
		}

		if parentID != nil {
			modelAcc.ParentAccountID = *parentID
		} else {
			modelAcc.ParentAccountID = ""
		}
		modelAccounts = append(modelAccounts, modelAcc)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating account rows during list: %w", err)
	}

	return toDomainAccountSlice(modelAccounts), nil // Use mapping helper
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

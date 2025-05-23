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
	"github.com/SscSPs/money_managemet_app/internal/utils/mapping"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
	// Import pgx specifically for error handling like ErrNoRows if needed
	// "github.com/jackc/pgx/v5"
)

type PgxAccountRepository struct {
	BaseRepository
}

// newPgxAccountRepository creates a new repository for account data.
func newPgxAccountRepository(pool *pgxpool.Pool) portsrepo.AccountRepositoryWithTx {
	return &PgxAccountRepository{
		BaseRepository: BaseRepository{Pool: pool},
	}
}

// Ensure PgxAccountRepository implements portsrepo.AccountRepositoryWithTx
var _ portsrepo.AccountRepositoryWithTx = (*PgxAccountRepository)(nil)

// SaveAccount inserts a new account.
// Note: Update/Inactivate logic will be added in later milestones/methods.
func (r *PgxAccountRepository) SaveAccount(ctx context.Context, account domain.Account) error {
	modelAcc := mapping.ToModelAccount(account)

	query := `
		INSERT INTO accounts (
			account_id, workplace_id, cfid, name, account_type, 
			currency_code, parent_account_id, description, is_active, 
			created_at, created_by, last_updated_at, last_updated_by, balance
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14);
	`
	// Use sql.NullString for potentially NULL parent_account_id and cfid
	var parentID sql.NullString
	if modelAcc.ParentAccountID != "" {
		parentID.String = modelAcc.ParentAccountID
		parentID.Valid = true
	}

	var cfid sql.NullString
	if modelAcc.CFID != "" {
		cfid.String = modelAcc.CFID
		cfid.Valid = true
	}

	_, err := r.Pool.Exec(ctx, query,
		modelAcc.AccountID,
		modelAcc.WorkplaceID,
		cfid,
		modelAcc.Name,
		modelAcc.AccountType,
		modelAcc.CurrencyCode,
		parentID,
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
				return apperrors.NewConflictError("account with ID "+modelAcc.AccountID+" already exists")
			}
		}
		return apperrors.NewAppError(500, "failed to save account "+modelAcc.AccountID, err)
	}
	return nil
}

// FindAccountByCFID retrieves an account by its CFID (Customer Facing ID) and workplace ID.
func (r *PgxAccountRepository) FindAccountByCFID(ctx context.Context, cfid string, workplaceID string) (*domain.Account, error) {
	if cfid == "" {
		return nil, apperrors.NewBadRequestError("CFID cannot be empty")
	}

	query := `
		SELECT 
			account_id, workplace_id, cfid, name, account_type, 
			currency_code, parent_account_id, description, is_active, 
			created_at, created_by, last_updated_at, last_updated_by, balance
		FROM accounts
		WHERE cfid = $1 AND workplace_id = $2;
	`

	var modelAcc models.Account
	var parentID sql.NullString // For potentially NULL parent_account_id
	var cfidVal sql.NullString  // For potentially NULL cfid
	var balance decimal.Decimal

	err := r.Pool.QueryRow(ctx, query, cfid, workplaceID).Scan(
		&modelAcc.AccountID,
		&modelAcc.WorkplaceID,
		&cfidVal,
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
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NewNotFoundError("account with CFID "+cfid+" not found in workplace "+workplaceID)
		}
		return nil, apperrors.NewAppError(500, "failed to query account by CFID", err)
	}

	if parentID.Valid {
		modelAcc.ParentAccountID = parentID.String
	} else {
		modelAcc.ParentAccountID = ""
	}

	if cfidVal.Valid {
		modelAcc.CFID = cfidVal.String
	} else {
		modelAcc.CFID = ""
	}

	modelAcc.Balance = balance

	// Convert to domain model
	account := mapping.ToDomainAccount(modelAcc)
	return &account, nil
}

// FindAccountByID retrieves an account by its ID.
func (r *PgxAccountRepository) FindAccountByID(ctx context.Context, accountID string) (*domain.Account, error) {
	query := `
		SELECT 
			account_id, workplace_id, cfid, name, account_type, 
			currency_code, parent_account_id, description, is_active, 
			created_at, created_by, last_updated_at, last_updated_by, balance
		FROM accounts
		WHERE account_id = $1;
	`
	var modelAcc models.Account
	var parentID sql.NullString // For potentially NULL parent_account_id
	var cfid sql.NullString     // For potentially NULL cfid
	var balance decimal.Decimal

	err := r.Pool.QueryRow(ctx, query, accountID).Scan(
		&modelAcc.AccountID,
		&modelAcc.WorkplaceID,
		&cfid, // Scan into sql.NullString
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
			return nil, apperrors.NewNotFoundError("account with ID "+accountID+" not found")
		}
		return nil, apperrors.NewAppError(500, "failed to find account by ID", err)
	}

	if parentID.Valid {
		modelAcc.ParentAccountID = parentID.String
	} else {
		modelAcc.ParentAccountID = ""
	}

	if cfid.Valid {
		modelAcc.CFID = cfid.String
	} else {
		modelAcc.CFID = ""
	}

	modelAcc.Balance = balance
	domainAcc := mapping.ToDomainAccount(modelAcc)
	return &domainAcc, nil
}

// FindAccountsByIDs retrieves multiple accounts by their IDs.
func (r *PgxAccountRepository) FindAccountsByIDs(ctx context.Context, accountIDs []string) (map[string]domain.Account, error) {
	if len(accountIDs) == 0 {
		return map[string]domain.Account{}, nil
	}

	query := `
		SELECT 
			account_id, workplace_id, cfid, name, account_type, 
			currency_code, parent_account_id, description, is_active, 
			created_at, created_by, last_updated_at, last_updated_by, balance
		FROM accounts
		WHERE account_id = ANY($1);
	`

	rows, err := r.Pool.Query(ctx, query, accountIDs)
	if err != nil {
		return nil, apperrors.NewAppError(500, "failed to query accounts by IDs", err)
	}
	defer rows.Close()

	accountsMap := make(map[string]domain.Account)
	for rows.Next() {
		var modelAcc models.Account
		var parentID sql.NullString // For potentially NULL parent_account_id
		var cfid sql.NullString     // For potentially NULL cfid
		var balance decimal.Decimal

		err := rows.Scan(
			&modelAcc.AccountID,
			&modelAcc.WorkplaceID,
			&cfid, // Scan into sql.NullString
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
			return nil, apperrors.NewAppError(500, "failed to scan account row", err)
		}

		if parentID.Valid {
			modelAcc.ParentAccountID = parentID.String
		} else {
			modelAcc.ParentAccountID = ""
		}

		modelAcc.Balance = balance
		accountsMap[modelAcc.AccountID] = mapping.ToDomainAccount(modelAcc)
	}

	if err := rows.Err(); err != nil {
		return nil, apperrors.NewAppError(500, "error iterating account rows", err)
	}

	return accountsMap, nil
}

// ListAccounts retrieves a paginated list of accounts for a workplace.
func (r *PgxAccountRepository) ListAccounts(ctx context.Context, workplaceID string, limit int, offset int) ([]domain.Account, error) {
	// Default limit handling
	if limit <= 0 {
		limit = 20 // Default limit
	}
	if offset < 0 {
		offset = 0 // Default offset
	}

	query := `
		SELECT 
			account_id, workplace_id, cfid, name, account_type, 
			currency_code, parent_account_id, description, is_active, 
			created_at, created_by, last_updated_at, last_updated_by, balance
		FROM accounts
		WHERE workplace_id = $1
		ORDER BY name
		LIMIT $2 OFFSET $3;
	`

	rows, err := r.Pool.Query(ctx, query, workplaceID, limit, offset)
	if err != nil {
		return nil, apperrors.NewAppError(500, "failed to query accounts for workplace", err)
	}
	defer rows.Close()

	var accounts []models.Account
	for rows.Next() {
		var modelAcc models.Account
		var parentID sql.NullString
		var cfid sql.NullString
		var balance decimal.Decimal

		err := rows.Scan(
			&modelAcc.AccountID,
			&modelAcc.WorkplaceID,
			&cfid,
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
			return nil, apperrors.NewAppError(500, "failed to scan account row", err)
		}

		if parentID.Valid {
			modelAcc.ParentAccountID = parentID.String
		} else {
			modelAcc.ParentAccountID = ""
		}

		if cfid.Valid {
			modelAcc.CFID = cfid.String
		} else {
			modelAcc.CFID = ""
		}

		modelAcc.Balance = balance
		accounts = append(accounts, modelAcc)
	}

	if err := rows.Err(); err != nil {
		return nil, apperrors.NewAppError(500, "error iterating account rows", err)
	}

	return mapping.ToDomainAccountSlice(accounts), nil
}

// UpdateAccount updates an existing account.
func (r *PgxAccountRepository) UpdateAccount(ctx context.Context, account domain.Account) error {
	modelAcc := mapping.ToModelAccount(account)

	// Use sql.NullString for potentially NULL parent_account_id and cfid
	var parentID sql.NullString
	if modelAcc.ParentAccountID != "" {
		parentID = sql.NullString{String: modelAcc.ParentAccountID, Valid: true}
	}

	var cfid sql.NullString
	if modelAcc.CFID != "" {
		cfid = sql.NullString{String: modelAcc.CFID, Valid: true}
	}

	query := `
		UPDATE accounts
		SET name = $2,
			description = $3,
			parent_account_id = $4,
			cfid = $5,
			is_active = $6,
			last_updated_at = $7,
			last_updated_by = $8
		WHERE account_id = $1;
	`

	result, err := r.Pool.Exec(ctx, query,
		modelAcc.AccountID,
		modelAcc.Name,
		modelAcc.Description,
		parentID,
		cfid,
		modelAcc.IsActive,
		modelAcc.LastUpdatedAt,
		modelAcc.LastUpdatedBy,
	)
	if err != nil {
		return apperrors.NewAppError(500, "failed to update account", err)
	}

	if result.RowsAffected() == 0 {
		return apperrors.NewNotFoundError("account not found")
	}

	return nil
}

// DeactivateAccount marks an account as inactive.
func (r *PgxAccountRepository) DeactivateAccount(ctx context.Context, accountID string, userID string, now time.Time) error {
	query := `
		UPDATE accounts
		SET is_active = false,
			last_updated_at = $1,
			last_updated_by = $2
		WHERE account_id = $3 AND is_active = true;
	`

	result, err := r.Pool.Exec(ctx, query, now, userID, accountID)
	if err != nil {
		return apperrors.NewAppError(500, "failed to deactivate account", err)
	}

	if result.RowsAffected() == 0 {
		// Check if the account exists at all
		existsQuery := `SELECT EXISTS(SELECT 1 FROM accounts WHERE account_id = $1);`
		var exists bool
		err := r.Pool.QueryRow(ctx, existsQuery, accountID).Scan(&exists)
		if err != nil {
			return apperrors.NewAppError(500, "failed to check if account exists", err)
		}

		if !exists {
			return apperrors.NewNotFoundError("account not found")
		}
		// Account exists but is already inactive - considered successful
		return nil // no error - idempotent operation
	}

	return nil
}

// FindAccountsByIDsForUpdate locks and retrieves accounts for update within a transaction.
func (r *PgxAccountRepository) FindAccountsByIDsForUpdate(ctx context.Context, tx pgx.Tx, accountIDs []string) (map[string]domain.Account, error) {
	if len(accountIDs) == 0 {
		return map[string]domain.Account{}, nil
	}

	query := `
		SELECT 
			account_id, workplace_id, cfid, name, account_type, 
			currency_code, parent_account_id, description, is_active, 
			created_at, created_by, last_updated_at, last_updated_by, balance
		FROM accounts
		WHERE account_id = ANY($1)
		FOR UPDATE; -- Lock rows for update
	`

	// Using the transaction provided by the caller
	rows, err := tx.Query(ctx, query, accountIDs)
	if err != nil {
		return nil, apperrors.NewAppError(500, "failed to query accounts by IDs for update", err)
	}
	defer rows.Close()

	accountsMap := make(map[string]domain.Account)
	foundIDs := make(map[string]struct{})
	for rows.Next() {
		var modelAcc models.Account
		var parentID sql.NullString // For potentially NULL parent_account_id
		var cfid sql.NullString     // For potentially NULL cfid
		var balance decimal.Decimal
		err := rows.Scan(
			&modelAcc.AccountID,
			&modelAcc.WorkplaceID,
			&cfid, // Scan into sql.NullString
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
			return nil, apperrors.NewAppError(500, "failed to scan account row", err)
		}

		if parentID.Valid {
			modelAcc.ParentAccountID = parentID.String
		} else {
			modelAcc.ParentAccountID = ""
		}

		if cfid.Valid {
			modelAcc.CFID = cfid.String
		} else {
			modelAcc.CFID = ""
		}

		modelAcc.Balance = balance
		accountsMap[modelAcc.AccountID] = mapping.ToDomainAccount(modelAcc)
		foundIDs[modelAcc.AccountID] = struct{}{}
	}

	if err := rows.Err(); err != nil {
		return nil, apperrors.NewAppError(500, "error iterating account rows", err)
	}

	// Check if all requested accounts were found
	if len(foundIDs) != len(accountIDs) {
		// Find which IDs are missing
		var missingIDs []string
		for _, id := range accountIDs {
			if _, found := foundIDs[id]; !found {
				missingIDs = append(missingIDs, id)
			}
		}
		return nil, apperrors.NewNotFoundError("accounts not found: "+fmt.Sprint(missingIDs))
	}

	return accountsMap, nil
}

// UpdateAccountBalancesInTx updates the balance for multiple accounts within a transaction.
func (r *PgxAccountRepository) UpdateAccountBalancesInTx(ctx context.Context, tx pgx.Tx, balanceChanges map[string]decimal.Decimal, userID string, now time.Time) error {
	if len(balanceChanges) == 0 {
		return nil // Nothing to update
	}

	// Prepare a statement for better performance with multiple updates
	statement, err := tx.Prepare(ctx, "update_balance", `
		UPDATE accounts
		SET balance = balance + $1,
			last_updated_at = $2,
			last_updated_by = $3
		WHERE account_id = $4 AND is_active = true
		RETURNING balance;
	`)
	if err != nil {
		return apperrors.NewAppError(500, "failed to prepare statement for balance updates", err)
	}

	// Track any accounts that failed to update
	failedAccounts := make([]string, 0)

	// Process each account's balance change
	for accountID, change := range balanceChanges {
		// Skip accounts with zero change to avoid unnecessary updates
		if change.IsZero() {
			continue
		}

		// Execute the update statement for this account
		var newBalance decimal.Decimal
		err := tx.QueryRow(ctx, statement.Name, change, now, userID, accountID).Scan(&newBalance)

		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				// This means either the account doesn't exist or it's inactive
				failedAccounts = append(failedAccounts, accountID)
				slog.Error("Account not found or inactive during balance update",
					"account_id", accountID,
					"change", change.String())
				continue
			}
			return apperrors.NewAppError(500, "failed to update balance for account "+accountID, err)
		}

		// Log the successful update
		slog.Debug("Account balance updated",
			"account_id", accountID,
			"change", change.String(),
			"new_balance", newBalance.String())
	}

	// If any accounts failed to update, return an error
	if len(failedAccounts) > 0 {
		return apperrors.NewNotFoundError("one or more accounts not found or inactive: "+fmt.Sprint(failedAccounts))
	}

	return nil
}

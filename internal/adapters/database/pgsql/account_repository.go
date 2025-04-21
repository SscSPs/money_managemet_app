package pgsql

import (
	"context"
	"fmt"
	"time"

	"github.com/SscSPs/money_managemet_app/internal/core/ports"
	"github.com/SscSPs/money_managemet_app/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
	// Import pgx specifically for error handling like ErrNoRows if needed
	// "github.com/jackc/pgx/v5"
)

type accountRepository struct {
	pool *pgxpool.Pool
}

// NewAccountRepository creates a new repository for account data.
func NewAccountRepository(pool *pgxpool.Pool) ports.AccountRepository {
	return &accountRepository{pool: pool}
}

// SaveAccount inserts a new account.
// Note: Update/Inactivate logic will be added in later milestones/methods.
func (r *accountRepository) SaveAccount(ctx context.Context, account models.Account) error {
	// Use actual UserID when available
	creatorUserID := "SYSTEM_MVP"
	now := time.Now().UTC()

	query := `
		INSERT INTO accounts (account_id, name, account_type, currency_code, parent_account_id, description, is_active, created_at, created_by, last_updated_at, last_updated_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11);
	`
	// Handle potential nil parentAccountID if DB requires NULL explicitly
	var parentID *string
	if account.ParentAccountID != "" {
		parentID = &account.ParentAccountID
	}

	_, err := r.pool.Exec(ctx, query,
		account.AccountID, // Assuming ID is generated beforehand or use DB default
		account.Name,
		account.AccountType,
		account.CurrencyCode,
		parentID,
		account.Description,
		account.IsActive, // Should default to true
		now,              // created_at
		creatorUserID,    // created_by
		now,              // last_updated_at
		creatorUserID,    // last_updated_by
	)

	if err != nil {
		// TODO: Check for specific errors like unique constraint violation
		return fmt.Errorf("failed to save account %s: %w", account.AccountID, err)
	}
	return nil
}

// FindAccountByID retrieves an account by its ID.
func (r *accountRepository) FindAccountByID(ctx context.Context, accountID string) (*models.Account, error) {
	query := `
		SELECT account_id, name, account_type, currency_code, parent_account_id, description, is_active, created_at, created_by, last_updated_at, last_updated_by
		FROM accounts
		WHERE account_id = $1;
	`
	var acc models.Account
	// Need to handle potential NULL parent_account_id from DB
	var parentID *string

	err := r.pool.QueryRow(ctx, query, accountID).Scan(
		&acc.AccountID,
		&acc.Name,
		&acc.AccountType,
		&acc.CurrencyCode,
		&parentID, // Scan into nullable pointer
		&acc.Description,
		&acc.IsActive,
		&acc.CreatedAt,
		&acc.CreatedBy,
		&acc.LastUpdatedAt,
		&acc.LastUpdatedBy,
	)

	if parentID != nil {
		acc.ParentAccountID = *parentID // Assign if not NULL
	} else {
		acc.ParentAccountID = "" // Or keep as empty string
	}

	if err != nil {
		// TODO: Handle pgx.ErrNoRows specifically if needed
		return nil, fmt.Errorf("failed to find account by ID %s: %w", accountID, err)
	}
	return &acc, nil
}

// TODO: Implement methods for UpdateAccount (for editing/inactivating) and ListAccounts in M2.

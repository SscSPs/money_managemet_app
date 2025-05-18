package repositories

import (
	"context"
	"time"

	"github.com/SscSPs/money_managemet_app/internal/core/domain"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
)

// AccountReader defines read operations for account data
type AccountReader interface {
	// FindAccountByID retrieves a specific account by its unique identifier.
	FindAccountByID(ctx context.Context, accountID string) (*domain.Account, error)

	// FindAccountByCFID retrieves an account by its CFID (Customer Facing ID) and workplace ID
	FindAccountByCFID(ctx context.Context, cfid string, workplaceID string) (*domain.Account, error)

	// FindAccountsByIDs retrieves multiple accounts by their IDs.
	FindAccountsByIDs(ctx context.Context, accountIDs []string) (map[string]domain.Account, error)

	// ListAccounts retrieves a paginated list of accounts for a given workplace.
	ListAccounts(ctx context.Context, workplaceID string, limit int, offset int) ([]domain.Account, error)
}

// AccountWriter defines write operations for account data
type AccountWriter interface {
	// SaveAccount persists a new account.
	SaveAccount(ctx context.Context, account domain.Account) error

	// UpdateAccount updates an existing account's details.
	UpdateAccount(ctx context.Context, account domain.Account) error

	// DeactivateAccount marks an account as inactive.
	DeactivateAccount(ctx context.Context, accountID string, userID string, now time.Time) error
}

// AccountTransactionSupport defines operations that support account transactions
type AccountTransactionSupport interface {
	// FindAccountsByIDsForUpdate selects accounts and locks them for update within a transaction.
	FindAccountsByIDsForUpdate(ctx context.Context, tx pgx.Tx, accountIDs []string) (map[string]domain.Account, error)

	// UpdateAccountBalancesInTx updates the balance for multiple accounts within a given transaction.
	UpdateAccountBalancesInTx(ctx context.Context, tx pgx.Tx, balanceChanges map[string]decimal.Decimal, userID string, now time.Time) error
}

// AccountRepositoryFacade combines all account-related repository interfaces
// This is a facade for clients that need access to all operations
type AccountRepositoryFacade interface {
	AccountReader
	AccountWriter
	AccountTransactionSupport
}

// AccountRepositoryWithTx extends AccountRepositoryFacade with transaction capabilities
type AccountRepositoryWithTx interface {
	AccountRepositoryFacade
	TransactionManager
}

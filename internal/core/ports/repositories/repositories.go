package repositories

import (
	"context"
	"time"

	"github.com/SscSPs/money_managemet_app/internal/core/domain"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
)

// RepositoryProvider holds all repository interfaces needed by services.
// This makes passing dependencies to the service container constructor cleaner.
type RepositoryProvider struct {
	AccountRepo      AccountRepository
	CurrencyRepo     CurrencyRepository
	ExchangeRateRepo ExchangeRateRepository
	UserRepo         UserRepository
	JournalRepo      JournalRepository
	WorkplaceRepo    WorkplaceRepository
}

// Note: Specific method signatures might evolve. Context is included for potential cancellation/timeouts.

// AccountRepository defines the persistence operations for Accounts.
type AccountRepository interface {
	SaveAccount(ctx context.Context, account domain.Account) error
	FindAccountByID(ctx context.Context, accountID string) (*domain.Account, error)
	FindAccountsByIDs(ctx context.Context, accountIDs []string) (map[string]domain.Account, error)
	// FindAccountsByIDsForUpdate selects accounts and locks them for update within a transaction.
	// Requires the transaction (tx) as an argument.
	FindAccountsByIDsForUpdate(ctx context.Context, tx pgx.Tx, accountIDs []string) (map[string]domain.Account, error)
	ListAccounts(ctx context.Context, workplaceID string, limit int, offset int) ([]domain.Account, error)
	UpdateAccount(ctx context.Context, account domain.Account) error
	// UpdateAccountBalancesInTx updates the balance for multiple accounts within a given transaction.
	// It expects a map of accountID to the *change* in balance (delta).
	UpdateAccountBalancesInTx(ctx context.Context, tx pgx.Tx, balanceChanges map[string]decimal.Decimal, userID string, now time.Time) error
	DeactivateAccount(ctx context.Context, accountID string, userID string, now time.Time) error
}

// JournalRepository defines the persistence operations for Journals and their Transactions.
// Saving a Journal implies saving its associated Transactions atomically.
type JournalRepository interface {
	// SaveJournal now requires balanceChanges map[accountID]delta
	SaveJournal(ctx context.Context, journal domain.Journal, transactions []domain.Transaction, balanceChanges map[string]decimal.Decimal) error
	FindJournalByID(ctx context.Context, journalID string) (*domain.Journal, error)
	FindTransactionsByJournalID(ctx context.Context, journalID string) ([]domain.Transaction, error)
	FindTransactionsByAccountID(ctx context.Context, workplaceID, accountID string) ([]domain.Transaction, error)
	ListJournalsByWorkplace(ctx context.Context, workplaceID string, limit int, offset int) ([]domain.Journal, error)
	FindTransactionsByJournalIDs(ctx context.Context, journalIDs []string) (map[string][]domain.Transaction, error)
	// UpdateJournalStatus(ctx context.Context, journalID string, status domain.JournalStatus) error
}

// CurrencyRepository defines persistence operations for Currencies.
type CurrencyRepository interface {
	SaveCurrency(ctx context.Context, currency domain.Currency) error                      // Use domain.Currency
	FindCurrencyByCode(ctx context.Context, currencyCode string) (*domain.Currency, error) // Use domain.Currency
	ListCurrencies(ctx context.Context) ([]domain.Currency, error)                         // Use domain.Currency
}

// ExchangeRateRepository defines persistence operations for ExchangeRates.
type ExchangeRateRepository interface {
	SaveExchangeRate(ctx context.Context, rate domain.ExchangeRate) error                                        // Use domain.ExchangeRate
	FindExchangeRate(ctx context.Context, fromCurrencyCode, toCurrencyCode string) (*domain.ExchangeRate, error) // Use domain.ExchangeRate
	// ListExchangeRates... ?
}

// UserRepository defines persistence operations for Users (needed for CreatedBy/UpdatedBy).
type UserRepository interface {
	SaveUser(ctx context.Context, user domain.User) error
	FindUserByID(ctx context.Context, userID string) (*domain.User, error)
	// Added methods for List, Update, Delete
	FindUsers(ctx context.Context, limit int, offset int) ([]domain.User, error)
	UpdateUser(ctx context.Context, user domain.User) error
	MarkUserDeleted(ctx context.Context, userID string, deletedAt time.Time, deletedBy string) error // Using soft delete
}

// WorkplaceRepository defines persistence operations for Workplaces and UserWorkplace memberships.
type WorkplaceRepository interface {
	SaveWorkplace(ctx context.Context, workplace domain.Workplace) error
	FindWorkplaceByID(ctx context.Context, workplaceID string) (*domain.Workplace, error)
	AddUserToWorkplace(ctx context.Context, membership domain.UserWorkplace) error
	FindUserWorkplaceRole(ctx context.Context, userID, workplaceID string) (*domain.UserWorkplace, error) // Returns membership details including role
	ListWorkplacesByUserID(ctx context.Context, userID string) ([]domain.Workplace, error)                // List workplaces a user belongs to
	// Potentially add methods like: FindUsersByWorkplaceID, RemoveUserFromWorkplace, UpdateUserRoleInWorkplace
}

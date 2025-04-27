package repositories

import (
	"context"
	"time"

	"github.com/SscSPs/money_managemet_app/internal/core/domain"
)

// Note: Specific method signatures might evolve. Context is included for potential cancellation/timeouts.

// AccountRepository defines the persistence operations for Accounts.
type AccountRepository interface {
	SaveAccount(ctx context.Context, account domain.Account) error
	FindAccountByID(ctx context.Context, accountID string) (*domain.Account, error)
	FindAccountsByIDs(ctx context.Context, accountIDs []string) (map[string]domain.Account, error)
	ListAccounts(ctx context.Context, limit int, offset int) ([]domain.Account, error)
	UpdateAccount(ctx context.Context, account domain.Account) error
	// Add methods for updating (inactivation), listing etc. in later milestones
	// ListAccounts(ctx context.Context) ([]domain.Account, error)
	// UpdateAccount(ctx context.Context, account domain.Account) error
	// DeactivateAccount(ctx context.Context, accountID string, userID string) error // Consider deactivate vs delete
	DeactivateAccount(ctx context.Context, accountID string, userID string, now time.Time) error
}

// JournalRepository defines the persistence operations for Journals and their Transactions.
// Saving a Journal implies saving its associated Transactions atomically.
type JournalRepository interface {
	SaveJournal(ctx context.Context, journal domain.Journal, transactions []domain.Transaction) error
	FindJournalByID(ctx context.Context, journalID string) (*domain.Journal, error)
	FindTransactionsByJournalID(ctx context.Context, journalID string) ([]domain.Transaction, error)
	FindTransactionsByAccountID(ctx context.Context, accountID string) ([]domain.Transaction, error)
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

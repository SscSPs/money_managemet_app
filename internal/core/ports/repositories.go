package ports

import (
	"context"

	"github.com/SscSPs/money_managemet_app/internal/models"
)

// Note: Specific method signatures might evolve. Context is included for potential cancellation/timeouts.

// AccountRepository defines the persistence operations for Accounts.
type AccountRepository interface {
	SaveAccount(ctx context.Context, account models.Account) error
	FindAccountByID(ctx context.Context, accountID string) (*models.Account, error)
	// Add methods for updating (inactivation), listing etc. in later milestones
	// ListAccounts(ctx context.Context) ([]models.Account, error)
	// UpdateAccount(ctx context.Context, account models.Account) error
}

// JournalRepository defines the persistence operations for Journals and their Transactions.
// Saving a Journal implies saving its associated Transactions atomically.
type JournalRepository interface {
	SaveJournal(ctx context.Context, journal models.Journal, transactions []models.Transaction) error
	FindJournalByID(ctx context.Context, journalID string) (*models.Journal, error)
	FindTransactionsByJournalID(ctx context.Context, journalID string) ([]models.Transaction, error)
	// UpdateJournalStatus(ctx context.Context, journalID string, status models.JournalStatus) error // Needed for Reversals in M4
}

// CurrencyRepository defines persistence operations for Currencies.
type CurrencyRepository interface {
	SaveCurrency(ctx context.Context, currency models.Currency) error // Primarily for initial setup
	FindCurrencyByCode(ctx context.Context, currencyCode string) (*models.Currency, error)
	ListCurrencies(ctx context.Context) ([]models.Currency, error)
}

// ExchangeRateRepository defines persistence operations for ExchangeRates.
type ExchangeRateRepository interface {
	SaveExchangeRate(ctx context.Context, rate models.ExchangeRate) error                                        // Primarily for initial setup
	FindExchangeRate(ctx context.Context, fromCurrencyCode, toCurrencyCode string) (*models.ExchangeRate, error) // Find latest effective?
	// ListExchangeRates... ?
}

// UserRepository defines persistence operations for Users (needed for CreatedBy/UpdatedBy).
type UserRepository interface {
	SaveUser(ctx context.Context, user models.User) error
	FindUserByID(ctx context.Context, userID string) (*models.User, error)
	// SaveUser might be needed if users are managed within the app later
}

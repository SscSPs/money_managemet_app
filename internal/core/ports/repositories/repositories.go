package repositories

import (
	"context"
	"time"

	"github.com/SscSPs/money_managemet_app/internal/core/domain"
)

// RepositoryProvider holds all repository interfaces needed by services.
// This makes passing dependencies to the service container constructor cleaner.
type RepositoryProvider struct {
	AccountRepo      AccountRepositoryFacade
	CurrencyRepo     CurrencyRepository
	ExchangeRateRepo ExchangeRateRepository
	UserRepo         UserRepository
	JournalRepo      JournalRepositoryFacade
	WorkplaceRepo    WorkplaceRepository
}

// Note: Specific method signatures might evolve. Context is included for potential cancellation/timeouts.

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

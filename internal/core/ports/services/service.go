package services

import (
	"context"

	"github.com/SscSPs/money_managemet_app/internal/core/domain"
	"github.com/SscSPs/money_managemet_app/internal/dto"
	"github.com/shopspring/decimal"
)

// AccountService defines the interface for account-related business logic.
type AccountService interface {
	CreateAccount(ctx context.Context, workplaceID string, req dto.CreateAccountRequest, userID string) (*domain.Account, error)
	GetAccountByID(ctx context.Context, workplaceID string, accountID string) (*domain.Account, error)
	ListAccounts(ctx context.Context, workplaceID string, limit int, offset int) ([]domain.Account, error)
	UpdateAccount(ctx context.Context, workplaceID string, accountID string, req dto.UpdateAccountRequest, userID string) (*domain.Account, error)
	DeactivateAccount(ctx context.Context, workplaceID string, accountID string, userID string) error
	CalculateAccountBalance(ctx context.Context, workplaceID string, accountID string) (decimal.Decimal, error) // Assuming balance calculation stays
}

// JournalService defines the interface for journal-related business logic.
type JournalService interface {
	// Replaces PersistJournal
	CreateJournal(ctx context.Context, workplaceID string, req dto.CreateJournalRequest, creatorUserID string) (*domain.Journal, error)
	// Replaces GetJournalWithTransactions (just returns journal now)
	GetJournalByID(ctx context.Context, workplaceID string, journalID string, requestingUserID string) (*domain.Journal, error)
	// New method for listing journals in a workplace
	ListJournals(ctx context.Context, workplaceID string, limit int, offset int, requestingUserID string) ([]domain.Journal, error)
	// New method for updating journal details (excluding transactions)
	UpdateJournal(ctx context.Context, workplaceID string, journalID string, req dto.UpdateJournalRequest, requestingUserID string) (*domain.Journal, error)
	// New method for deactivating (soft deleting) a journal
	DeactivateJournal(ctx context.Context, workplaceID string, journalID string, requestingUserID string) error
	// New method for listing transactions for a specific account
	ListTransactionsByAccount(ctx context.Context, workplaceID string, accountID string, limit int, offset int, requestingUserID string) ([]domain.Transaction, error)
	// New method for calculating account balance
	CalculateAccountBalance(ctx context.Context, workplaceID string, accountID string) (decimal.Decimal, error)
}

// WorkplaceService defines the interface for workplace and membership logic.
type WorkplaceService interface {
	CreateWorkplace(ctx context.Context, name, description, creatorUserID string) (*domain.Workplace, error)
	AddUserToWorkplace(ctx context.Context, addingUserID, targetUserID, workplaceID string, role domain.UserWorkplaceRole) error
	ListUserWorkplaces(ctx context.Context, userID string) ([]domain.Workplace, error)
	AuthorizeUserAction(ctx context.Context, userID, workplaceID string, requiredRole domain.UserWorkplaceRole) error
	FindWorkplaceByID(ctx context.Context, workplaceID string) (*domain.Workplace, error) // Added for potential checks
}

// UserService defines the interface for user management logic.
type UserService interface {
	CreateUser(ctx context.Context, req dto.CreateUserRequest) (*domain.User, error)
	GetUserByID(ctx context.Context, userID string) (*domain.User, error)
	ListUsers(ctx context.Context, limit, offset int) ([]domain.User, error)
	UpdateUser(ctx context.Context, userID string, req dto.UpdateUserRequest, requestingUserID string) (*domain.User, error)
	DeleteUser(ctx context.Context, userID string, requestingUserID string) error
	AuthenticateUser(ctx context.Context, email, password string) (*domain.User, error)
}

// CurrencyService defines the interface for currency and exchange rate logic.
type CurrencyService interface {
	CreateCurrency(ctx context.Context, req dto.CreateCurrencyRequest, creatorUserID string) (*domain.Currency, error)
	GetCurrencyByCode(ctx context.Context, currencyCode string) (*domain.Currency, error)
	ListCurrencies(ctx context.Context) ([]domain.Currency, error)
	// Add methods for Exchange Rates if needed
}

// ExchangeRateService defines the interface for exchange rate logic.
type ExchangeRateService interface {
	CreateExchangeRate(ctx context.Context, req dto.CreateExchangeRateRequest, creatorUserID string) (*domain.ExchangeRate, error)
	GetExchangeRate(ctx context.Context, fromCode, toCode string) (*domain.ExchangeRate, error)
	// Add List method if needed later
}

// StaticDataService defines the interface for managing static data like currencies.
type StaticDataService interface {
	InitializeStaticData(ctx context.Context) error
}

// TransactionService might be needed later for more complex transaction queries/operations
// type TransactionService interface {
// 	 FindTransactionsByAccount(ctx context.Context, workplaceID, accountID string, params dto.TransactionListParams) ([]domain.Transaction, error)
// 	 FindTransactionsByJournal(ctx context.Context, workplaceID, journalID string) ([]domain.Transaction, error)
// }

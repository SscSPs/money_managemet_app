package services

import (
	"context"

	"github.com/SscSPs/money_managemet_app/internal/core/domain"
	"github.com/SscSPs/money_managemet_app/internal/dto"
	"github.com/shopspring/decimal"
)

// ServiceContainer holds instances of all the application services.
type ServiceContainer struct {
	Account      AccountSvcFacade
	Currency     CurrencySvcFacade
	ExchangeRate ExchangeRateSvcFacade
	User         UserSvcFacade
	Journal      JournalSvcFacade
	Workplace    WorkplaceSvcFacade
}

// Legacy interfaces below - these will be replaced by the segmented interfaces
// in account_services.go, journal_services.go, etc.

// AccountService defines the interface for account-related business logic.
type AccountService interface {
	CreateAccount(ctx context.Context, workplaceID string, req dto.CreateAccountRequest, userID string) (*domain.Account, error)
	GetAccountByID(ctx context.Context, workplaceID string, accountID string) (*domain.Account, error)
	GetAccountByIDs(ctx context.Context, workplaceID string, accountIDs []string) (map[string]domain.Account, error)
	ListAccounts(ctx context.Context, workplaceID string, limit int, offset int) ([]domain.Account, error)
	UpdateAccount(ctx context.Context, workplaceID string, accountID string, req dto.UpdateAccountRequest, userID string) (*domain.Account, error)
	DeactivateAccount(ctx context.Context, workplaceID string, accountID string, userID string) error
	CalculateAccountBalance(ctx context.Context, workplaceID string, accountID string) (decimal.Decimal, error)
}

// JournalService defines the interface for journal-related business logic.
type JournalService interface {
	CreateJournal(ctx context.Context, workplaceID string, req dto.CreateJournalRequest, creatorUserID string) (*domain.Journal, error)
	GetJournalByID(ctx context.Context, workplaceID string, journalID string, requestingUserID string) (*domain.Journal, error)
	ListJournals(ctx context.Context, workplaceID string, userID string, params dto.ListJournalsParams) (*dto.ListJournalsResponse, error)
	UpdateJournal(ctx context.Context, workplaceID string, journalID string, req dto.UpdateJournalRequest, requestingUserID string) (*domain.Journal, error)
	DeactivateJournal(ctx context.Context, workplaceID string, journalID string, requestingUserID string) error
	ListTransactionsByAccount(ctx context.Context, workplaceID string, accountID string, userID string, params dto.ListTransactionsParams) (*dto.ListTransactionsResponse, error)
	CalculateAccountBalance(ctx context.Context, workplaceID string, accountID string) (decimal.Decimal, error)
	ReverseJournal(ctx context.Context, workplaceID string, journalID string, userID string) (*domain.Journal, error)
}

// WorkplaceService defines the interface for workplace and membership logic.
type WorkplaceService interface {
	CreateWorkplace(ctx context.Context, name, description, defaultCurrencyCode, creatorUserID string) (*domain.Workplace, error)
	AddUserToWorkplace(ctx context.Context, addingUserID, targetUserID, workplaceID string, role domain.UserWorkplaceRole) error
	ListUserWorkplaces(ctx context.Context, userID string, includeDisabled bool) ([]domain.Workplace, error)
	AuthorizeUserAction(ctx context.Context, userID, workplaceID string, requiredRole domain.UserWorkplaceRole) error
	FindWorkplaceByID(ctx context.Context, workplaceID string) (*domain.Workplace, error)
	DeactivateWorkplace(ctx context.Context, workplaceID string, requestingUserID string) error
	ActivateWorkplace(ctx context.Context, workplaceID string, requestingUserID string) error
	ListWorkplaceUsers(ctx context.Context, workplaceID string, requestingUserID string) ([]domain.UserWorkplace, error)
	RemoveUserFromWorkplace(ctx context.Context, requestingUserID, targetUserID, workplaceID string) error
	UpdateUserWorkplaceRole(ctx context.Context, requestingUserID, targetUserID, workplaceID string, newRole domain.UserWorkplaceRole) error
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
}

// ExchangeRateService defines the interface for exchange rate logic.
type ExchangeRateService interface {
	CreateExchangeRate(ctx context.Context, req dto.CreateExchangeRateRequest, creatorUserID string) (*domain.ExchangeRate, error)
	GetExchangeRate(ctx context.Context, fromCode, toCode string) (*domain.ExchangeRate, error)
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

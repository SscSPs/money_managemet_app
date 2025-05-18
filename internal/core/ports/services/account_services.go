package services

import (
	"context"

	"github.com/SscSPs/money_managemet_app/internal/core/domain"
	"github.com/SscSPs/money_managemet_app/internal/dto"
	"github.com/shopspring/decimal"
)

// AccountReaderSvc defines read operations for account data
type AccountReaderSvc interface {
	// GetAccountByID retrieves a specific account by its unique identifier.
	GetAccountByID(ctx context.Context, workplaceID string, accountID string, userID string) (*domain.Account, error)

	// GetAccountByCFID retrieves an account by its CFID (Customer Facing ID) and workplace ID
	GetAccountByCFID(ctx context.Context, workplaceID string, cfid string, userID string) (*domain.Account, error)

	// GetAccountByIDs retrieves multiple accounts by their IDs.
	GetAccountByIDs(ctx context.Context, workplaceID string, accountIDs []string, userID string) (map[string]domain.Account, error)

	// ListAccounts retrieves a paginated list of accounts for a given workplace.
	ListAccounts(ctx context.Context, workplaceID string, limit int, offset int) ([]domain.Account, error)
}

// AccountWriterSvc defines write operations for account data
type AccountWriterSvc interface {
	// CreateAccount persists a new account.
	CreateAccount(ctx context.Context, workplaceID string, req dto.CreateAccountRequest, userID string) (*domain.Account, error)

	// UpdateAccount updates an existing account's details.
	UpdateAccount(ctx context.Context, workplaceID string, accountID string, req dto.UpdateAccountRequest, userID string) (*domain.Account, error)

	// DeactivateAccount marks an account as inactive.
	DeactivateAccount(ctx context.Context, workplaceID string, accountID string, userID string) error
}

// AccountCalculatorSvc defines calculation operations for account data
type AccountCalculatorSvc interface {
	// CalculateAccountBalance calculates the current balance of an account.
	CalculateAccountBalance(ctx context.Context, workplaceID string, accountID string, userID string) (decimal.Decimal, error)
}

// AccountSvcFacade combines all account-related service interfaces
// This is a facade for clients that need access to all operations
type AccountSvcFacade interface {
	AccountReaderSvc
	AccountWriterSvc
	AccountCalculatorSvc
}

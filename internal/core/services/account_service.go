package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/SscSPs/money_managemet_app/internal/apperrors"
	"github.com/SscSPs/money_managemet_app/internal/core/domain" // Use domain types
	portsrepo "github.com/SscSPs/money_managemet_app/internal/core/ports/repositories"
	"github.com/SscSPs/money_managemet_app/internal/dto"
	"github.com/SscSPs/money_managemet_app/internal/middleware" // Import middleware for GetLoggerFromCtx
	"github.com/google/uuid"                                    // For generating AccountID
)

type AccountService struct {
	AccountRepository portsrepo.AccountRepository
	// Potentially add CurrencyRepository if validation is needed
}

func NewAccountService(repo portsrepo.AccountRepository) *AccountService {
	return &AccountService{AccountRepository: repo}
}

func (s *AccountService) CreateAccount(ctx context.Context, req dto.CreateAccountRequest, userID string) (*domain.Account, error) {
	logger := middleware.GetLoggerFromCtx(ctx) // Get logger from context
	// Basic validation (currency existence, parent account existence) could be added here
	// For now, assume input is valid per DTO binding

	now := time.Now()
	newAccountID := uuid.NewString() // Generate a new UUID for the account

	parentID := ""
	if req.ParentAccountID != nil {
		parentID = *req.ParentAccountID
		// TODO: Validate parent account exists and is suitable (e.g., not the same account)
	}

	// Create domain.Account
	account := domain.Account{
		AccountID:       newAccountID,
		Name:            req.Name,
		AccountType:     domain.AccountType(req.AccountType), // Convert from model type if different, seems same here
		CurrencyCode:    req.CurrencyCode,
		ParentAccountID: parentID,
		Description:     req.Description,
		IsActive:        true, // Default to active on creation
		AuditFields: domain.AuditFields{
			CreatedAt:     now,
			CreatedBy:     userID,
			LastUpdatedAt: now,
			LastUpdatedBy: userID,
		},
	}

	err := s.AccountRepository.SaveAccount(ctx, account) // Pass domain.Account
	if err != nil {
		logger.Error("Failed to save account in repository", slog.String("error", err.Error()), slog.String("account_id", account.AccountID))
		// Propagate repository error (error handling improvements later)
		return nil, err
	}

	logger.Info("Account created successfully in service", slog.String("account_id", account.AccountID))
	return &account, nil
}

func (s *AccountService) GetAccountByID(ctx context.Context, accountID string) (*domain.Account, error) {
	logger := middleware.GetLoggerFromCtx(ctx)                          // Get logger from context
	account, err := s.AccountRepository.FindAccountByID(ctx, accountID) // Expect domain.Account
	if err != nil {
		// Log the error occurred during repository call
		// Note: Don't log if error is ErrNotFound, as it's an expected outcome
		if !errors.Is(err, apperrors.ErrNotFound) {
			logger.Error("Failed to find account by ID in repository", slog.String("error", err.Error()), slog.String("account_id", accountID))
		}
		// Propagate the error (including apperrors.ErrNotFound)
		return nil, err
	}
	logger.Debug("Account retrieved successfully from service", slog.String("account_id", account.AccountID))
	return account, nil
}

// ListAccounts retrieves a paginated list of active accounts.
func (s *AccountService) ListAccounts(ctx context.Context, limit int, offset int) ([]domain.Account, error) {
	logger := middleware.GetLoggerFromCtx(ctx)
	accounts, err := s.AccountRepository.ListAccounts(ctx, limit, offset)
	if err != nil {
		logger.Error("Failed to list accounts from repository", slog.String("error", err.Error()), slog.Int("limit", limit), slog.Int("offset", offset))
		return nil, fmt.Errorf("failed to list accounts: %w", err)
	}

	if accounts == nil {
		return []domain.Account{}, nil // Return empty slice if repo returns nil
	}

	logger.Debug("Accounts listed successfully from service", slog.Int("count", len(accounts)))
	return accounts, nil
}

// UpdateAccount updates specific fields of an existing account.
func (s *AccountService) UpdateAccount(ctx context.Context, accountID string, req dto.UpdateAccountRequest, userID string) (*domain.Account, error) {
	logger := middleware.GetLoggerFromCtx(ctx)

	// Fetch the existing account
	account, err := s.AccountRepository.FindAccountByID(ctx, accountID)
	if err != nil {
		// Propagate error (including ErrNotFound)
		if !errors.Is(err, apperrors.ErrNotFound) {
			logger.Error("Failed to find account by ID for update", slog.String("error", err.Error()), slog.String("account_id", accountID))
		}
		return nil, err
	}

	// TODO: Authorization Check - Does userID have permission to update this account?
	// (e.g., check if account.CreatedBy == userID, or based on roles/permissions)

	// Apply updates from the request DTO if fields are provided
	updated := false
	if req.Name != nil {
		account.Name = *req.Name
		updated = true
	}
	if req.Description != nil {
		account.Description = *req.Description
		updated = true
	}
	if req.IsActive != nil {
		account.IsActive = *req.IsActive
		updated = true
	}

	// If no fields were updated, just return the fetched account
	if !updated {
		logger.Debug("No fields provided for account update", slog.String("account_id", accountID))
		return account, nil
	}

	// Update audit fields
	now := time.Now()
	account.LastUpdatedAt = now
	account.LastUpdatedBy = userID

	// Persist changes using the newly added repository method
	err = s.AccountRepository.UpdateAccount(ctx, *account) // Correctly calling UpdateAccount
	if err != nil {
		logger.Error("Failed to update account in repository", slog.String("error", err.Error()), slog.String("account_id", accountID))
		return nil, err
	}

	logger.Info("Account updated successfully in service", slog.String("account_id", account.AccountID))
	return account, nil
}

// DeactivateAccount marks an account as inactive (soft delete).
// Method name reverted to match repository interface and original implementation.
func (s *AccountService) DeactivateAccount(ctx context.Context, accountID string, userID string) error {
	logger := middleware.GetLoggerFromCtx(ctx)

	// We could add checks here (e.g., check balance is zero before deactivating?)
	// Fetching the account first might be useful for checks, but the repo method handles not found.

	now := time.Now()
	err := s.AccountRepository.DeactivateAccount(ctx, accountID, userID, now) // Correctly calling DeactivateAccount
	if err != nil {
		if !errors.Is(err, apperrors.ErrNotFound) && !errors.Is(err, apperrors.ErrValidation) {
			// Log unexpected repository errors
			logger.Error("Failed to deactivate account in repository", slog.String("error", err.Error()), slog.String("account_id", accountID))
		}
		// Propagate known errors (NotFound, Validation[already inactive]) and unexpected ones
		return err
	}

	logger.Info("Account deactivated successfully in service", slog.String("account_id", accountID))
	return nil
}

/* // Removed the incorrect DeleteAccount implementation
// DeleteAccount marks an account as inactive (soft delete).
// Renamed from DeactivateAccount for consistency with handler.
func (s *AccountService) DeleteAccount(ctx context.Context, accountID string, userID string) error {
	...
}
*/

/* // Commenting out the old DeactivateAccount - now handled above
// DeactivateAccount marks an account as inactive.
func (s *AccountService) DeactivateAccount(ctx context.Context, accountID string, userID string) error {
	...
}
*/

// Remove the outdated TODO

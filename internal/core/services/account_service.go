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
	portssvc "github.com/SscSPs/money_managemet_app/internal/core/ports/services" // Added portssvc import
	"github.com/SscSPs/money_managemet_app/internal/dto"
	"github.com/SscSPs/money_managemet_app/internal/middleware" // Import middleware for GetLoggerFromCtx
	"github.com/google/uuid"                                    // For generating AccountID
	"github.com/shopspring/decimal"
)

type accountService struct {
	AccountRepository portsrepo.AccountRepository
	// Potentially add CurrencyRepository if validation is needed
}

func NewAccountService(repo portsrepo.AccountRepository) portssvc.AccountService { // Revert to concrete pointer type
	return &accountService{AccountRepository: repo}
}

func (s *accountService) CreateAccount(ctx context.Context, workplaceID string, req dto.CreateAccountRequest, userID string) (*domain.Account, error) {
	logger := middleware.GetLoggerFromCtx(ctx)

	// TODO: Authorization: Check if userID has permission to create accounts in workplaceID
	// Use AuthorizeUserAction(ctx, userID, workplaceID, domain.RoleMember) // Or RoleAdmin?

	now := time.Now()
	newAccountID := uuid.NewString()

	parentID := ""
	if req.ParentAccountID != nil {
		parentID = *req.ParentAccountID
		// TODO: Validate parent account exists AND belongs to the same workplaceID
	}

	// Create domain.Account, ensuring WorkplaceID is set
	account := domain.Account{
		AccountID:       newAccountID,
		WorkplaceID:     workplaceID, // Set from parameter
		Name:            req.Name,
		AccountType:     domain.AccountType(req.AccountType),
		CurrencyCode:    req.CurrencyCode,
		ParentAccountID: parentID,
		Description:     req.Description,
		IsActive:        true,
		AuditFields: domain.AuditFields{
			CreatedAt:     now,
			CreatedBy:     userID,
			LastUpdatedAt: now,
			LastUpdatedBy: userID,
		},
	}

	err := s.AccountRepository.SaveAccount(ctx, account)
	if err != nil {
		logger.Error("Failed to save account in repository", slog.String("error", err.Error()), slog.String("account_id", account.AccountID), slog.String("workplace_id", workplaceID))
		return nil, err
	}

	logger.Info("Account created successfully in service", slog.String("account_id", account.AccountID), slog.String("workplace_id", workplaceID))
	return &account, nil
}

func (s *accountService) GetAccountByID(ctx context.Context, workplaceID string, accountID string) (*domain.Account, error) {
	logger := middleware.GetLoggerFromCtx(ctx)
	account, err := s.AccountRepository.FindAccountByID(ctx, accountID)
	if err != nil {
		if !errors.Is(err, apperrors.ErrNotFound) {
			logger.Error("Failed to find account by ID in repository", slog.String("error", err.Error()), slog.String("account_id", accountID))
		}
		return nil, err // Propagate error (including NotFound)
	}

	// Authorization: Check if the fetched account belongs to the expected workplace
	if account.WorkplaceID != workplaceID {
		logger.Warn("Account found but belongs to different workplace", slog.String("account_id", accountID), slog.String("account_workplace", account.WorkplaceID), slog.String("requested_workplace", workplaceID))
		// Return NotFound to obscure existence from unauthorized workplaces
		return nil, apperrors.ErrNotFound
	}

	// TODO: Further auth check: Does the requesting user (from ctx) belong to this workplace?

	logger.Debug("Account retrieved successfully from service", slog.String("account_id", account.AccountID), slog.String("workplace_id", account.WorkplaceID))
	return account, nil
}

// ListAccounts retrieves a paginated list of active accounts for a specific workplace.
func (s *accountService) ListAccounts(ctx context.Context, workplaceID string, limit int, offset int) ([]domain.Account, error) {
	logger := middleware.GetLoggerFromCtx(ctx)

	// TODO: Authorization - Check if the user associated with the ctx has access to this workplaceID.

	accounts, err := s.AccountRepository.ListAccounts(ctx, workplaceID, limit, offset) // Pass workplaceID
	if err != nil {
		logger.Error("Failed to list accounts from repository", slog.String("error", err.Error()), slog.String("workplace_id", workplaceID), slog.Int("limit", limit), slog.Int("offset", offset))
		return nil, fmt.Errorf("failed to list accounts for workplace %s: %w", workplaceID, err)
	}

	if accounts == nil {
		return []domain.Account{}, nil // Return empty slice if repo returns nil
	}

	logger.Debug("Accounts listed successfully from service", slog.Int("count", len(accounts)), slog.String("workplace_id", workplaceID))
	return accounts, nil
}

// UpdateAccount updates specific fields of an existing account.
func (s *accountService) UpdateAccount(ctx context.Context, workplaceID string, accountID string, req dto.UpdateAccountRequest, userID string) (*domain.Account, error) {
	logger := middleware.GetLoggerFromCtx(ctx)

	// Fetch the existing account
	account, err := s.AccountRepository.FindAccountByID(ctx, accountID)
	if err != nil {
		if !errors.Is(err, apperrors.ErrNotFound) {
			logger.Error("Failed to find account by ID for update", slog.String("error", err.Error()), slog.String("account_id", accountID))
		}
		return nil, err
	}

	// Authorization: Check if the fetched account belongs to the expected workplace
	if account.WorkplaceID != workplaceID {
		logger.Warn("Attempt to update account from wrong workplace", slog.String("account_id", accountID), slog.String("account_workplace", account.WorkplaceID), slog.String("requested_workplace", workplaceID))
		return nil, apperrors.ErrNotFound // Treat as NotFound
	}

	// TODO: Authorization Check - Does userID have permission to update this account in this workplace?

	// Apply updates...
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
	if !updated {
		logger.Debug("No fields provided for account update", slog.String("account_id", accountID))
		return account, nil
	}

	// Update audit fields
	now := time.Now()
	account.LastUpdatedAt = now
	account.LastUpdatedBy = userID

	err = s.AccountRepository.UpdateAccount(ctx, *account)
	if err != nil {
		logger.Error("Failed to update account in repository", slog.String("error", err.Error()), slog.String("account_id", accountID))
		return nil, err
	}

	logger.Info("Account updated successfully in service", slog.String("account_id", account.AccountID), slog.String("workplace_id", account.WorkplaceID))
	return account, nil
}

// DeactivateAccount marks an account as inactive (soft delete).
func (s *accountService) DeactivateAccount(ctx context.Context, workplaceID string, accountID string, userID string) error {
	logger := middleware.GetLoggerFromCtx(ctx)

	// Fetch the existing account first to check workplace ownership
	account, err := s.AccountRepository.FindAccountByID(ctx, accountID)
	if err != nil {
		if !errors.Is(err, apperrors.ErrNotFound) {
			logger.Error("Failed to find account by ID for deactivate", slog.String("error", err.Error()), slog.String("account_id", accountID))
		}
		return err // Propagate NotFound or other errors
	}

	// Authorization: Check if the fetched account belongs to the expected workplace
	if account.WorkplaceID != workplaceID {
		logger.Warn("Attempt to deactivate account from wrong workplace", slog.String("account_id", accountID), slog.String("account_workplace", account.WorkplaceID), slog.String("requested_workplace", workplaceID))
		return apperrors.ErrNotFound // Treat as NotFound
	}

	// TODO: Authorization Check - Does userID have permission to deactivate accounts in this workplace?

	// Now call the repository method which handles already inactive state
	now := time.Now()
	err = s.AccountRepository.DeactivateAccount(ctx, accountID, userID, now)
	if err != nil {
		if !errors.Is(err, apperrors.ErrNotFound) && !errors.Is(err, apperrors.ErrValidation) {
			logger.Error("Failed to deactivate account in repository", slog.String("error", err.Error()), slog.String("account_id", accountID))
		}
		return err
	}

	logger.Info("Account deactivated successfully in service", slog.String("account_id", accountID), slog.String("workplace_id", workplaceID))
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

// CalculateAccountBalance calculates the current balance for a given account.
// Now that balance is persisted on the account, this primarily reads the value.
// The transaction-based calculation logic is kept commented for potential validation/reconciliation.
func (s *accountService) CalculateAccountBalance(ctx context.Context, workplaceID string, accountID string) (decimal.Decimal, error) {
	logger := middleware.GetLoggerFromCtx(ctx)

	// TODO: Authorization: Check user can access workplaceID/accountID (partially done by GetAccountByID)

	// Fetch the account, which now includes the persisted balance
	account, err := s.GetAccountByID(ctx, workplaceID, accountID)
	if err != nil {
		// GetAccountByID handles NotFound and logging
		return decimal.Zero, fmt.Errorf("failed to get account %s for balance calculation: %w", accountID, err)
	}

	logger.Debug("Retrieved persisted balance for account", slog.String("account_id", accountID), slog.String("balance", account.Balance.String()))
	return account.Balance, nil

	/* --- OLD CALCULATION LOGIC (Keep for reference/validation?) ---
	logger.Warn("CalculateAccountBalance currently returns zero - IMPLEMENTATION NEEDED", slog.String("account_id", accountID), slog.String("workplace_id", workplaceID))

	// 1. Authorization: Check user can access workplaceID (maybe done implicitly if called by other authorized services)
	// 2. Fetch account to verify it belongs to workplaceID (redundant if called after GetAccountByID?)
	// 3. Fetch relevant transactions for the accountID within the workplaceID.
	// 4. Sum transactions based on account type (debit/credit).
	// 5. Return balance.

	// --- TEMPORARY: Return zero and nil error --- \
	// Replace this with actual calculation logic
	return decimal.Zero, nil
	// --- /TEMPORARY ---
	*/
}

// Remove the outdated TODO

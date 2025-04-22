package services

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/SscSPs/money_managemet_app/internal/apperrors"
	"github.com/SscSPs/money_managemet_app/internal/core/ports"
	"github.com/SscSPs/money_managemet_app/internal/dto"
	"github.com/SscSPs/money_managemet_app/internal/middleware" // Import middleware for GetLoggerFromCtx
	"github.com/SscSPs/money_managemet_app/internal/models"
	"github.com/google/uuid" // For generating AccountID
)

type AccountService struct {
	accountRepo ports.AccountRepository
	// Potentially add CurrencyRepository if validation is needed
}

func NewAccountService(accountRepo ports.AccountRepository) *AccountService {
	return &AccountService{accountRepo: accountRepo}
}

func (s *AccountService) CreateAccount(ctx context.Context, req dto.CreateAccountRequest, userID string) (*models.Account, error) {
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

	account := models.Account{
		AccountID:       newAccountID,
		Name:            req.Name,
		AccountType:     req.AccountType,
		CurrencyCode:    req.CurrencyCode,
		ParentAccountID: parentID,
		Description:     req.Description,
		IsActive:        true, // Default to active on creation
		AuditFields: models.AuditFields{
			CreatedAt:     now,
			CreatedBy:     userID,
			LastUpdatedAt: now,
			LastUpdatedBy: userID,
		},
	}

	err := s.accountRepo.SaveAccount(ctx, account)
	if err != nil {
		logger.Error("Failed to save account in repository", slog.String("error", err.Error()), slog.String("account_id", account.AccountID))
		// Propagate repository error (error handling improvements later)
		return nil, err
	}

	logger.Info("Account created successfully in service", slog.String("account_id", account.AccountID))
	return &account, nil
}

func (s *AccountService) GetAccountByID(ctx context.Context, accountID string) (*models.Account, error) {
	logger := middleware.GetLoggerFromCtx(ctx) // Get logger from context
	account, err := s.accountRepo.FindAccountByID(ctx, accountID)
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

// TODO: Add ListAccounts, UpdateAccount, DeactivateAccount methods later

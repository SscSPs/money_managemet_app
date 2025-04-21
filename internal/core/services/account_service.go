package services

import (
	"context"
	"log"
	"time"

	"github.com/SscSPs/money_managemet_app/internal/core/ports"
	"github.com/SscSPs/money_managemet_app/internal/dto"
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
			CreatedBy:     userID, // TODO: Get actual user ID from context/auth
			LastUpdatedAt: now,
			LastUpdatedBy: userID, // TODO: Get actual user ID from context/auth
		},
	}

	err := s.accountRepo.SaveAccount(ctx, account)
	if err != nil {
		// TODO: Add structured logging
		log.Println("err, unable to save account", err)
		return nil, err // Propagate repository error
	}

	return &account, nil
}

func (s *AccountService) GetAccountByID(ctx context.Context, accountID string) (*models.Account, error) {
	account, err := s.accountRepo.FindAccountByID(ctx, accountID)
	if err != nil {
		// TODO: Handle specific errors like "not found" differently if needed
		// TODO: Add structured logging
		return nil, err
	}
	return account, nil
}

// TODO: Add ListAccounts, UpdateAccount, DeactivateAccount methods later

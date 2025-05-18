package services

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/SscSPs/money_managemet_app/internal/apperrors"
	"github.com/SscSPs/money_managemet_app/internal/core/domain"
	portsrepo "github.com/SscSPs/money_managemet_app/internal/core/ports/repositories"
	portssvc "github.com/SscSPs/money_managemet_app/internal/core/ports/services"
	"github.com/SscSPs/money_managemet_app/internal/dto"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// accountService implements the AccountSvcFacade interface
type accountService struct {
	BaseService
	accountRepo      portsrepo.AccountRepositoryFacade
	currencyRepo     portsrepo.CurrencyReader
	workplaceService portssvc.WorkplaceReaderSvc
}

// AccountServiceOption is a functional option for configuring the account service
type AccountServiceOption func(*accountService)

// WithWorkplaceService adds workplace service dependency
func WithWorkplaceService(svc portssvc.WorkplaceReaderSvc) AccountServiceOption {
	return func(s *accountService) {
		s.workplaceService = svc
	}
}

// WithWorkplaceAuthorizer adds workplace authorizer dependency
func WithWorkplaceAuthorizer(authorizer portssvc.WorkplaceAuthorizerSvc) AccountServiceOption {
	return func(s *accountService) {
		s.WorkplaceAuthorizer = authorizer
	}
}

// WithCurrencyRepository adds currency repository dependency
func WithCurrencyRepository(repo portsrepo.CurrencyReader) AccountServiceOption {
	return func(s *accountService) {
		s.currencyRepo = repo
	}
}

// NewAccountService creates a new account service with the provided options
func NewAccountService(repo portsrepo.AccountRepositoryFacade, options ...AccountServiceOption) portssvc.AccountSvcFacade {
	svc := &accountService{
		accountRepo: repo,
	}

	// Apply all options
	for _, option := range options {
		option(svc)
	}

	return svc
}

// Ensure accountService implements the AccountSvcFacade interface
var _ portssvc.AccountSvcFacade = (*accountService)(nil)

func (s *accountService) CreateAccount(ctx context.Context, workplaceID string, req dto.CreateAccountRequest, userID string) (*domain.Account, error) {
	// Authorize user action
	if err := s.AuthorizeUser(ctx, userID, workplaceID, domain.RoleMember); err != nil {
		s.LogError(ctx, err, "User not authorized to create account",
			slog.String("user_id", userID),
			slog.String("workplace_id", workplaceID))
		return nil, err
	}

	// Validate currency if currencyRepo is available
	if s.currencyRepo != nil {
		if _, err := s.currencyRepo.FindCurrencyByCode(ctx, req.CurrencyCode); err != nil {
			s.LogError(ctx, err, "Invalid currency code",
				slog.String("currency_code", req.CurrencyCode))
			return nil, fmt.Errorf("invalid currency code: %w", err)
		}
	}

	now := time.Now()
	newAccountID := uuid.NewString()

	parentID := ""
	if req.ParentAccountID != nil {
		parentID = *req.ParentAccountID
		// Validate parent account exists and belongs to same workplace
		parentAccount, err := s.accountRepo.FindAccountByID(ctx, parentID)
		if err != nil {
			s.LogError(ctx, err, "Failed to find parent account",
				slog.String("parent_id", parentID))
			return nil, fmt.Errorf("invalid parent account: %w", err)
		}
		if parentAccount.WorkplaceID != workplaceID {
			err := apperrors.ErrValidation
			s.LogError(ctx, err, "Parent account belongs to different workplace",
				slog.String("parent_workplace", parentAccount.WorkplaceID),
				slog.String("requested_workplace", workplaceID))
			return nil, fmt.Errorf("parent account belongs to different workplace: %w", err)
		}
	}

	// Create domain.Account, ensuring WorkplaceID is set
	account := domain.Account{
		AccountID:       newAccountID,
		WorkplaceID:     workplaceID,
		Name:            req.Name,
		AccountType:     domain.AccountType(req.AccountType),
		CurrencyCode:    req.CurrencyCode,
		CFID:            req.CFID,
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

	err := s.accountRepo.SaveAccount(ctx, account)
	if err != nil {
		s.LogError(ctx, err, "Failed to save account",
			slog.String("account_id", account.AccountID),
			slog.String("workplace_id", workplaceID))
		return nil, err
	}

	s.LogInfo(ctx, "Account created successfully",
		slog.String("account_id", account.AccountID),
		slog.String("workplace_id", workplaceID))
	return &account, nil
}

func (s *accountService) GetAccountByID(ctx context.Context, workplaceID string, accountID string, userID string) (*domain.Account, error) {
	// Authorize user action
	if err := s.AuthorizeUser(ctx, userID, workplaceID, domain.RoleReadOnly); err != nil {
		s.LogError(ctx, err, "User not authorized to view account",
			slog.String("workplace_id", workplaceID),
			slog.String("account_id", accountID))
		return nil, err
	}

	account, err := s.accountRepo.FindAccountByID(ctx, accountID)
	if err != nil {
		s.LogError(ctx, err, "Failed to find account by ID",
			slog.String("account_id", accountID))
		return nil, fmt.Errorf("failed to find account: %w", err)
	}

	// Verify the account belongs to the specified workplace
	if account.WorkplaceID != workplaceID {
		err := apperrors.ErrNotFound
		s.LogError(ctx, err, "Account not found in specified workplace",
			slog.String("account_id", accountID),
			slog.String("workplace_id", workplaceID))
		return nil, err
	}

	return account, nil
}

func (s *accountService) GetAccountByCFID(ctx context.Context, workplaceID string, cfid string, userID string) (*domain.Account, error) {
	// Validate input
	if cfid == "" {
		err := apperrors.NewValidationFailedError("CFID cannot be empty")
		s.LogError(ctx, err, "Empty CFID provided")
		return nil, err
	}

	// Authorize user action - allow any authenticated user with read-only role
	if err := s.AuthorizeUser(ctx, "", workplaceID, domain.RoleReadOnly); err != nil {
		s.LogError(ctx, err, "User not authorized to view account by CFID",
			slog.String("workplace_id", workplaceID),
			slog.String("cfid", cfid))
		return nil, err
	}

	// Call repository to find account by CFID
	account, err := s.accountRepo.FindAccountByCFID(ctx, cfid, workplaceID)
	if err != nil {
		s.LogError(ctx, err, "Failed to find account by CFID",
			slog.String("cfid", cfid),
			slog.String("workplace_id", workplaceID))
		return nil, fmt.Errorf("failed to find account: %w", err)
	}

	// Verify the account belongs to the specified workplace (should be handled by repo, but double-checking)
	if account.WorkplaceID != workplaceID {
		err := apperrors.ErrNotFound
		s.LogError(ctx, err, "Account not found in specified workplace",
			slog.String("cfid", cfid),
			slog.String("workplace_id", workplaceID))
		return nil, err
	}

	s.LogDebug(ctx, "Account retrieved by CFID successfully",
		slog.String("account_id", account.AccountID),
		slog.String("cfid", cfid),
		slog.String("workplace_id", account.WorkplaceID))

	return account, nil
}

func (s *accountService) GetAccountByIDs(ctx context.Context, workplaceID string, accountIDs []string, userID string) (map[string]domain.Account, error) {
	accounts, err := s.accountRepo.FindAccountsByIDs(ctx, accountIDs)
	if err != nil {
		s.LogError(ctx, err, "Failed to find accounts by IDs",
			slog.String("account_ids", fmt.Sprintf("%v", accountIDs)))
		return nil, err
	}

	// Authorization: Check if all accounts belong to the expected workplace
	for _, account := range accounts {
		if account.WorkplaceID != workplaceID {
			s.LogDebug(ctx, "Account found but belongs to different workplace",
				slog.String("account_id", account.AccountID),
				slog.String("account_workplace", account.WorkplaceID),
				slog.String("requested_workplace", workplaceID))
			return nil, apperrors.ErrNotFound
		}
	}

	return accounts, nil
}

func (s *accountService) ListAccounts(ctx context.Context, workplaceID string, limit int, offset int) ([]domain.Account, error) {
	accounts, err := s.accountRepo.ListAccounts(ctx, workplaceID, limit, offset)
	if err != nil {
		s.LogError(ctx, err, "Failed to list accounts",
			slog.String("workplace_id", workplaceID),
			slog.Int("limit", limit),
			slog.Int("offset", offset))
		return nil, fmt.Errorf("failed to list accounts for workplace %s: %w", workplaceID, err)
	}

	if accounts == nil {
		return []domain.Account{}, nil // Return empty slice if repo returns nil
	}

	s.LogDebug(ctx, "Accounts listed successfully",
		slog.Int("count", len(accounts)),
		slog.String("workplace_id", workplaceID))
	return accounts, nil
}

func (s *accountService) UpdateAccount(ctx context.Context, workplaceID string, accountID string, req dto.UpdateAccountRequest, userID string) (*domain.Account, error) {
	// Fetch the existing account
	account, err := s.GetAccountByID(ctx, workplaceID, accountID, userID)
	if err != nil {
		return nil, err // GetAccountByID already logs errors
	}

	// Apply updates
	updated := false
	if req.Name != nil {
		account.Name = *req.Name
		updated = true
	}
	if req.Description != nil {
		account.Description = *req.Description
		updated = true
	}
	if req.CFID != nil {
		account.CFID = *req.CFID
		updated = true
	}
	if req.IsActive != nil {
		account.IsActive = *req.IsActive
		updated = true
	}
	if !updated {
		s.LogDebug(ctx, "No fields provided for account update",
			slog.String("account_id", accountID))
		return account, nil
	}

	// Update audit fields
	now := time.Now()
	account.LastUpdatedAt = now
	account.LastUpdatedBy = userID

	err = s.accountRepo.UpdateAccount(ctx, *account)
	if err != nil {
		s.LogError(ctx, err, "Failed to update account",
			slog.String("account_id", accountID))
		return nil, err
	}

	s.LogInfo(ctx, "Account updated successfully",
		slog.String("account_id", account.AccountID),
		slog.String("workplace_id", account.WorkplaceID))
	return account, nil
}

func (s *accountService) DeactivateAccount(ctx context.Context, workplaceID string, accountID string, userID string) error {
	// First verify that the account exists and belongs to the workplace
	_, err := s.GetAccountByID(ctx, workplaceID, accountID, userID)
	if err != nil {
		return err // GetAccountByID already logs errors
	}

	// Deactivate the account
	now := time.Now()
	err = s.accountRepo.DeactivateAccount(ctx, accountID, userID, now)
	if err != nil {
		s.LogError(ctx, err, "Failed to deactivate account",
			slog.String("account_id", accountID))
		return err
	}

	s.LogInfo(ctx, "Account deactivated successfully",
		slog.String("account_id", accountID),
		slog.String("workplace_id", workplaceID))
	return nil
}

func (s *accountService) CalculateAccountBalance(ctx context.Context, workplaceID string, accountID string, userID string) (decimal.Decimal, error) {
	// First check if account exists and belongs to workplace
	account, err := s.GetAccountByID(ctx, workplaceID, accountID, userID)
	if err != nil {
		s.LogError(ctx, err, "Failed to find account for balance calculation",
			slog.String("account_id", accountID),
			slog.String("workplace_id", workplaceID))
		return decimal.Zero, err
	}

	return account.Balance, nil
}

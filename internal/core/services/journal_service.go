package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/SscSPs/money_managemet_app/internal/apperrors"
	"github.com/SscSPs/money_managemet_app/internal/core/domain"
	portsrepo "github.com/SscSPs/money_managemet_app/internal/core/ports/repositories"
	portssvc "github.com/SscSPs/money_managemet_app/internal/core/ports/services"
	"github.com/SscSPs/money_managemet_app/internal/dto"
	"github.com/SscSPs/money_managemet_app/internal/middleware"
	"github.com/SscSPs/money_managemet_app/internal/models"
	"github.com/shopspring/decimal"
	// We might need a UUID library later, e.g., "github.com/google/uuid"
)

var (
	ErrJournalUnbalanced = errors.New("journal entries do not balance to zero")
	ErrJournalMinEntries = errors.New("journal must have at least two transaction entries")
	ErrAccountNotFound   = errors.New("account not found")
	ErrCurrencyMismatch  = errors.New("account currency does not match journal currency")
)

// journalService provides core journal and transaction operations.
type journalService struct {
	accountRepo  portsrepo.AccountRepository
	journalRepo  portsrepo.JournalRepository
	workplaceSvc portssvc.WorkplaceService // Added for authorization checks
	// userRepo portsrepo.UserRepository
}

// NewJournalService creates a new JournalService.
func NewJournalService(accountRepo portsrepo.AccountRepository, journalRepo portsrepo.JournalRepository, workplaceSvc portssvc.WorkplaceService) portssvc.JournalService {
	return &journalService{
		accountRepo:  accountRepo,
		journalRepo:  journalRepo,
		workplaceSvc: workplaceSvc,
	}
}

// Ensure JournalService implements the portssvc.JournalService interface
var _ portssvc.JournalService = (*journalService)(nil)

// getSignedAmount applies the correct sign to a transaction amount based on account type and transaction type.
func (s *journalService) getSignedAmount(txn domain.Transaction, accountType domain.AccountType) (decimal.Decimal, error) {
	signedAmount := txn.Amount
	isDebit := txn.TransactionType == domain.Debit

	// Determine sign based on convention (PRD FR-M1-03)
	// DEBIT to ASSET/EXPENSE -> Positive (+)
	// CREDIT to ASSET/EXPENSE -> Negative (-)
	// DEBIT to LIABILITY/EQUITY/INCOME -> Negative (-)
	// CREDIT to LIABILITY/EQUITY/INCOME -> Positive (+)
	switch accountType {
	case domain.Asset, domain.Expense:
		if !isDebit { // Credit to Asset/Expense
			signedAmount = signedAmount.Neg()
		}
	case domain.Liability, domain.Equity, domain.Income:
		if isDebit { // Debit to Liability/Equity/Income
			signedAmount = signedAmount.Neg()
		}
	default:
		// This indicates an invalid account type, potentially a data integrity issue or bug
		return decimal.Zero, fmt.Errorf("unknown account type '%s' encountered for account ID %s", accountType, txn.AccountID)
	}
	return signedAmount, nil
}

// validateJournalBalance checks if the transactions for a journal balance to zero.
func (s *journalService) validateJournalBalance(transactions []domain.Transaction, accountTypes map[string]domain.AccountType) error {
	if len(transactions) < 2 {
		return ErrJournalMinEntries
	}

	zero := decimal.NewFromInt(0)
	sum := zero

	for _, txn := range transactions {
		// Ensure amount is positive (as per PRD)
		if txn.Amount.LessThanOrEqual(zero) {
			return fmt.Errorf("transaction amount must be positive for transaction ID %s", txn.TransactionID)
		}

		accountType, ok := accountTypes[txn.AccountID]
		if !ok {
			// This shouldn't happen if the calling function pre-fetches correctly
			return fmt.Errorf("account type not found for account ID %s", txn.AccountID)
		}

		signedAmount, err := s.getSignedAmount(txn, accountType)
		if err != nil {
			// Propagate error from getSignedAmount (e.g., unknown account type)
			return fmt.Errorf("error calculating signed amount for transaction %s: %w", txn.TransactionID, err)
		}

		sum = sum.Add(signedAmount)
	}

	if !sum.Equal(zero) {
		return fmt.Errorf("%w: sum is %s", ErrJournalUnbalanced, sum.String())
	}

	return nil
}

// Helper to convert []models.Transaction to []domain.Transaction
func modelToDomainTransactions(ms []models.Transaction) []domain.Transaction {
	ds := make([]domain.Transaction, len(ms))
	for i, m := range ms {
		ds[i] = domain.Transaction{
			TransactionID:   m.TransactionID,
			JournalID:       m.JournalID,
			AccountID:       m.AccountID,
			Amount:          m.Amount,
			TransactionType: domain.TransactionType(m.TransactionType),
			CurrencyCode:    m.CurrencyCode,
			Notes:           m.Notes,
			AuditFields: domain.AuditFields{
				CreatedAt:     m.CreatedAt,
				CreatedBy:     m.CreatedBy,
				LastUpdatedAt: m.LastUpdatedAt,
				LastUpdatedBy: m.LastUpdatedBy,
			},
		}
	}
	return ds
}

// Helper to convert models.Journal to domain.Journal
func modelToDomainJournal(m models.Journal) domain.Journal {
	return domain.Journal{
		JournalID:    m.JournalID,
		JournalDate:  m.JournalDate,
		Description:  m.Description,
		CurrencyCode: m.CurrencyCode,
		Status:       domain.JournalStatus(m.Status),
		AuditFields: domain.AuditFields{
			CreatedAt:     m.CreatedAt,
			CreatedBy:     m.CreatedBy,
			LastUpdatedAt: m.LastUpdatedAt,
			LastUpdatedBy: m.LastUpdatedBy,
		},
	}
}

// CreateJournal creates a new journal entry with its transactions after validation.
// Implements portssvc.JournalService
func (s *journalService) CreateJournal(ctx context.Context, workplaceID string, req dto.CreateJournalRequest, creatorUserID string) (*domain.Journal, error) {
	logger := middleware.GetLoggerFromCtx(ctx)

	// --- Authorization Check --- (Requires WorkplaceService injection)
	if s.workplaceSvc != nil {
		if err := s.workplaceSvc.AuthorizeUserAction(ctx, creatorUserID, workplaceID, domain.RoleMember); err != nil {
			logger.Warn("Authorization failed for CreateJournal", slog.String("user_id", creatorUserID), slog.String("workplace_id", workplaceID), slog.String("error", err.Error()))
			return nil, err // Propagate auth error (NotFound or Forbidden)
		}
	} else {
		logger.Warn("WorkplaceService not available for authorization check in CreateJournal")
	}

	// --- Basic Validation ---
	if len(req.Transactions) < 2 {
		return nil, ErrJournalMinEntries
	}

	now := time.Now().UTC()
	journalID := uuid.NewString()

	// Prepare domain transactions from DTO
	domainTransactions := make([]domain.Transaction, len(req.Transactions))
	accountIDs := make([]string, 0, len(req.Transactions))
	for i, txnReq := range req.Transactions {
		// Validate positive amount (already done by binding, but good practice)
		if txnReq.Amount.LessThanOrEqual(decimal.Zero) {
			return nil, fmt.Errorf("%w: transaction amount must be positive for account %s", apperrors.ErrValidation, txnReq.AccountID)
		}
		domainTransactions[i] = domain.Transaction{
			TransactionID:   uuid.NewString(),
			JournalID:       journalID, // Link to the new journal
			AccountID:       txnReq.AccountID,
			Amount:          txnReq.Amount,
			TransactionType: txnReq.TransactionType,
			CurrencyCode:    req.CurrencyCode, // Use journal's currency
			Notes:           txnReq.Notes,
			AuditFields: domain.AuditFields{
				CreatedAt:     now,
				CreatedBy:     creatorUserID,
				LastUpdatedAt: now,
				LastUpdatedBy: creatorUserID,
			},
			// RunningBalance will be calculated and set by the repository
		}
		accountIDs = append(accountIDs, txnReq.AccountID)
	}

	// --- Fetch Accounts and Validate Further ---
	uniqueAccountIDs := uniqueStrings(accountIDs)
	accountsMap, err := s.accountRepo.FindAccountsByIDs(ctx, uniqueAccountIDs)
	if err != nil {
		logger.Error("Failed to fetch accounts for journal creation", slog.String("error", err.Error()), slog.String("workplace_id", workplaceID))
		return nil, fmt.Errorf("failed to fetch accounts: %w", err)
	}

	accountTypes := make(map[string]domain.AccountType)
	for _, id := range uniqueAccountIDs {
		acc, found := accountsMap[id]
		if !found {
			return nil, fmt.Errorf("%w: ID %s", ErrAccountNotFound, id)
		}
		if acc.WorkplaceID != workplaceID {
			logger.Warn("Account used in journal belongs to a different workplace", slog.String("journal_workplace", workplaceID), slog.String("account_id", id), slog.String("account_workplace", acc.WorkplaceID))
			return nil, fmt.Errorf("%w: account %s does not belong to workplace %s", ErrAccountNotFound, id, workplaceID)
		}
		if !acc.IsActive {
			return nil, fmt.Errorf("%w: account %s is inactive", apperrors.ErrValidation, id)
		}
		// Validate currency match
		if acc.CurrencyCode != req.CurrencyCode {
			return nil, fmt.Errorf("%w: account currency %s does not match journal currency %s for account %s", ErrCurrencyMismatch, acc.CurrencyCode, req.CurrencyCode, id)
		}
		accountTypes[id] = acc.AccountType
	}

	// Validate Balance (double-entry check)
	if err = s.validateJournalBalance(domainTransactions, accountTypes); err != nil {
		return nil, err
	}

	// --- Calculate Net Balance Changes for Accounts ---
	balanceChanges := make(map[string]decimal.Decimal)
	for _, txn := range domainTransactions {
		accountType := accountTypes[txn.AccountID] // We know this exists from validation
		signedAmount, err := s.getSignedAmount(txn, accountType)
		if err != nil {
			// Should not happen after validation, but handle defensively
			logger.Error("Error calculating signed amount during balance change calculation", slog.String("error", err.Error()), slog.String("transaction_id", txn.TransactionID))
			return nil, fmt.Errorf("internal error calculating balance changes: %w", err)
		}
		if currentChange, ok := balanceChanges[txn.AccountID]; ok {
			balanceChanges[txn.AccountID] = currentChange.Add(signedAmount)
		} else {
			balanceChanges[txn.AccountID] = signedAmount
		}
	}

	// --- Persistence ---
	domainJournal := domain.Journal{
		JournalID:    journalID,
		WorkplaceID:  workplaceID,
		JournalDate:  req.Date,
		Description:  req.Description,
		CurrencyCode: req.CurrencyCode,
		Status:       domain.Posted, // Default status
		AuditFields: domain.AuditFields{
			CreatedAt:     now,
			CreatedBy:     creatorUserID,
			LastUpdatedAt: now,
			LastUpdatedBy: creatorUserID,
		},
	}

	// Pass balance changes to the repository method
	err = s.journalRepo.SaveJournal(ctx, domainJournal, domainTransactions, balanceChanges)
	if err != nil {
		logger.Error("Failed to save journal", slog.String("error", err.Error()), slog.String("workplace_id", workplaceID))
		return nil, fmt.Errorf("failed to save journal: %w", err)
	}

	logger.Info("Journal created successfully", slog.String("journal_id", domainJournal.JournalID), slog.String("workplace_id", workplaceID))
	// Return the journal without transactions populated by default (as per GetJournalByID)
	// Caller can fetch transactions separately if needed.
	domainJournal.Transactions = nil // Clear transactions before returning
	return &domainJournal, nil
}

// GetJournalByID retrieves a specific journal entry (without transactions).
// Implements portssvc.JournalService
func (s *journalService) GetJournalByID(ctx context.Context, workplaceID string, journalID string, requestingUserID string) (*domain.Journal, error) {
	logger := middleware.GetLoggerFromCtx(ctx)

	// --- Authorization Check --- (Requires WorkplaceService injection)
	// Check if the requesting user is a member of the target workplace.
	if s.workplaceSvc != nil {
		if err := s.workplaceSvc.AuthorizeUserAction(ctx, requestingUserID, workplaceID, domain.RoleMember); err != nil {
			logger.Warn("Authorization failed for GetJournalByID", slog.String("user_id", requestingUserID), slog.String("workplace_id", workplaceID), slog.String("journal_id", journalID), slog.String("error", err.Error()))
			return nil, err // Return NotFound or Forbidden
		}
	} else {
		logger.Warn("WorkplaceService not available for authorization check in GetJournalByID")
	}

	// Fetch journal from repository
	journal, err := s.journalRepo.FindJournalByID(ctx, journalID)
	if err != nil {
		if !errors.Is(err, apperrors.ErrNotFound) {
			logger.Error("Failed to find journal by ID", slog.String("error", err.Error()), slog.String("journal_id", journalID))
		}
		// Propagate NotFound
		return nil, fmt.Errorf("failed to find journal by ID %s: %w", journalID, err)
	}

	// Final check: Ensure the found journal actually belongs to the requested workplace
	if journal.WorkplaceID != workplaceID {
		logger.Warn("Journal found but belongs to different workplace", slog.String("journal_id", journalID), slog.String("journal_workplace", journal.WorkplaceID), slog.String("requested_workplace", workplaceID))
		return nil, apperrors.ErrNotFound // Obscure existence
	}

	// Fetch associated transactions
	transactions, err := s.journalRepo.FindTransactionsByJournalID(ctx, journalID)
	if err != nil {
		// Log error but don't necessarily fail the whole request?
		// Depending on requirements, maybe return journal header even if transactions fail?
		// For now, let's fail if transactions can't be fetched.
		logger.Error("Failed to fetch transactions for journal", slog.String("error", err.Error()), slog.String("journal_id", journalID))
		return nil, fmt.Errorf("failed to retrieve transactions for journal %s: %w", journalID, apperrors.ErrInternal) // Return generic internal error
	}

	// Populate the transactions field
	journal.Transactions = transactions

	logger.Debug("Journal and transactions retrieved successfully", slog.String("journal_id", journalID), slog.String("workplace_id", workplaceID), slog.Int("transaction_count", len(transactions)))
	return journal, nil
}

// ListJournals retrieves a paginated list of journals for a specific workplace.
// Implements portssvc.JournalService
func (s *journalService) ListJournals(ctx context.Context, workplaceID string, limit int, offset int, requestingUserID string) ([]domain.Journal, error) {
	logger := middleware.GetLoggerFromCtx(ctx)

	// --- Authorization Check --- (Requires WorkplaceService injection)
	if s.workplaceSvc != nil {
		if err := s.workplaceSvc.AuthorizeUserAction(ctx, requestingUserID, workplaceID, domain.RoleMember); err != nil {
			logger.Warn("Authorization failed for ListJournals", slog.String("user_id", requestingUserID), slog.String("workplace_id", workplaceID), slog.String("error", err.Error()))
			return nil, err // Return NotFound or Forbidden
		}
	} else {
		logger.Warn("WorkplaceService not available for authorization check in ListJournals")
	}

	// Fetch from repository using workplaceID, limit, offset
	domainJournals, err := s.journalRepo.ListJournalsByWorkplace(ctx, workplaceID, limit, offset)
	if err != nil {
		logger.Error("Failed to list journals from repository", slog.String("error", err.Error()), slog.String("workplace_id", workplaceID))
		return nil, fmt.Errorf("failed to retrieve journals: %w", apperrors.ErrInternal)
	}

	// If no journals found, return early
	if len(domainJournals) == 0 {
		return []domain.Journal{}, nil
	}

	// Extract journal IDs to fetch their transactions
	journalIDs := make([]string, len(domainJournals))
	for i, j := range domainJournals {
		journalIDs[i] = j.JournalID
	}

	// Fetch all transactions for these journals in one go
	transactionsMap, err := s.journalRepo.FindTransactionsByJournalIDs(ctx, journalIDs)
	if err != nil {
		logger.Error("Failed to fetch transactions for listed journals", slog.String("error", err.Error()), slog.String("workplace_id", workplaceID))
		// Non-fatal? Maybe return journals without transactions if this fails?
		// For now, treat as an internal error and return failure.
		return nil, fmt.Errorf("failed to retrieve journal details: %w", apperrors.ErrInternal)
	}

	// Populate transactions into the journal objects
	for i := range domainJournals {
		// Use index to modify the slice element directly
		if txns, ok := transactionsMap[domainJournals[i].JournalID]; ok {
			domainJournals[i].Transactions = txns
		} else {
			// Should not happen if FindTransactionsByJournalIDs ensures all IDs have an entry
			domainJournals[i].Transactions = []domain.Transaction{} // Ensure it's an empty slice, not nil
		}
	}

	return domainJournals, nil
}

// UpdateJournal updates details of a specific journal entry.
// Implements portssvc.JournalService
func (s *journalService) UpdateJournal(ctx context.Context, workplaceID string, journalID string, req dto.UpdateJournalRequest, requestingUserID string) (*domain.Journal, error) {
	logger := middleware.GetLoggerFromCtx(ctx)

	// --- Authorization Check --- (Requires WorkplaceService injection)
	if s.workplaceSvc != nil {
		// Check if user is at least a member (maybe admin required? TBD)
		if err := s.workplaceSvc.AuthorizeUserAction(ctx, requestingUserID, workplaceID, domain.RoleMember); err != nil {
			logger.Warn("Authorization failed for UpdateJournal", slog.String("user_id", requestingUserID), slog.String("workplace_id", workplaceID), slog.String("journal_id", journalID), slog.String("error", err.Error()))
			return nil, err
		}
	} else {
		logger.Warn("WorkplaceService not available for authorization check in UpdateJournal")
	}

	// Fetch the journal
	journal, err := s.journalRepo.FindJournalByID(ctx, journalID)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			logger.Warn("Journal not found for update", slog.String("journal_id", journalID), slog.String("workplace_id", workplaceID))
		} else {
			logger.Error("Failed to find journal for update", slog.String("error", err.Error()), slog.String("journal_id", journalID))
		}
		return nil, err // Propagate NotFound or other find errors
	}

	// Verify workplace ID match
	if journal.WorkplaceID != workplaceID {
		logger.Warn("Attempt to update journal from wrong workplace", slog.String("journal_id", journalID), slog.String("journal_workplace", journal.WorkplaceID), slog.String("requested_workplace", workplaceID))
		return nil, apperrors.ErrNotFound
	}

	// TODO: Add check: Can only update journals with status 'Posted'?
	// if journal.Status != domain.Posted { ... return apperrors.ErrValidation(...) }

	// Apply updates from request DTO
	updated := false
	if req.Date != nil {
		journal.JournalDate = *req.Date
		updated = true
	}
	if req.Description != nil {
		journal.Description = *req.Description
		updated = true
	}

	if !updated {
		logger.Debug("No fields provided for journal update", slog.String("journal_id", journalID))
		return journal, nil // Return unmodified journal if no changes
	}

	now := time.Now()
	journal.LastUpdatedAt = now
	journal.LastUpdatedBy = requestingUserID

	// TODO: Add and call s.journalRepo.UpdateJournal(ctx, *journal)
	// err = s.journalRepo.UpdateJournal(ctx, *journal)
	// if err != nil { ... }

	logger.Warn("UpdateJournal service method not fully implemented - repo call missing")
	// Return the potentially modified journal for now, but indicate it's not saved
	// return journal, nil // Incorrect - should return error until repo call implemented
	return nil, fmt.Errorf("UpdateJournal repository call not implemented") // Placeholder Error
}

// DeactivateJournal marks a journal as inactive (conceptually; might involve changing status).
// Implements portssvc.JournalService
func (s *journalService) DeactivateJournal(ctx context.Context, workplaceID string, journalID string, requestingUserID string) error {
	logger := middleware.GetLoggerFromCtx(ctx)

	// --- Authorization Check --- (Requires WorkplaceService injection)
	if s.workplaceSvc != nil {
		// Typically requires Admin role for deactivation/reversal
		if err := s.workplaceSvc.AuthorizeUserAction(ctx, requestingUserID, workplaceID, domain.RoleAdmin); err != nil {
			logger.Warn("Authorization failed for DeactivateJournal", slog.String("user_id", requestingUserID), slog.String("workplace_id", workplaceID), slog.String("journal_id", journalID), slog.String("error", err.Error()))
			return err
		}
	} else {
		logger.Warn("WorkplaceService not available for authorization check in DeactivateJournal")
	}

	// Fetch journal
	journal, err := s.journalRepo.FindJournalByID(ctx, journalID)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			logger.Warn("Journal not found for deactivation", slog.String("journal_id", journalID), slog.String("workplace_id", workplaceID))
		} else {
			logger.Error("Failed to find journal for deactivation", slog.String("error", err.Error()), slog.String("journal_id", journalID))
		}
		return err // Propagate NotFound or other find errors
	}

	// Verify workplace ID match
	if journal.WorkplaceID != workplaceID {
		logger.Warn("Attempt to deactivate journal from wrong workplace", slog.String("journal_id", journalID), slog.String("journal_workplace", journal.WorkplaceID), slog.String("requested_workplace", workplaceID))
		return apperrors.ErrNotFound
	}

	// Check if already inactive/reversed (prevents repeated action)
	if journal.Status == domain.Reversed { // Assuming Reversed is the 'inactive' state
		logger.Warn("Attempt to deactivate already reversed journal", slog.String("journal_id", journalID))
		return fmt.Errorf("%w: journal %s is already reversed", apperrors.ErrValidation, journalID)
	}

	// TODO: Add check: Can only deactivate journals with status 'Posted'?

	// TODO: Add and call s.journalRepo.UpdateJournalStatus(ctx, journalID, domain.Reversed, requestingUserID, time.Now())
	// err = s.journalRepo.UpdateJournalStatus(ctx, journalID, domain.Reversed, requestingUserID, time.Now())
	// if err != nil { ... }

	logger.Warn("DeactivateJournal service method not fully implemented - repo call missing")
	return fmt.Errorf("DeactivateJournal repository call not implemented") // Placeholder Error
}

// ListTransactionsByAccount retrieves a paginated list of transactions for a specific account within a workplace.
func (s *journalService) ListTransactionsByAccount(ctx context.Context, workplaceID string, accountID string, limit int, offset int, requestingUserID string) ([]domain.Transaction, error) {
	logger := middleware.GetLoggerFromCtx(ctx)

	// --- Authorization Check --- (User must be member of workplace)
	if s.workplaceSvc != nil {
		if err := s.workplaceSvc.AuthorizeUserAction(ctx, requestingUserID, workplaceID, domain.RoleMember); err != nil {
			logger.Warn("Authorization failed for ListTransactionsByAccount", slog.String("user_id", requestingUserID), slog.String("workplace_id", workplaceID), slog.String("account_id", accountID), slog.String("error", err.Error()))
			return nil, err // Return NotFound or Forbidden
		}
	} else {
		logger.Warn("WorkplaceService not available for authorization check in ListTransactionsByAccount")
	}

	// --- Optional: Verify account exists and belongs to workplace ---
	// This prevents querying transactions for accounts the user shouldn't see,
	// even if they have access to the workplace.
	_, err := s.accountRepo.FindAccountByID(ctx, accountID)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			logger.Warn("Account not found when listing transactions", slog.String("account_id", accountID), slog.String("workplace_id", workplaceID))
			return nil, apperrors.ErrNotFound // Account not found
		}
		logger.Error("Failed to verify account existence before listing transactions", slog.String("account_id", accountID), slog.String("workplace_id", workplaceID), slog.String("error", err.Error()))
		return nil, fmt.Errorf("failed to verify account %s: %w", accountID, err)
	}
	// We might add an explicit check here: if account.WorkplaceID != workplaceID { return nil, apperrors.ErrNotFound }
	// However, FindTransactionsByAccountID in the repo already filters by workplaceID,
	// so this check might be redundant depending on desired error message/behavior.
	// Let's rely on the repo's filter for now.

	// --- Fetch Transactions ---
	// TODO: Adapt repository method or add new one to handle pagination (limit, offset)
	// Current FindTransactionsByAccountID does not support pagination.
	transactions, err := s.journalRepo.FindTransactionsByAccountID(ctx, workplaceID, accountID)
	if err != nil {
		logger.Error("Failed to list transactions by account from repository", slog.String("error", err.Error()), slog.String("account_id", accountID), slog.String("workplace_id", workplaceID))
		return nil, fmt.Errorf("failed to retrieve transactions: %w", apperrors.ErrInternal)
	}

	// Apply pagination manually for now until repo is updated
	totalTransactions := len(transactions)
	startIndex := offset
	endIndex := offset + limit

	if startIndex < 0 {
		startIndex = 0
	}
	if startIndex >= totalTransactions {
		return []domain.Transaction{}, nil // Offset is beyond the total number of transactions
	}
	if endIndex > totalTransactions {
		endIndex = totalTransactions
	}

	pagedTransactions := transactions[startIndex:endIndex]

	logger.Debug("Transactions listed successfully for account", slog.String("account_id", accountID), slog.String("workplace_id", workplaceID), slog.Int("count", len(pagedTransactions)))
	return pagedTransactions, nil
}

// uniqueStrings returns a slice containing only the unique strings from the input.
func uniqueStrings(input []string) []string {
	seen := make(map[string]struct{}, len(input))
	result := make([]string, 0, len(input))
	for _, str := range input {
		if _, ok := seen[str]; !ok {
			seen[str] = struct{}{}
			result = append(result, str)
		}
	}
	return result
}

// CalculateAccountBalance calculates the current balance of a given account within its workplace.
// Note: This might be better placed in AccountService if it doesn't need journal specifics beyond transactions.
func (s *journalService) CalculateAccountBalance(ctx context.Context, workplaceID string, accountID string) (decimal.Decimal, error) {
	logger := middleware.GetLoggerFromCtx(ctx)

	// --- Authorization Check? ---
	// Should the caller (e.g., handler) already have verified user access to the workplace?
	// Or should this service check it? Assume caller handles it for now.

	// 1. Find the account to verify existence, activity, type, and workplace match
	account, err := s.accountRepo.FindAccountByID(ctx, accountID)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			return decimal.Zero, fmt.Errorf("%w: ID %s", ErrAccountNotFound, accountID)
		}
		return decimal.Zero, fmt.Errorf("failed to find account %s: %w", accountID, err)
	}
	if account.WorkplaceID != workplaceID {
		logger.Warn("CalculateAccountBalance requested for account in wrong workplace", slog.String("account_id", accountID), slog.String("account_workplace", account.WorkplaceID), slog.String("requested_workplace", workplaceID))
		return decimal.Zero, fmt.Errorf("account %s not found in workplace %s", accountID, workplaceID)
	}
	if !account.IsActive {
		return decimal.Zero, fmt.Errorf("account %s is inactive", accountID)
	}

	// 2. Fetch all transactions for this account within the workplace
	transactions, err := s.journalRepo.FindTransactionsByAccountID(ctx, workplaceID, accountID)
	if err != nil {
		logger.Error("Failed to fetch transactions for account balance", slog.String("error", err.Error()), slog.String("account_id", accountID), slog.String("workplace_id", workplaceID))
		return decimal.Zero, fmt.Errorf("failed to fetch transactions for account %s in workplace %s: %w", accountID, workplaceID, err)
	}

	// 3. Calculate the balance by summing signed amounts
	balance := decimal.Zero
	for _, txn := range transactions {
		if txn.Amount.LessThanOrEqual(decimal.Zero) {
			logger.Error("Invalid non-positive transaction amount found during balance calculation", slog.String("transaction_id", txn.TransactionID), slog.String("account_id", accountID))
			return decimal.Zero, fmt.Errorf("invalid non-positive transaction amount found (ID: %s) for account %s", txn.TransactionID, accountID)
		}

		signedAmount, err := s.getSignedAmount(txn, account.AccountType)
		if err != nil {
			return decimal.Zero, fmt.Errorf("error calculating signed amount for transaction %s: %w", txn.TransactionID, err)
		}
		balance = balance.Add(signedAmount)
	}

	logger.Debug("Account balance calculated successfully", slog.String("account_id", accountID), slog.String("workplace_id", workplaceID), slog.String("balance", balance.String()))
	return balance, nil
}

// TODO: Add methods for:
// - FindJournalByID(ctx context.Context, journalID string) (*models.Journal, []models.Transaction, error)
// - Static data initialization trigger? (FR-M1-06)

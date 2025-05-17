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
	accountSvc   portssvc.AccountSvcFacade
	journalRepo  portsrepo.JournalRepositoryFacade
	workplaceSvc portssvc.WorkplaceSvcFacade // Updated to use WorkplaceSvcFacade
	// userRepo portsrepo.UserRepository
}

// NewJournalService creates a new JournalService.
func NewJournalService(journalRepo portsrepo.JournalRepositoryFacade, accountSvc portssvc.AccountSvcFacade, workplaceSvc portssvc.WorkplaceSvcFacade) portssvc.JournalSvcFacade {
	return &journalService{
		accountSvc:   accountSvc,
		journalRepo:  journalRepo,
		workplaceSvc: workplaceSvc,
	}
}

// Ensure JournalService implements the portssvc.JournalSvcFacade interface
var _ portssvc.JournalSvcFacade = (*journalService)(nil)

// getSignedAmount applies the correct sign to a transaction amount based on account type and transaction type.
func (s *journalService) getSignedAmount(txn domain.Transaction, accountType domain.AccountType) (decimal.Decimal, error) {
	signedAmount := txn.Amount
	isDebit := txn.TransactionType == domain.Debit

	// Determine sign based on convention (PRD FR-M1-03)
	// DEBIT to ASSET/EXPENSE -> Positive (+)
	// CREDIT to ASSET/EXPENSE -> Negative (-)
	// DEBIT to LIABILITY/EQUITY/REVENUE -> Negative (-)
	// CREDIT to LIABILITY/EQUITY/REVENUE -> Positive (+)
	switch accountType {
	case domain.Asset, domain.Expense:
		if !isDebit { // Credit to Asset/Expense
			signedAmount = signedAmount.Neg()
		}
	case domain.Liability, domain.Equity, domain.Revenue:
		if isDebit { // Debit to Liability/Equity/Income
			signedAmount = signedAmount.Neg()
		}
	default:
		// This indicates an invalid account type, potentially a data integrity issue or bug
		return decimal.Zero, fmt.Errorf("unknown account type '%s' encountered for account ID %s", accountType, txn.AccountID)
	}
	return signedAmount, nil
}

// validateJournalBalance checks if the transactions for a journal balance properly.
func (s *journalService) validateJournalBalance(transactions []domain.Transaction, accountTypes map[string]domain.AccountType) error {
	if len(transactions) < 2 {
		return ErrJournalMinEntries
	}

	zero := decimal.NewFromInt(0)

	// For double-entry accounting, the sum of debits should equal the sum of credits
	// regardless of account type
	debitsSum := zero
	creditsSum := zero

	for _, txn := range transactions {
		// Ensure amount is positive (as per PRD)
		if txn.Amount.LessThanOrEqual(zero) {
			return fmt.Errorf("transaction amount must be positive for transaction ID %s", txn.TransactionID)
		}

		// Simply sum all debits and credits separately
		if txn.TransactionType == domain.Debit {
			debitsSum = debitsSum.Add(txn.Amount)
		} else {
			creditsSum = creditsSum.Add(txn.Amount)
		}
	}

	// Check if debits equal credits
	if !debitsSum.Equal(creditsSum) {
		return fmt.Errorf("%w: debits sum is %s and credits sum is %s",
			ErrJournalUnbalanced, debitsSum.String(), creditsSum.String())
	}

	return nil
}

// calculateJournalAmount computes the true economic value of a journal.
// For a balanced journal with equal debit and credit sides,
// we need to pick one side that represents the actual money movement.
func (s *journalService) calculateJournalAmount(transactions []domain.Transaction, accountTypes map[string]domain.AccountType) decimal.Decimal {
	if len(transactions) == 0 {
		return decimal.Zero
	}

	// For a balanced journal, the sum of all debit entries equals the sum of all credit entries.
	// This sum represents the total economic value of the journal.
	totalDebits := decimal.Zero
	for _, txn := range transactions {
		// Ensure the account exists in the provided accountTypes map.
		// This retains a safety check from the original logic, though the accountType itself isn't used in the simplified sum.
		_, exists := accountTypes[txn.AccountID]
		if !exists {
			// If a logger was available and this scenario is unexpected, it could be logged.
			// e.g., s.logger.Warnf("Transaction for unknown account ID %s skipped in amount calculation", txn.AccountID)
			continue
		}

		if txn.TransactionType == domain.Debit {
			totalDebits = totalDebits.Add(txn.Amount)
		}
	}
	return totalDebits
}

// calculateJournalAmountSimple is a simplified fallback when account types aren't available.
// This is unreliable and should be avoided. It calculates the amount by summing debits.
// DEPRECATED: This should be replaced with the proper account-type-aware calculation.
func calculateJournalAmountSimple(transactions []domain.Transaction) decimal.Decimal {
	totalAmount := decimal.Zero
	for _, txn := range transactions {
		if txn.TransactionType == domain.Debit {
			totalAmount = totalAmount.Add(txn.Amount)
		}
	}
	return totalAmount
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
// Implements portssvc.JournalSvcFacade
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
	accountsMap, err := s.accountSvc.GetAccountByIDs(ctx, workplaceID, uniqueAccountIDs)
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

	// Calculate the total amount of the journal using account types information
	totalAmount := s.calculateJournalAmount(domainTransactions, accountTypes)
	domainJournal.Amount = totalAmount

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
// Implements portssvc.JournalSvcFacade
func (s *journalService) GetJournalByID(ctx context.Context, workplaceID string, journalID string, requestingUserID string) (*domain.Journal, error) {
	logger := middleware.GetLoggerFromCtx(ctx)

	// --- Authorization Check --- (Requires WorkplaceService injection)
	// Check if the requesting user is a member of the target workplace.
	if s.workplaceSvc != nil {
		if err := s.workplaceSvc.AuthorizeUserAction(ctx, requestingUserID, workplaceID, domain.RoleReadOnly); err != nil {
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

	// Calculate the total amount of the journal properly, getting account types
	if len(transactions) > 0 {
		// Extract account IDs from transactions
		accountIDs := make([]string, 0, len(transactions))
		for _, txn := range transactions {
			accountIDs = append(accountIDs, txn.AccountID)
		}

		// Fetch account details to get account types
		accountsMap, err := s.accountSvc.GetAccountByIDs(ctx, workplaceID, uniqueStrings(accountIDs))
		if err != nil {
			logger.Warn("Could not fetch account types for journal amount calculation",
				slog.String("error", err.Error()),
				slog.String("journal_id", journalID))
			// Fallback to simple calculation if we can't get account types
			totalAmount := calculateJournalAmountSimple(transactions)
			journal.Amount = totalAmount
		} else {
			// Create account types map
			accountTypes := make(map[string]domain.AccountType)
			for id, account := range accountsMap {
				accountTypes[id] = account.AccountType
			}

			// Calculate accurate journal amount using account types
			totalAmount := s.calculateJournalAmount(transactions, accountTypes)
			journal.Amount = totalAmount
		}
	}

	logger.Debug("Journal and transactions retrieved successfully", slog.String("journal_id", journalID), slog.String("workplace_id", workplaceID), slog.Int("transaction_count", len(transactions)))
	return journal, nil
}

// ListJournalsParams holds parameters for listing journals.
// NOTE: This might be defined in a DTO package instead.
// We will update this later to use NextToken.
type ListJournalsParams struct {
	WorkplaceID string
	Limit       int
	NextToken   *string // Added for token pagination
}

// ListJournals retrieves a paginated list of journals for a specific workplace.
func (j *journalService) ListJournals(ctx context.Context, workplaceID string, userID string, params dto.ListJournalsParams) (*dto.ListJournalsResponse, error) {
	logger := middleware.GetLoggerFromCtx(ctx)

	// Authorize user action (at least ReadOnly required to list journals)
	if err := j.workplaceSvc.AuthorizeUserAction(ctx, userID, workplaceID, domain.RoleReadOnly); err != nil {
		logger.Warn("Authorization failed for ListJournals", "error", err)
		return nil, err
	}

	// Fetch journals from repository using token
	// Note: Assuming params dto.ListJournalsParams is updated to include NextToken
	journals, nextToken, err := j.journalRepo.ListJournalsByWorkplace(ctx, workplaceID, params.Limit, params.NextToken, params.IncludeReversals)
	if err != nil {
		logger.Error("Failed to list journals from repository", "error", err)
		// Don't wrap; return specific error if needed, otherwise the original error
		return nil, err
	}

	// Convert domain journals to DTO responses
	journalResponses := make([]dto.JournalResponse, len(journals))
	for i, journal := range journals {
		// Ensure transactions are nil/empty for list view unless specifically requested later
		journal.Transactions = nil
		journalResponses[i] = dto.ToJournalResponse(&journal)
	}

	// Populate the response DTO with journals and the next token
	resp := &dto.ListJournalsResponse{
		Journals:  journalResponses,
		NextToken: nextToken,
	}

	logger.Info("Journals listed successfully", "count", len(journals))
	return resp, nil
}

// UpdateJournal updates the description and date of a journal entry.
// Implements portssvc.JournalSvcFacade
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
	err = s.journalRepo.UpdateJournal(ctx, *journal)
	if err != nil {
		logger.Error("Failed to save journal update to repository", slog.String("error", err.Error()), slog.String("journal_id", journalID))
		// Propagate potential ErrNotFound from repo
		return nil, fmt.Errorf("failed to save journal update: %w", err)
	}

	logger.Info("Journal updated successfully in repository", slog.String("journal_id", journalID))
	// Return the updated journal (without transactions)
	journal.Transactions = nil
	return journal, nil
}

// DeactivateJournal marks a journal as inactive (conceptually; might involve changing status).
// Implements portssvc.JournalSvcFacade
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

// ListTransactionsByAccount retrieves transactions for a specific account within a workplace.
func (j *journalService) ListTransactionsByAccount(ctx context.Context, workplaceID string, accountID string, userID string, params dto.ListTransactionsParams) (*dto.ListTransactionsResponse, error) {
	logger := middleware.GetLoggerFromCtx(ctx)

	// Authorize user action (at least ReadOnly required to list transactions)
	if err := j.workplaceSvc.AuthorizeUserAction(ctx, userID, workplaceID, domain.RoleReadOnly); err != nil {
		logger.Warn("Authorization failed for ListTransactionsByAccount", "error", err)
		return nil, err
	}

	// Set default limit if not provided
	limit := params.Limit
	if limit <= 0 {
		limit = 20 // Default limit
	}

	// Fetch transactions from repository with pagination
	transactions, nextToken, err := j.journalRepo.ListTransactionsByAccountID(ctx, workplaceID, accountID, limit, params.NextToken)
	if err != nil {
		logger.Error("Failed to list transactions by account from repository", "error", err)
		return nil, fmt.Errorf("failed to retrieve transactions: %w", err)
	}

	// Convert domain transactions to DTOs
	transactionResponses := dto.ToTransactionResponses(transactions)

	// Prepare response with pagination
	resp := &dto.ListTransactionsResponse{
		Transactions: transactionResponses,
		NextToken:    nextToken,
	}

	logger.Info("Transactions listed successfully for account", "count", len(transactions))
	return resp, nil
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

// CalculateAccountBalance calculates the current balance for a specific account.
// Note: This might be better placed in AccountService if it doesn't need journal specifics beyond transactions.
func (s *journalService) CalculateAccountBalance(ctx context.Context, workplaceID string, accountID string) (decimal.Decimal, error) {
	logger := middleware.GetLoggerFromCtx(ctx)

	// --- Authorization Check ---
	// Since this is a read operation, we only need ReadOnly access
	// Note: We'll rely on the account service's authorization check when calling GetAccountByID

	// 1. Find the account to verify existence, activity, type, and workplace match
	account, err := s.accountSvc.GetAccountByID(ctx, workplaceID, accountID)
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

	// 2. Fetch all transactions for this account within the workplace using pagination
	balance := decimal.Zero
	var nextToken *string
	const pageSize = 100 // Fetch more transactions per page for efficiency

	// Paginate through all transactions
	for {
		transactions, newNextToken, err := s.journalRepo.ListTransactionsByAccountID(ctx, workplaceID, accountID, pageSize, nextToken)
		if err != nil {
			logger.Error("Failed to fetch transactions page for account balance", slog.String("error", err.Error()), slog.String("account_id", accountID), slog.String("workplace_id", workplaceID))
			return decimal.Zero, fmt.Errorf("failed to fetch transactions for account %s in workplace %s: %w", accountID, workplaceID, err)
		}

		// Process this page of transactions
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

		// If no more pages, break the loop
		if newNextToken == nil || *newNextToken == "" {
			break
		}

		// Continue with next page
		nextToken = newNextToken
	}

	logger.Debug("Account balance calculated successfully", slog.String("account_id", accountID), slog.String("workplace_id", workplaceID), slog.String("balance", balance.String()))
	return balance, nil
}

// ReverseJournal creates a new journal entry that reverses a previously posted journal.
func (j *journalService) ReverseJournal(ctx context.Context, workplaceID string, journalID string, userID string) (*domain.Journal, error) {
	logger := middleware.GetLoggerFromCtx(ctx)

	// 1. Authorize user action (e.g., require member role)
	if err := j.workplaceSvc.AuthorizeUserAction(ctx, userID, workplaceID, domain.RoleMember); err != nil {
		logger.Warn("Authorization failed for ReverseJournal", "error", err)
		return nil, err // Error already contains appropriate type (e.g., ErrForbidden)
	}

	// 2. Fetch the original journal
	originalJournal, err := j.journalRepo.FindJournalByID(ctx, journalID)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			logger.Warn("Original journal not found for reversal")
			return nil, apperrors.ErrNotFound
		}
		logger.Error("Failed to fetch original journal for reversal", "error", err)
		return nil, fmt.Errorf("failed to retrieve original journal: %w", err)
	}

	// 3. Validate the original journal
	if originalJournal.WorkplaceID != workplaceID {
		logger.Warn("Attempted to reverse journal from wrong workplace")
		return nil, apperrors.ErrNotFound // Treat as not found in this context
	}
	if originalJournal.Status != domain.Posted {
		logger.Warn("Attempted to reverse non-posted journal", "status", originalJournal.Status)
		return nil, fmt.Errorf("%w: journal status is %s, expected POSTED", apperrors.ErrConflict, originalJournal.Status)
	}

	// 4. Fetch original transactions
	originalTransactions, err := j.journalRepo.FindTransactionsByJournalID(ctx, journalID)
	if err != nil {
		logger.Error("Failed to fetch original transactions for reversal", "error", err)
		return nil, fmt.Errorf("failed to retrieve original transactions: %w", err)
	}
	if len(originalTransactions) < 2 {
		// This should ideally not happen for a POSTED journal, but check anyway
		logger.Error("Original posted journal has less than 2 transactions", "transaction_count", len(originalTransactions))
		return nil, fmt.Errorf("internal consistency error: original journal %s has insufficient transactions", journalID)
	}

	now := time.Now()
	newJournalID := uuid.NewString()

	// 5. Create the reversing journal domain object
	reversingJournal := domain.Journal{
		JournalID:         newJournalID,
		WorkplaceID:       workplaceID,
		JournalDate:       now, // Use current time for reversal date
		Description:       fmt.Sprintf("Reversal of Journal: %s", originalJournal.JournalID),
		CurrencyCode:      originalJournal.CurrencyCode,
		Status:            domain.Posted,
		OriginalJournalID: &originalJournal.JournalID, // Direct link to the journal being reversed (regardless of whether it's an original or a reversal itself)
		AuditFields: domain.AuditFields{
			CreatedAt:     now,
			CreatedBy:     userID,
			LastUpdatedAt: now,
			LastUpdatedBy: userID,
		},
	}

	// 6. Create reversed transaction domain objects
	reversingTransactions := make([]domain.Transaction, len(originalTransactions))
	accountIDs := make(map[string]struct{})
	for i, origTx := range originalTransactions {
		accountIDs[origTx.AccountID] = struct{}{}
		newTxType := domain.Credit // Flip type
		if origTx.TransactionType == domain.Credit {
			newTxType = domain.Debit
		}

		reversingTransactions[i] = domain.Transaction{
			TransactionID:   uuid.NewString(),
			JournalID:       newJournalID,
			AccountID:       origTx.AccountID,
			Amount:          origTx.Amount, // Amount stays positive
			TransactionType: newTxType,
			CurrencyCode:    origTx.CurrencyCode,
			Notes:           origTx.Notes, // Copy original notes
			AuditFields: domain.AuditFields{
				CreatedAt:     now,
				CreatedBy:     userID,
				LastUpdatedAt: now,
				LastUpdatedBy: userID,
			},
			// RunningBalance will be calculated by SaveJournal repo method
		}
	}

	// 7. Fetch accounts involved to calculate balance changes
	accIDList := make([]string, 0, len(accountIDs))
	for id := range accountIDs {
		accIDList = append(accIDList, id)
	}
	accountsMap, err := j.accountSvc.GetAccountByIDs(ctx, workplaceID, accIDList)
	if err != nil {
		logger.Error("Failed to fetch accounts for reversal balance calculation", "error", err)
		return nil, fmt.Errorf("failed to get account details for reversal: %w", err)
	}

	// Calculate the total amount of the reversal journal using account types
	accountTypes := make(map[string]domain.AccountType)
	for id, acc := range accountsMap {
		accountTypes[id] = acc.AccountType
	}
	totalAmount := j.calculateJournalAmount(reversingTransactions, accountTypes)
	reversingJournal.Amount = totalAmount

	// 8. Calculate balance changes for the reversing journal
	balanceChanges := make(map[string]decimal.Decimal)
	for _, revTx := range reversingTransactions {
		acc, ok := accountsMap[revTx.AccountID]
		if !ok {
			// Should not happen if GetAccountByIDs worked
			logger.Error("Account missing from map during reversal balance calculation", "accountID", revTx.AccountID)
			return nil, fmt.Errorf("internal error: account %s not found during balance calculation", revTx.AccountID)
		}
		signedAmount, err := j.getSignedAmount(revTx, acc.AccountType) // Use existing helper
		if err != nil {
			logger.Error("Failed to calculate signed amount for reversal transaction", "transactionID", revTx.TransactionID, "error", err)
			return nil, fmt.Errorf("failed to calculate signed amount for reversal: %w", err)
		}
		balanceChanges[revTx.AccountID] = balanceChanges[revTx.AccountID].Add(signedAmount)
	}

	// 9. Save the reversing journal and its transactions atomically
	if err := j.journalRepo.SaveJournal(ctx, reversingJournal, reversingTransactions, balanceChanges); err != nil {
		logger.Error("Failed to save reversing journal entry", "error", err)
		return nil, fmt.Errorf("failed to save reversing journal: %w", err)
	}

	// 10. Update the original journal's status and link to the new reversing journal
	// Keep the original journal's OriginalJournalID - preserving the reversal chain
	if err := j.journalRepo.UpdateJournalStatusAndLinks(ctx, originalJournal.JournalID, domain.Reversed, &newJournalID, originalJournal.OriginalJournalID, userID, now); err != nil {
		// Log this error, as the reversal DID succeed, but the linking failed.
		// This is a state inconsistency that might need manual correction or a retry mechanism.
		logger.Error("CRITICAL: Failed to update original journal status after successful reversal", "originalJournalID", originalJournal.JournalID, "reversingJournalID", newJournalID, "error", err)
		// We might still return the reversing journal, but log the critical error.
		// Or return a specific error indicating partial success.
		// For now, let's return the created journal but log heavily.
		// return nil, fmt.Errorf("reversal created but failed to update original journal status: %w", err)
	}

	logger.Info("Journal reversed successfully", "reversingJournalID", newJournalID)
	// Return the newly created reversing journal (without transactions populated)
	reversingJournal.Transactions = nil // Ensure transactions aren't returned by default
	return &reversingJournal, nil
}

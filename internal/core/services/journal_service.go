package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/SscSPs/money_managemet_app/internal/apperrors"
	"github.com/SscSPs/money_managemet_app/internal/core/domain"
	portsrepo "github.com/SscSPs/money_managemet_app/internal/core/ports/repositories"
	portssvc "github.com/SscSPs/money_managemet_app/internal/core/ports/services"
	"github.com/SscSPs/money_managemet_app/internal/dto"
	"github.com/SscSPs/money_managemet_app/internal/middleware"
	"github.com/shopspring/decimal"
)

var (
	ErrJournalUnbalanced  = errors.New("journal entries do not balance to zero")
	ErrJournalMinEntries  = errors.New("journal must have at least two transaction entries")
	ErrJournalMinAccounts = errors.New("journal must affect at least two different accounts")
	ErrAccountNotFound    = errors.New("account not found")
	ErrCurrencyMismatch   = errors.New("account currency does not match journal currency")
	ErrNotPosted          = errors.New("journal must be posted to be updated")
	ErrDescriptionMissing = errors.New("journal description is required")
)

// journalService provides core journal and transaction operations.
type journalService struct {
	accountSvc   portssvc.AccountSvcFacade
	journalRepo  portsrepo.JournalRepositoryWithTx
	workplaceSvc portssvc.WorkplaceSvcFacade // Updated to use WorkplaceSvcFacade
}

// NewJournalService creates a new JournalService.
func NewJournalService(journalRepo portsrepo.JournalRepositoryWithTx, accountSvc portssvc.AccountSvcFacade, workplaceSvc portssvc.WorkplaceSvcFacade) portssvc.JournalSvcFacade {
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
func (s *journalService) validateJournalBalance(transactions []domain.Transaction) error {
	if len(transactions) < 2 {
		return ErrJournalMinEntries
	}

	zero := decimal.NewFromInt(0)

	// For double-entry accounting, the sum of debits should equal the sum of credits
	debitsSum := zero
	creditsSum := zero

	for _, txn := range transactions {
		// Ensure amount is positive (as per PRD)
		if txn.Amount.LessThanOrEqual(zero) {
			return fmt.Errorf("transaction amount must be positive for transaction ID %s", txn.TransactionID)
		}

		// Sum all debits and credits
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
func (s *journalService) calculateJournalAmount(transactions []domain.Transaction) decimal.Decimal {
	if len(transactions) == 0 {
		return decimal.Zero
	}

	// For a balanced journal, the sum of all debit entries equals the sum of all credit entries.
	// This sum represents the total economic value of the journal.
	totalDebits := decimal.Zero
	for _, txn := range transactions {
		if txn.TransactionType == domain.Debit {
			totalDebits = totalDebits.Add(txn.Amount)
		}
	}
	return totalDebits
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

	// Check that transactions involve at least 2 different accounts
	accountSet := make(map[string]bool)
	for _, txn := range req.Transactions {
		accountSet[txn.AccountID] = true
	}
	if len(accountSet) < 2 {
		return nil, ErrJournalMinAccounts
	}

	//the description must not be empty
	if req.Description == "" {
		return nil, ErrDescriptionMissing
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

		// Set transaction date to the provided date or default to journal date
		transactionDate := req.Date
		if txnReq.TransactionDate != nil {
			transactionDate = *txnReq.TransactionDate
		}

		domainTransactions[i] = domain.Transaction{
			TransactionID:   uuid.NewString(),
			JournalID:       journalID, // Link to the new journal
			AccountID:       txnReq.AccountID,
			Amount:          txnReq.Amount,
			TransactionType: txnReq.TransactionType,
			CurrencyCode:    req.CurrencyCode, // Use journal's currency
			Notes:           txnReq.Notes,
			TransactionDate: transactionDate, // Set the transaction date
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

	// Validate Balance (double-entry check)
	if err := s.validateJournalBalance(domainTransactions); err != nil {
		return nil, err
	}

	// --- Fetch Accounts and Validate Further ---
	uniqueAccountIDs := uniqueStrings(accountIDs)
	accountsMap, err := s.accountSvc.GetAccountByIDs(ctx, workplaceID, uniqueAccountIDs, creatorUserID)
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
	totalAmount := s.calculateJournalAmount(domainTransactions)
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
	//add the journal specific details in the transactions
	for i := range transactions {
		transactions[i].JournalID = journalID
		transactions[i].JournalDate = journal.JournalDate
		transactions[i].JournalDescription = journal.Description
	}
	journal.Transactions = transactions

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
	journals, nextToken, err := j.journalRepo.ListJournalsByWorkplace(ctx, workplaceID, params.Limit, params.NextToken, params.IncludeReversals)
	if err != nil {
		logger.Error("Failed to list journals from repository", "error", err)
		return nil, fmt.Errorf("failed to retrieve journals: %w", err)
	}

	// Convert domain journals to DTO responses
	journalResponses := make([]dto.JournalResponse, len(journals))

	// If transactions are requested, fetch them in a batch for all journals
	var transactionsMap map[string][]domain.Transaction
	if params.IncludeTransactions && len(journals) > 0 {
		journalIDs := make([]string, len(journals))
		for i, journal := range journals {
			journalIDs[i] = journal.JournalID
		}
		transactionsMap, err = j.journalRepo.FindTransactionsByJournalIDs(ctx, journalIDs)
		if err != nil {
			logger.Warn("Failed to fetch transactions for journals", "error", err)
			// Continue without transactions rather than failing the whole request
		}
	}

	for i, journal := range journals {
		// Set transactions if they were requested and available
		if transactionsMap != nil {
			if txs, exists := transactionsMap[journal.JournalID]; exists {
				journal.Transactions = txs
			}
		} else {
			journal.Transactions = nil
		}
		journalResponses[i] = dto.ToJournalResponse(&journal)
	}

	// Populate the response DTO with journals and the next token
	resp := &dto.ListJournalsResponse{
		Journals:  journalResponses,
		NextToken: nextToken,
	}

	logger.Info("Journals listed successfully", "count", len(journals), "includeTxn", params.IncludeTransactions)
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

	if journal.Status != domain.Posted {
		return nil, ErrNotPosted
	}

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

func (s *journalService) validateReverseJournalActionAndGetOriginalJournal(ctx context.Context, journalID string, userID string, workplaceID string) (*domain.Journal, []domain.Transaction, error) {
	logger := middleware.GetLoggerFromCtx(ctx)

	// 1. Authorize user action (e.g., require member role)
	if err := s.workplaceSvc.AuthorizeUserAction(ctx, userID, workplaceID, domain.RoleMember); err != nil {
		logger.Warn("Authorization failed for ReverseJournal", "error", err)
		return nil, nil, err // Error already contains appropriate type (e.g., ErrForbidden)
	}

	// 2. Fetch the original journal
	originalJournal, err := s.journalRepo.FindJournalByID(ctx, journalID)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			logger.Warn("Original journal not found for reversal")
			return nil, nil, apperrors.ErrNotFound
		}
		logger.Error("Failed to fetch original journal for reversal", "error", err)
		return nil, nil, fmt.Errorf("failed to retrieve original journal: %w", err)
	}

	// 3. Validate the original journal
	if originalJournal.WorkplaceID != workplaceID {
		logger.Warn("Attempted to reverse journal from wrong workplace")
		return nil, nil, apperrors.ErrNotFound // Treat as not found in this context
	}
	if originalJournal.Status != domain.Posted {
		logger.Warn("Attempted to reverse non-posted journal", "status", originalJournal.Status)
		return nil, nil, fmt.Errorf("%w: journal status is %s, expected POSTED", apperrors.ErrConflict, originalJournal.Status)
	}

	// 4. Prevent reversing a reversal
	if originalJournal.OriginalJournalID != nil {
		logger.Warn("Attempted to reverse a journal that is already a reversal", "journalID", journalID)
		return nil, nil, fmt.Errorf("%w: cannot reverse a journal that is already a reversal", apperrors.ErrConflict)
	}

	// 5. Fetch original transactions
	originalTransactions, err := s.journalRepo.FindTransactionsByJournalID(ctx, journalID)
	if err != nil {
		logger.Error("Failed to fetch original transactions for reversal", "error", err)
		return nil, nil, fmt.Errorf("failed to retrieve original transactions: %w", err)
	}
	return originalJournal, originalTransactions, nil
}

// ReverseJournal creates a new journal entry that reverses a previously posted journal.
// WithTransaction executes the given function within a database transaction.
// It begins a transaction, executes the function, and then commits or rolls back.
func (s *journalService) WithTransaction(ctx context.Context, fn func(txRepo portsrepo.JournalRepositoryWithTx) (interface{}, error)) (interface{}, error) {
	// Check if the repository supports transactions.
	txRepo, ok := s.journalRepo.(portsrepo.JournalRepositoryWithTx)
	if !ok {
		return nil, errors.New("repository does not support transactions")
	}

	// Execute the function with the repository that now has transaction capabilities.
	return fn(txRepo)
}

// ReverseJournal creates a new journal entry that reverses a previously posted journal.
func (s *journalService) ReverseJournal(ctx context.Context, workplaceID string, journalID string, userID string) (*domain.Journal, error) {
	logger := middleware.GetLoggerFromCtx(ctx)

	// Execute the entire reversal process within a single transaction.
	result, err := s.WithTransaction(ctx, func(txRepo portsrepo.JournalRepositoryWithTx) (interface{}, error) {
		// The 'txRepo' is now the interface that can handle transactions.
		// All repository calls within this function will be part of the same transaction.

		originalJournal, originalTransactions, err := s.validateReverseJournalActionAndGetOriginalJournal(ctx, journalID, userID, workplaceID)
		if err != nil {
			return nil, err
		}

		now := time.Now()
		newJournalID := uuid.NewString()

		// Create the reversing journal domain object.
		reversingJournal := domain.Journal{
			JournalID:    newJournalID,
			WorkplaceID:  workplaceID,
			JournalDate:  originalJournal.JournalDate,
			CurrencyCode: originalJournal.CurrencyCode,
			Status:       domain.Posted,
			AuditFields: domain.AuditFields{
				CreatedAt:     now,
				CreatedBy:     userID,
				LastUpdatedAt: now,
				LastUpdatedBy: userID,
			},
		}

		isReversingAReversal := originalJournal.OriginalJournalID != nil
		if isReversingAReversal {
			reversingJournal.Description = strings.TrimPrefix(originalJournal.Description, "Reversal of Journal: ")
		} else {
			reversingJournal.OriginalJournalID = &originalJournal.JournalID
			reversingJournal.Description = fmt.Sprintf("Reversal of Journal: %s", originalJournal.Description)
		}

		// Create reversed transaction domain objects.
		reversingTransactions := make([]domain.Transaction, len(originalTransactions))
		accIDList := make([]string, 0)
		for i, origTx := range originalTransactions {
			accIDList = append(accIDList, origTx.AccountID)
			newTxType := domain.Credit
			if origTx.TransactionType == domain.Credit {
				newTxType = domain.Debit
			}
			reversingTransactions[i] = domain.Transaction{
				TransactionID:   uuid.NewString(),
				JournalID:       newJournalID,
				AccountID:       origTx.AccountID,
				Amount:          origTx.Amount,
				TransactionType: newTxType,
				CurrencyCode:    origTx.CurrencyCode,
				Notes:           origTx.Notes,
				AuditFields: domain.AuditFields{
					CreatedAt:     now,
					CreatedBy:     userID,
					LastUpdatedAt: now,
					LastUpdatedBy: userID,
				},
			}
		}

		accountsMap, err := s.accountSvc.GetAccountByIDs(ctx, workplaceID, accIDList, userID)
		if err != nil {
			logger.Error("Failed to fetch accounts for reversal balance calculation", "error", err)
			return nil, fmt.Errorf("failed to get account details for reversal: %w", err)
		}

		reversingJournal.Amount = originalJournal.Amount

		balanceChanges := make(map[string]decimal.Decimal)
		for _, revTx := range reversingTransactions {
			acc, ok := accountsMap[revTx.AccountID]
			if !ok {
				logger.Error("Account missing from map during reversal balance calculation", "accountID", revTx.AccountID)
				return nil, fmt.Errorf("internal error: account %s not found during balance calculation", revTx.AccountID)
			}
			signedAmount, err := s.getSignedAmount(revTx, acc.AccountType)
			if err != nil {
				logger.Error("Failed to calculate signed amount for reversal transaction", "transactionID", revTx.TransactionID, "error", err)
				return nil, fmt.Errorf("failed to calculate signed amount for reversal: %w", err)
			}
			balanceChanges[revTx.AccountID] = balanceChanges[revTx.AccountID].Add(signedAmount)
		}

		// Save the reversing journal and update the original journal's status atomically.
		if err := txRepo.SaveJournal(ctx, reversingJournal, reversingTransactions, balanceChanges); err != nil {
			logger.Error("Failed to save reversing journal entry", "error", err)
			return nil, fmt.Errorf("failed to save reversing journal: %w", err)
		}

		if !isReversingAReversal {
			if err := txRepo.UpdateJournalStatusAndLinks(ctx, originalJournal.JournalID, domain.Reversed, &newJournalID, originalJournal.OriginalJournalID, userID, now); err != nil {
				logger.Error("Failed to update original journal status after successful reversal", "originalJournalID", originalJournal.JournalID, "reversingJournalID", newJournalID, "error", err)
				return nil, fmt.Errorf("failed to update original journal status: %w", err)
			}
		}

		logger.Info("Journal reversed successfully", "reversingJournalID", newJournalID)
		reversingJournal.Transactions = nil
		return &reversingJournal, nil
	})

	if err != nil {
		return nil, err
	}

	return result.(*domain.Journal), nil
}

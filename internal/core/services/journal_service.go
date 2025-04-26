package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/SscSPs/money_managemet_app/internal/apperrors"
	"github.com/SscSPs/money_managemet_app/internal/core/domain"
	portsrepo "github.com/SscSPs/money_managemet_app/internal/core/ports/repositories"
	"github.com/SscSPs/money_managemet_app/internal/dto"
	"github.com/SscSPs/money_managemet_app/internal/models"
	"github.com/shopspring/decimal"
	// We might need a UUID library later, e.g., "github.com/google/uuid"
)

var (
	ErrJournalUnbalanced = errors.New("journal entries do not balance to zero")
	ErrJournalMinEntries = errors.New("journal must have at least two transaction entries")
	ErrAccountNotFound   = errors.New("account not found")
	ErrCurrencyMismatch  = errors.New("transactions must use the journal's currency")
)

// JournalService provides core journal and transaction operations.
type JournalService struct {
	accountRepo portsrepo.AccountRepository
	journalRepo portsrepo.JournalRepository
	// userRepo portsrepo.UserRepository // Needed later for CreatedBy/UpdatedBy
}

// NewJournalService creates a new JournalService.
func NewJournalService(accountRepo portsrepo.AccountRepository, journalRepo portsrepo.JournalRepository) *JournalService {
	return &JournalService{
		accountRepo: accountRepo,
		journalRepo: journalRepo,
	}
}

// getSignedAmount applies the correct sign to a transaction amount based on account type and transaction type.
func (s *JournalService) getSignedAmount(txn domain.Transaction, accountType domain.AccountType) (decimal.Decimal, error) {
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
func (s *JournalService) validateJournalBalance(transactions []domain.Transaction, accountTypes map[string]domain.AccountType) error {
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

// PersistJournal creates a new journal entry with its transactions after validation.
func (s *JournalService) PersistJournal(ctx context.Context, req dto.CreateJournalAndTxn, userID string) (*domain.Journal, error) {
	// --- Use models from DTO ---
	modelJournal := req.Journal
	modelTransactions := req.Transactions

	// --- Validation ---
	if len(modelTransactions) < 2 { // Validate against original DTO models
		return nil, ErrJournalMinEntries
	}

	// --- Currency Validation ---
	for _, txn := range modelTransactions {
		if txn.CurrencyCode != modelJournal.CurrencyCode {
			// TODO: Log this validation failure
			return nil, fmt.Errorf("%w: journal is %s, transaction involves %s",
				ErrCurrencyMismatch, modelJournal.CurrencyCode, txn.CurrencyCode)
		}
	}

	// --- Convert models to domain for further validation ---
	domainTransactions := modelToDomainTransactions(modelTransactions)

	// --- Fetch Account Types and Validate Accounts ---
	accountIDs := make([]string, 0, len(domainTransactions))
	for _, txn := range domainTransactions { // Iterate domain transactions
		accountIDs = append(accountIDs, txn.AccountID)
	}
	uniqueAccountIDs := uniqueStrings(accountIDs)

	// Fetch accounts in batch
	accountsMap, err := s.accountRepo.FindAccountsByIDs(ctx, uniqueAccountIDs)
	if err != nil {
		// Handle potential repository error during batch fetch
		return nil, fmt.Errorf("failed to fetch accounts: %w", err)
	}

	// Check if all required accounts were found and active, and gather types
	accountTypes := make(map[string]domain.AccountType)
	for _, id := range uniqueAccountIDs {
		acc, found := accountsMap[id]
		if !found {
			return nil, fmt.Errorf("%w: ID %s", ErrAccountNotFound, id)
		}
		if !acc.IsActive {
			return nil, fmt.Errorf("account %s is inactive", id)
		}
		accountTypes[id] = acc.AccountType
	}

	// Validate Balance (Uses domain transactions now)
	if err = s.validateJournalBalance(domainTransactions, accountTypes); err != nil { // Assign err to existing var
		return nil, err
	}

	// --- Persistence --- (Prepare domain objects for saving)
	creatorUserID := userID
	now := time.Now().UTC()

	// Create final domain journal object
	domainJournal := domain.Journal{
		JournalID:    uuid.NewString(),
		JournalDate:  modelJournal.JournalDate,
		Description:  modelJournal.Description,
		CurrencyCode: modelJournal.CurrencyCode,
		Status:       domain.Posted,
		AuditFields: domain.AuditFields{
			CreatedAt:     now,
			CreatedBy:     creatorUserID,
			LastUpdatedAt: now,
			LastUpdatedBy: creatorUserID,
		},
	}

	// Populate remaining fields in domain transactions
	for i := range domainTransactions {
		domainTransactions[i].TransactionID = uuid.NewString()
		domainTransactions[i].JournalID = domainJournal.JournalID
		domainTransactions[i].CurrencyCode = domainJournal.CurrencyCode // Assign journal currency
		// Assign AuditFields to transactions
		domainTransactions[i].AuditFields = domain.AuditFields{
			CreatedAt:     now,
			CreatedBy:     creatorUserID,
			LastUpdatedAt: now,
			LastUpdatedBy: creatorUserID,
		}
	}

	// Save atomically via repository (Repo expects domain types)
	err = s.journalRepo.SaveJournal(ctx, domainJournal, domainTransactions)
	if err != nil {
		return nil, fmt.Errorf("failed to save journal: %w", err)
	}

	return &domainJournal, nil // Return the created domain journal
}

// GetJournalWithTransactions retrieves a specific journal and its associated transaction lines.
func (s *JournalService) GetJournalWithTransactions(ctx context.Context, journalID string) (*domain.Journal, []domain.Transaction, error) {
	journal, err := s.journalRepo.FindJournalByID(ctx, journalID)
	if err != nil {
		// TODO: Handle specific 'not found' errors from repo if available
		return nil, nil, fmt.Errorf("failed to find journal by ID %s: %w", journalID, err)
	}
	if journal == nil { // Explicit check if repo returns nil on not found
		return nil, nil, fmt.Errorf("journal with ID %s not found", journalID)
	}

	transactions, err := s.journalRepo.FindTransactionsByJournalID(ctx, journalID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to find transactions for journal ID %s: %w", journalID, err)
	}

	return journal, transactions, nil
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

// CalculateAccountBalance calculates the current balance of a given account.
func (s *JournalService) CalculateAccountBalance(ctx context.Context, accountID string) (decimal.Decimal, error) {
	// 1. Find the account to verify existence, activity, and get type
	account, err := s.accountRepo.FindAccountByID(ctx, accountID)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			return decimal.Zero, fmt.Errorf("%w: ID %s", ErrAccountNotFound, accountID)
		}
		return decimal.Zero, fmt.Errorf("failed to find account %s: %w", accountID, err)
	}
	if !account.IsActive {
		return decimal.Zero, fmt.Errorf("account %s is inactive", accountID)
	}

	// 2. Fetch all transactions for this account
	transactions, err := s.journalRepo.FindTransactionsByAccountID(ctx, accountID)
	if err != nil {
		// Handle potential repository error during transaction fetch
		return decimal.Zero, fmt.Errorf("failed to fetch transactions for account %s: %w", accountID, err)
	}

	// 3. Calculate the balance by summing signed amounts
	balance := decimal.Zero
	for _, txn := range transactions {
		// Ensure transaction amount is positive (should be guaranteed by PersistJournal, but double-check)
		if txn.Amount.LessThanOrEqual(decimal.Zero) {
			// Log this inconsistency, but potentially continue calculation?
			// For now, return an error as it indicates a data issue.
			return decimal.Zero, fmt.Errorf("invalid non-positive transaction amount found (ID: %s) for account %s", txn.TransactionID, accountID)
		}

		// Transaction currency should ideally match account currency, though the current
		// service doesn't explicitly enforce/handle multi-currency aggregation here.
		// Assuming all fetched transactions are in the account's currency for now.

		signedAmount, err := s.getSignedAmount(txn, account.AccountType)
		if err != nil {
			// Propagate error from getSignedAmount (e.g., unknown account type)
			return decimal.Zero, fmt.Errorf("error calculating signed amount for transaction %s: %w", txn.TransactionID, err)
		}
		balance = balance.Add(signedAmount)
	}

	return balance, nil
}

// TODO: Add methods for:
// - FindJournalByID(ctx context.Context, journalID string) (*models.Journal, []models.Transaction, error)
// - Static data initialization trigger? (FR-M1-06)

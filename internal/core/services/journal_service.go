package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/SscSPs/money_managemet_app/internal/core/ports"
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
	accountRepo ports.AccountRepository
	journalRepo ports.JournalRepository
	// userRepo ports.UserRepository // Needed later for CreatedBy/UpdatedBy
}

// NewJournalService creates a new JournalService.
func NewJournalService(
	accountRepo ports.AccountRepository,
	journalRepo ports.JournalRepository,
) *JournalService {
	return &JournalService{
		accountRepo: accountRepo,
		journalRepo: journalRepo,
	}
}

// getSignedAmount applies the correct sign to a transaction amount based on account type and transaction type.
func (s *JournalService) getSignedAmount(txn models.Transaction, accountType models.AccountType) (decimal.Decimal, error) {
	signedAmount := txn.Amount
	isDebit := txn.TransactionType == models.Debit

	// Determine sign based on convention (PRD FR-M1-03)
	// DEBIT to ASSET/EXPENSE -> Positive (+)
	// CREDIT to ASSET/EXPENSE -> Negative (-)
	// DEBIT to LIABILITY/EQUITY/INCOME -> Negative (-)
	// CREDIT to LIABILITY/EQUITY/INCOME -> Positive (+)
	switch accountType {
	case models.Asset, models.Expense:
		if !isDebit { // Credit to Asset/Expense
			signedAmount = signedAmount.Neg()
		}
	case models.Liability, models.Equity, models.Income:
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
// It requires a map of account IDs to their AccountType for determining debit/credit signs.
func (s *JournalService) validateJournalBalance(transactions []models.Transaction, accountTypes map[string]models.AccountType) error {
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

// PersistJournal creates a new journal entry with its transactions after validation.
// It ensures the journal balances and adheres to other M1 rules.
// TODO: Add UserID parameter for audit fields once user context is available.
func (s *JournalService) PersistJournal(ctx context.Context, journal models.Journal, transactions []models.Transaction, userID string) (*models.Journal, error) {
	// --- Validation ---
	// 1. Check minimum entries
	if len(transactions) < 2 {
		return nil, ErrJournalMinEntries
	}

	// 2. Fetch Account Types needed for balance validation
	accountIDs := make([]string, 0, len(transactions))
	accountTypes := make(map[string]models.AccountType)
	for _, txn := range transactions {
		accountIDs = append(accountIDs, txn.AccountID)
		// 3. Validate Currency Match (MVP Constraint)
		if txn.CurrencyCode != journal.CurrencyCode {
			return nil, fmt.Errorf("%w: journal is %s, transaction %s is %s",
				ErrCurrencyMismatch, journal.CurrencyCode, txn.TransactionID, txn.CurrencyCode)
		}
	}
	// TODO: Ideally fetch accounts in a single query if repository supports FindAccountsByIDs
	for _, id := range uniqueStrings(accountIDs) {
		acc, err := s.accountRepo.FindAccountByID(ctx, id)
		if err != nil {
			// Handle specific errors if repo provides them (e.g., ErrNotFound)
			return nil, fmt.Errorf("failed to find account %s: %w", id, err)
		}
		if acc == nil {
			return nil, fmt.Errorf("%w: ID %s", ErrAccountNotFound, id)
		}
		if !acc.IsActive {
			return nil, fmt.Errorf("account %s is inactive", id)
		}
		accountTypes[id] = acc.AccountType
	}

	// 4. Validate Balance
	if err := s.validateJournalBalance(transactions, accountTypes); err != nil {
		return nil, err
	}

	// --- Persistence ---
	// Populate Audit Fields & Defaults
	// TODO: Get actual UserID
	creatorUserID := userID
	now := time.Now().UTC()

	journal.JournalID = uuid.NewString() // Example if using UUIDs
	journal.Status = models.Posted       // Ensure default status
	journal.CreatedAt = now
	journal.CreatedBy = creatorUserID
	journal.LastUpdatedAt = now
	journal.LastUpdatedBy = creatorUserID

	for i := range transactions {
		// Assign Transaction ID if not provided
		transactions[i].TransactionID = uuid.NewString()
		// Link transaction to journal
		transactions[i].JournalID = journal.JournalID
		// Ensure currency matches journal (already validated, but good practice)
		transactions[i].CurrencyCode = journal.CurrencyCode
		// Set audit fields
		transactions[i].CreatedAt = now
		transactions[i].CreatedBy = creatorUserID
		transactions[i].LastUpdatedAt = now
		transactions[i].LastUpdatedBy = creatorUserID
	}

	// Save atomically via repository
	err := s.journalRepo.SaveJournal(ctx, journal, transactions)
	if err != nil {
		return nil, fmt.Errorf("failed to save journal: %w", err)
	}

	// Return the potentially updated journal (e.g., with generated ID)
	return &journal, nil
}

// GetJournalWithTransactions retrieves a specific journal and its associated transaction lines.
func (s *JournalService) GetJournalWithTransactions(ctx context.Context, journalID string) (*models.Journal, []models.Transaction, error) {
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

// Helper function (consider moving to a utils package)
func uniqueStrings(input []string) []string {
	seen := make(map[string]struct{}, len(input))
	j := 0
	for _, v := range input {
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		input[j] = v
		j++
	}
	return input[:j]
}

// CalculateAccountBalance computes the current balance for a given account.
func (s *JournalService) CalculateAccountBalance(ctx context.Context, accountID string) (decimal.Decimal, error) {
	// 1. Fetch the account details
	account, err := s.accountRepo.FindAccountByID(ctx, accountID)
	if err != nil {
		// Handle repository errors, potentially mapping specific ones like 'not found'
		// For now, return a generic error
		return decimal.Zero, fmt.Errorf("failed to find account %s: %w", accountID, err)
	}
	if account == nil {
		// If repo returns (nil, nil) for not found
		return decimal.Zero, fmt.Errorf("%w: ID %s", ErrAccountNotFound, accountID)
	}
	// Optionally check if account is inactive? PRD doesn't specify for balance calc.

	// 2. Fetch all transactions for this account
	transactions, err := s.journalRepo.FindTransactionsByAccountID(ctx, accountID)
	if err != nil {
		// Handle repository errors
		return decimal.Zero, fmt.Errorf("failed to find transactions for account %s: %w", accountID, err)
	}

	// 3. Calculate the balance by summing signed transaction amounts
	balance := decimal.NewFromInt(0)
	for _, txn := range transactions {
		// Ensure the transaction amount itself is valid before processing
		// (Note: validateJournalBalance already checks this for *new* transactions)
		if txn.Amount.LessThanOrEqual(decimal.Zero) {
			// Log this potential data issue?
			// For now, skip this transaction in the balance calculation
			// Or return an error? Let's skip for now.
			fmt.Printf("Warning: Skipping transaction %s for account %s due to non-positive amount: %s\\n", txn.TransactionID, accountID, txn.Amount.String())
			continue
		}

		// Use the refactored helper to get the signed amount
		signedAmount, err := s.getSignedAmount(txn, account.AccountType)
		if err != nil {
			// This indicates a problem (e.g., unknown account type was stored)
			// Log the error and potentially continue or return? Returning error seems safer.
			return decimal.Zero, fmt.Errorf("error calculating balance for account %s due to transaction %s: %w", accountID, txn.TransactionID, err)
		}

		balance = balance.Add(signedAmount)
	}

	return balance, nil
}

// TODO: Add methods for:
// - FindJournalByID(ctx context.Context, journalID string) (*models.Journal, []models.Transaction, error)
// - Static data initialization trigger? (FR-M1-06)

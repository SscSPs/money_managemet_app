package accounting

import (
	"fmt"

	"github.com/SscSPs/money_managemet_app/internal/core/domain"
	"github.com/shopspring/decimal"
)

// CalculateSignedAmount applies the correct sign to a transaction amount based on account type and transaction type.
// This is used in both services and repositories to ensure consistent accounting logic.
func CalculateSignedAmount(txn domain.Transaction, accountType domain.AccountType) (decimal.Decimal, error) {
	signedAmount := txn.Amount
	isDebit := txn.TransactionType == domain.Debit

	// Determine sign based on accounting convention
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
		if isDebit { // Debit to Liability/Equity/Revenue
			signedAmount = signedAmount.Neg()
		}
	default:
		return decimal.Zero, fmt.Errorf("unknown account type '%s' encountered for account ID %s", accountType, txn.AccountID)
	}
	return signedAmount, nil
}

// CalculateJournalAmount computes the true economic value of a journal.
// For a balanced journal with equal debit and credit sides,
// we need to pick one side that represents the actual money movement.
func CalculateJournalAmount(transactions []domain.Transaction, accountTypes map[string]domain.AccountType) decimal.Decimal {
	if len(transactions) == 0 {
		return decimal.Zero
	}

	// In double-entry accounting, for a balanced journal:
	// - The sum of all debits equals the sum of all credits
	// - The economic value of the transaction is the amount that changed hands

	// We'll calculate the sum of:
	// - DEBIT amounts for ASSET and EXPENSE accounts (money spent/used)
	// - CREDIT amounts for LIABILITY, EQUITY, and REVENUE accounts (money received/gained)

	totalAmount := decimal.Zero
	for _, txn := range transactions {
		accountType, exists := accountTypes[txn.AccountID]
		if !exists {
			// Skip if account type isn't provided (this should be caught in validation)
			continue
		}

		isDebit := txn.TransactionType == domain.Debit

		// Only count the transaction if it represents money movement in the positive direction
		switch accountType {
		case domain.Asset, domain.Expense:
			if isDebit {
				totalAmount = totalAmount.Add(txn.Amount)
			}
		case domain.Liability, domain.Equity, domain.Revenue:
			if !isDebit { // Credit
				totalAmount = totalAmount.Add(txn.Amount)
			}
		}
	}

	return totalAmount
}

// CalculateJournalAmountSimple is a simplified version that just returns the first transaction amount
// when account types aren't available.
func CalculateJournalAmountSimple(transactions []domain.Transaction) decimal.Decimal {
	if len(transactions) > 0 {
		return transactions[0].Amount
	}
	return decimal.Zero
}

// ValidateJournalBalance checks if the transactions for a journal balance to zero.
func ValidateJournalBalance(transactions []domain.Transaction, accountTypes map[string]domain.AccountType) error {
	if len(transactions) < 2 {
		return fmt.Errorf("journal must have at least two transaction entries")
	}

	zero := decimal.NewFromInt(0)
	sum := zero

	for _, txn := range transactions {
		// Ensure amount is positive
		if txn.Amount.LessThanOrEqual(zero) {
			return fmt.Errorf("transaction amount must be positive for transaction ID %s", txn.TransactionID)
		}

		accountType, ok := accountTypes[txn.AccountID]
		if !ok {
			return fmt.Errorf("account type not found for account ID %s", txn.AccountID)
		}

		signedAmount, err := CalculateSignedAmount(txn, accountType)
		if err != nil {
			return fmt.Errorf("error calculating signed amount for transaction %s: %w", txn.TransactionID, err)
		}

		sum = sum.Add(signedAmount)
	}

	if !sum.Equal(zero) {
		return fmt.Errorf("journal entries do not balance to zero: sum is %s", sum.String())
	}

	return nil
}

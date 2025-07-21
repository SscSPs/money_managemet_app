package domain

import (
	"fmt"
	"time"

	"github.com/SscSPs/money_managemet_app/internal/apperrors"
	"github.com/shopspring/decimal"
)

// TransactionType indicates whether a transaction line is a Debit or a Credit.
type TransactionType string

const (
	Debit  TransactionType = "DEBIT"
	Credit TransactionType = "CREDIT"
)

// Transaction represents a single line item within a Journal, affecting one account.
// Transaction represents a single line item within a Journal, affecting one account.
type Transaction struct {
	TransactionID    string           `json:"transactionID"`              // Primary Key (e.g., UUID)
	JournalID        string           `json:"journalID"`                  // FK -> Journal.journalID (Not Null)
	AccountID        string           `json:"accountID"`                  // FK -> Account.accountID (Not Null)
	Amount           decimal.Decimal  `json:"amount"`                     // Amount in journal's base currency (always positive)
	OriginalAmount   *decimal.Decimal `json:"originalAmount,omitempty"`   // Original amount in transaction's currency (if different from journal)
	OriginalCurrency *string          `json:"originalCurrency,omitempty"` // Original currency code (if different from journal)
	ExchangeRateID   *string          `json:"exchangeRateId,omitempty"`   // Reference to exchange rate used for conversion
	TransactionType  TransactionType  `json:"transactionType"`            // DEBIT or CREDIT (Not Null)
	CurrencyCode     string           `json:"currencyCode"`               // Journal's base currency (Not Null)
	Notes            string           `json:"notes"`                      // Nullable
	TransactionDate  time.Time        `json:"transactionDate"`            // Date of the transaction (may differ from journal date)
	AuditFields
	// RunningBalance represents the balance of the AccountID *after* this transaction was applied.
	// This needs to be calculated and stored by the repository during SaveJournal.
	RunningBalance     decimal.Decimal `json:"runningBalance"`
	JournalDate        time.Time       `json:"journalDate"`
	JournalDescription string          `json:"journalDescription"`
}

// IsMultiCurrency checks if the transaction is a multi-currency transaction.
// A transaction is considered multi-currency if it has both OriginalAmount and OriginalCurrency set.
func (t Transaction) IsMultiCurrency() bool {
	return t.OriginalAmount != nil && t.OriginalCurrency != nil && *t.OriginalCurrency != ""
}

// Validate performs validation on the transaction fields.
// It returns an error if the transaction is not valid.
func (t Transaction) Validate() error {
	// Basic validation for required fields
	if t.TransactionID == "" {
		return apperrors.NewValidationError("transaction ID is required")
	}
	if t.JournalID == "" {
		return apperrors.NewValidationError("journal ID is required")
	}
	if t.AccountID == "" {
		return apperrors.NewValidationError("account ID is required")
	}
	if t.CurrencyCode == "" {
		return apperrors.NewValidationError("currency code is required")
	}

	// Validate amount is positive
	if t.Amount.LessThanOrEqual(decimal.Zero) {
		return apperrors.NewValidationError("amount must be positive")
	}

	// Validate transaction type
	if t.TransactionType != Debit && t.TransactionType != Credit {
		return apperrors.NewValidationError("invalid transaction type")
	}

	// Multi-currency specific validations
	if t.IsMultiCurrency() {
		// Check exchange rate ID is provided
		if t.ExchangeRateID == nil || *t.ExchangeRateID == "" {
			return apperrors.NewValidationError("exchange rate ID is required for multi-currency transactions")
		}

		// Check original amount is positive
		if t.OriginalAmount.LessThanOrEqual(decimal.Zero) {
			return apperrors.NewValidationError("original amount must be positive")
		}

		// Check original currency is valid (3-letter ISO code)
		if len(*t.OriginalCurrency) != 3 {
			return apperrors.NewValidationError("original currency must be a 3-letter ISO code")
		}

		// Ensure original currency is different from the transaction currency
		if *t.OriginalCurrency == t.CurrencyCode {
			return apperrors.NewValidationError("original currency must be different from transaction currency")
		}
	} else {
		// If not multi-currency, ensure no exchange rate ID is provided
		if t.ExchangeRateID != nil && *t.ExchangeRateID != "" {
			return apperrors.NewValidationError("exchange rate ID should not be provided for single-currency transactions")
		}
	}

	// Validate transaction date is not zero
	if t.TransactionDate.IsZero() {
		return apperrors.NewValidationError("transaction date is required")
	}

	return nil
}

// GetSignedAmount returns the signed amount based on the transaction type.
// For Debit transactions, it returns positive amount; for Credit, negative.
func (t Transaction) GetSignedAmount() decimal.Decimal {
	if t.TransactionType == Debit {
		return t.Amount
	}
	return t.Amount.Neg()
}

// ConvertToBaseCurrency converts the original amount to the base currency using the provided rate.
// It updates the Amount field and sets the ExchangeRateID.
func (t *Transaction) ConvertToBaseCurrency(rate decimal.Decimal, rateID string) error {
	if !t.IsMultiCurrency() {
		return fmt.Errorf("cannot convert single-currency transaction")
	}

	if rate.LessThanOrEqual(decimal.Zero) {
		return fmt.Errorf("exchange rate must be positive")
	}

	// Calculate the amount in base currency
	t.Amount = t.OriginalAmount.Mul(rate).Round(2) // Round to 2 decimal places for currency
	t.ExchangeRateID = &rateID

	return nil
}

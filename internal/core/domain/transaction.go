package domain

import (
	"time"

	"github.com/shopspring/decimal"
)

// TransactionType indicates whether a transaction line is a Debit or a Credit.
type TransactionType string

const (
	Debit  TransactionType = "DEBIT"
	Credit TransactionType = "CREDIT"
)

// Transaction represents a single line item within a Journal, affecting one account.
type Transaction struct {
	TransactionID   string          `json:"transactionID"`   // Primary Key (e.g., UUID)
	JournalID       string          `json:"journalID"`       // FK -> Journal.journalID (Not Null)
	AccountID       string          `json:"accountID"`       // FK -> Account.accountID (Not Null)
	Amount          decimal.Decimal `json:"amount"`          // Positive value; Precise decimal type
	TransactionType TransactionType `json:"transactionType"` // DEBIT or CREDIT (Not Null)
	CurrencyCode    string          `json:"currencyCode"`    // Must match Journal currency (Not Null)
	Notes           string          `json:"notes"`           // Nullable
	AuditFields
	// RunningBalance represents the balance of the AccountID *after* this transaction was applied.
	// This needs to be calculated and stored by the repository during SaveJournal.
	// Note: Database schema (transactions table) needs a corresponding 'running_balance' column.
	RunningBalance     decimal.Decimal `json:"runningBalance"`
	JournalDate        time.Time       `json:"journalDate"`
	JournalDescription string          `json:"journalDescription"`
}

package models

import "github.com/shopspring/decimal"

// TransactionType indicates whether a transaction line is a Debit or a Credit.
type TransactionType string

const (
	Debit  TransactionType = "DEBIT"
	Credit TransactionType = "CREDIT"
)

// Transaction represents a single line item within a Journal, affecting one account.
// Note: Amount should use a precise decimal type like github.com/shopspring/decimal
type Transaction struct {
	TransactionID   string          `json:"transactionID"`   // Primary Key (e.g., UUID)
	JournalID       string          `json:"journalID"`       // FK -> Journal.journalID (Not Null)
	AccountID       string          `json:"accountID"`       // FK -> Account.accountID (Not Null)
	Amount          decimal.Decimal `json:"amount"`          // Positive value; Precise decimal type
	TransactionType TransactionType `json:"transactionType"` // DEBIT or CREDIT (Not Null)
	CurrencyCode    string          `json:"currencyCode"`    // Must match Journal currency (Not Null)
	Notes           string          `json:"notes"`           // Nullable
	AuditFields
	RunningBalance decimal.Decimal `json:"runningBalance"` // Added: Balance after this transaction
}

package domain

import "github.com/shopspring/decimal"

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
}

package models

import (
	"github.com/shopspring/decimal"
)

// AccountType defines the fundamental accounting type of an account.
type AccountType string

const (
	Asset     AccountType = "ASSET"
	Liability AccountType = "LIABILITY"
	Equity    AccountType = "EQUITY"
	Revenue   AccountType = "REVENUE"
	Expense   AccountType = "EXPENSE"
)

// Account represents a financial account within the ledger.
// Note: ParentAccountID uses string for nullable foreign key; DB handling may vary.
type Account struct {
	AccountID       string          `db:"account_id"`
	WorkplaceID     string          `db:"workplace_id"` // Added workplace_id
	CFID            string          `db:"cfid"`         // Customer Facing ID (optional, user-defined)
	Name            string          `db:"name"`
	AccountType     AccountType     `db:"account_type"` // Use type from common.go
	CurrencyCode    string          `db:"currency_code"`
	ParentAccountID string          `db:"parent_account_id"` // Nullable
	Description     string          `db:"description"`
	IsActive        bool            `db:"is_active"`
	AuditFields                     // Embed common audit fields
	Balance         decimal.Decimal `db:"balance"` // Added: Persisted account balance
}

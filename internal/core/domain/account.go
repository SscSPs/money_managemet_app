package domain

import (
	"github.com/shopspring/decimal"
)

// AccountType defines the fundamental accounting type of an account.
type AccountType string

const (
	Asset     AccountType = "ASSET"
	Liability AccountType = "LIABILITY"
	Equity    AccountType = "EQUITY"
	Income    AccountType = "INCOME"
	Expense   AccountType = "EXPENSE"
)

// Account represents a financial account within the core domain.
// This is the primary representation used by services.
type Account struct {
	AccountID       string          `json:"accountID"`       // Primary Key (e.g., UUID)
	WorkplaceID     string          `json:"workplaceID"`     // FK -> workplaces.workplace_id (NON-NULL)
	Name            string          `json:"name"`            // User-defined name
	AccountType     AccountType     `json:"accountType"`     // ASSET, LIABILITY, etc.
	CurrencyCode    string          `json:"currencyCode"`    // FK -> currencies.code (NON-NULL)
	ParentAccountID string          `json:"parentAccountID"` // Nullable FK -> accounts.account_id (Self-referencing)
	Description     string          `json:"description"`     // Nullable user description
	IsActive        bool            `json:"isActive"`        // Soft delete or status flag
	AuditFields                     // Embed CreatedAt, CreatedBy, etc.
	Balance         decimal.Decimal `json:"balance"` // Added: Persisted account balance
}

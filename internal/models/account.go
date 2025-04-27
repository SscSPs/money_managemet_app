package models

// AccountType defines the fundamental accounting type of an account.
type AccountType string

const (
	Asset     AccountType = "ASSET"
	Liability AccountType = "LIABILITY"
	Equity    AccountType = "EQUITY"
	Income    AccountType = "INCOME"
	Expense   AccountType = "EXPENSE"
)

// Account represents a financial account within the ledger.
// Note: ParentAccountID uses string for nullable foreign key; DB handling may vary.
type Account struct {
	AccountID       string      `json:"accountID"`       // Primary Key (e.g., UUID)
	Name            string      `json:"name"`            // User-defined name
	AccountType     AccountType `json:"accountType"`     // ASSET, LIABILITY, etc.
	CurrencyCode    string      `json:"currencyCode"`    // FK -> Currency.currencyCode (Not Null)
	ParentAccountID string      `json:"parentAccountID"` // Nullable FK -> Account.accountID
	Description     string      `json:"description"`     // Nullable
	IsActive        bool        `json:"isActive"`        // Default: true
	WorkplaceID     string      `json:"workplaceID"`     // Added workplace_id
	AuditFields
}

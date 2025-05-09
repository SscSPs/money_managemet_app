package domain

import (
	"time"

	"github.com/shopspring/decimal"
)

// JournalStatus indicates the state of a journal entry.
type JournalStatus string

const (
	Posted   JournalStatus = "POSTED"
	Reversed JournalStatus = "REVERSED"
)

// Journal represents a single, balanced financial event composed of multiple transactions.
type Journal struct {
	JournalID          string          `json:"journalID"`                    // Primary Key (e.g., UUID)
	WorkplaceID        string          `json:"workplaceID"`                  // FK -> workplaces.workplace_id (NON-NULL)
	JournalDate        time.Time       `json:"journalDate"`                  // Date the event occurred
	Description        string          `json:"description"`                  // Nullable user description
	CurrencyCode       string          `json:"currencyCode"`                 // Primary currency of the Journal (Not Null)
	Status             JournalStatus   `json:"status"`                       // Default: Posted
	Transactions       []Transaction   `json:"transactions,omitempty"`       // Added: Holds associated transactions when loaded
	OriginalJournalID  *string         `json:"originalJournalID,omitempty"`  // Link to the journal this one reverses
	ReversingJournalID *string         `json:"reversingJournalID,omitempty"` // Link to the journal that reverses this one
	Amount             decimal.Decimal `json:"amount,omitempty"`             // Total amount of movement (sum of debits or credits)
	AuditFields
}

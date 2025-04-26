package domain

import "time"

// JournalStatus indicates the state of a journal entry.
type JournalStatus string

const (
	Posted   JournalStatus = "POSTED"
	Reversed JournalStatus = "REVERSED"
)

// Journal represents a single, balanced financial event composed of multiple transactions.
type Journal struct {
	JournalID    string        `json:"journalID"`    // Primary Key (e.g., UUID)
	JournalDate  time.Time     `json:"journalDate"`  // Date the event occurred
	Description  string        `json:"description"`  // Nullable user description
	CurrencyCode string        `json:"currencyCode"` // Primary currency of the Journal (Not Null)
	Status       JournalStatus `json:"status"`       // Default: Posted
	AuditFields
}

package models

import "time"

// JournalStatus indicates the state of a journal entry.
type JournalStatus string

const (
	Posted   JournalStatus = "POSTED"
	Reversed JournalStatus = "REVERSED"
	// Maybe DRAFT later?
)

// Journal represents the database model for a journal entry.
type Journal struct {
	JournalID          string        `db:"journal_id"`
	WorkplaceID        string        `db:"workplace_id"` // Added workplace_id
	JournalDate        time.Time     `db:"journal_date"`
	Description        string        `db:"description"`
	CurrencyCode       string        `db:"currency_code"`
	Status             JournalStatus `db:"status"`               // Use type from common.go
	OriginalJournalID  *string       `db:"original_journal_id"`  // Link to the journal this one reverses
	ReversingJournalID *string       `db:"reversing_journal_id"` // Link to the journal that reverses this one
	AuditFields                      // Embed common audit fields
}

package repositories

import (
	"context"
	"time"

	"github.com/SscSPs/money_managemet_app/internal/core/domain"
	"github.com/shopspring/decimal"
)

// JournalReader defines read operations for journal data
type JournalReader interface {
	// FindJournalByID retrieves a specific journal by its unique identifier.
	FindJournalByID(ctx context.Context, journalID string) (*domain.Journal, error)

	// ListJournalsByWorkplace retrieves a paginated list of journals for a given workplace using token-based pagination.
	// It returns the journals, a token for the next page, and an error.
	ListJournalsByWorkplace(ctx context.Context, workplaceID string, limit int, nextToken *string, includeReversals bool) ([]domain.Journal, *string, error)
}

// JournalWriter defines write operations for journal data
type JournalWriter interface {
	// SaveJournal persists a journal and its transactions, updating account balances within a transaction.
	SaveJournal(ctx context.Context, journal domain.Journal, transactions []domain.Transaction, balanceChanges map[string]decimal.Decimal) error

	// UpdateJournalStatusAndLinks updates the status and reversal linkage (original/reversing IDs) of a journal.
	UpdateJournalStatusAndLinks(ctx context.Context, journalID string, status domain.JournalStatus, reversingJournalID *string, originalJournalID *string, updatedByUserID string, updatedAt time.Time) error

	// UpdateJournal updates non-status fields of a journal (like description, date).
	UpdateJournal(ctx context.Context, journal domain.Journal) error
}

// TransactionReader defines read operations for transaction data
type TransactionReader interface {
	// FindTransactionsByJournalID retrieves all transactions associated with a single journal ID.
	FindTransactionsByJournalID(ctx context.Context, journalID string) ([]domain.Transaction, error)

	// FindTransactionsByJournalIDs retrieves transactions for multiple journal IDs, grouped by journal ID.
	FindTransactionsByJournalIDs(ctx context.Context, journalIDs []string) (map[string][]domain.Transaction, error)

	// ListTransactionsByAccountID retrieves a paginated list of transactions for a specific account using token-based pagination.
	// It returns the transactions, a token for the next page, and an error.
	ListTransactionsByAccountID(ctx context.Context, workplaceID, accountID string, limit int, nextToken *string) ([]domain.Transaction, *string, error)
}

// JournalRepositoryFacade combines all journal-related repository interfaces
// This is a facade for clients that need access to all operations
type JournalRepositoryFacade interface {
	JournalReader
	JournalWriter
	TransactionReader
}

// JournalRepositoryWithTx extends JournalRepositoryFacade with transaction capabilities
type JournalRepositoryWithTx interface {
	JournalRepositoryFacade
	TransactionManager
}

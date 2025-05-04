package services

import (
	"context"

	"github.com/SscSPs/money_managemet_app/internal/core/domain"
	"github.com/SscSPs/money_managemet_app/internal/dto"
	"github.com/shopspring/decimal"
)

// JournalReaderSvc defines read operations for journal data
type JournalReaderSvc interface {
	// GetJournalByID retrieves a specific journal by its ID.
	GetJournalByID(ctx context.Context, workplaceID string, journalID string, requestingUserID string) (*domain.Journal, error)

	// ListJournals retrieves a paginated list of journals in a workplace.
	ListJournals(ctx context.Context, workplaceID string, userID string, params dto.ListJournalsParams) (*dto.ListJournalsResponse, error)
}

// JournalWriterSvc defines write operations for journal data
type JournalWriterSvc interface {
	// CreateJournal persists a new journal with its transactions.
	CreateJournal(ctx context.Context, workplaceID string, req dto.CreateJournalRequest, creatorUserID string) (*domain.Journal, error)

	// UpdateJournal updates journal details (excluding transactions).
	UpdateJournal(ctx context.Context, workplaceID string, journalID string, req dto.UpdateJournalRequest, requestingUserID string) (*domain.Journal, error)

	// DeactivateJournal marks a journal as inactive.
	DeactivateJournal(ctx context.Context, workplaceID string, journalID string, requestingUserID string) error

	// ReverseJournal creates a reversal journal for an existing journal.
	ReverseJournal(ctx context.Context, workplaceID string, journalID string, userID string) (*domain.Journal, error)
}

// TransactionReaderSvc defines read operations for transaction data
type TransactionReaderSvc interface {
	// ListTransactionsByAccount retrieves transactions for a specific account.
	ListTransactionsByAccount(ctx context.Context, workplaceID string, accountID string, userID string, params dto.ListTransactionsParams) (*dto.ListTransactionsResponse, error)
}

// JournalCalculatorSvc defines calculation operations related to journals
type JournalCalculatorSvc interface {
	// CalculateAccountBalance calculates the current balance of an account.
	CalculateAccountBalance(ctx context.Context, workplaceID string, accountID string) (decimal.Decimal, error)
}

// JournalSvcFacade combines all journal-related service interfaces
// This is a facade for clients that need access to all operations
type JournalSvcFacade interface {
	JournalReaderSvc
	JournalWriterSvc
	TransactionReaderSvc
	JournalCalculatorSvc
}

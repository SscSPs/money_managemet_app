package dto

import (
	"time"

	"github.com/SscSPs/money_managemet_app/internal/core/domain"
	"github.com/shopspring/decimal"
)

// --- Journal DTOs (Updated for CRUD within Workplace) ---

// CreateJournalRequest defines data for creating a journal entry (without transactions).
type CreateJournalRequest struct {
	Date         time.Time                  `json:"date" binding:"required"`
	Description  string                     `json:"description"`
	CurrencyCode string                     `json:"currencyCode" binding:"required,iso4217"`    // Enforce valid currency code
	Transactions []CreateTransactionRequest `json:"transactions" binding:"required,min=2,dive"` // Embed transactions
}

// CreateTransactionRequest defines data for a single transaction within a journal creation request.
type CreateTransactionRequest struct {
	AccountID       string                 `json:"accountID" binding:"required,uuid"`
	Amount          decimal.Decimal        `json:"amount" binding:"required,gt=0"` // Must be positive
	TransactionType domain.TransactionType `json:"transactionType" binding:"required,oneof=DEBIT CREDIT"`
	Notes           string                 `json:"notes"`
	// CurrencyCode is inherited from the Journal
}

// JournalResponse defines the data returned for a journal entry (excluding transactions).
type JournalResponse struct {
	JournalID    string    `json:"journalID"`
	WorkplaceID  string    `json:"workplaceID"`
	Date         time.Time `json:"date"`
	Description  string    `json:"description"`
	CurrencyCode string    `json:"currencyCode"`
	// Status domain.JournalStatus `json:"status"` // Status might not be needed/settable directly via CRUD
	CreatedAt     time.Time `json:"createdAt"`
	CreatedBy     string    `json:"createdBy"`
	LastUpdatedAt time.Time `json:"lastUpdatedAt"`
	LastUpdatedBy string    `json:"lastUpdatedBy"`
}

// ToJournalResponse converts domain.Journal to JournalResponse DTO.
func ToJournalResponse(j *domain.Journal) JournalResponse {
	return JournalResponse{
		JournalID:    j.JournalID,
		WorkplaceID:  j.WorkplaceID,
		Date:         j.JournalDate,
		Description:  j.Description,
		CurrencyCode: j.CurrencyCode,
		// Status: j.Status,
		CreatedAt:     j.CreatedAt,
		CreatedBy:     j.CreatedBy,
		LastUpdatedAt: j.LastUpdatedAt,
		LastUpdatedBy: j.LastUpdatedBy,
	}
}

// ListJournalsParams defines query parameters for listing journals.
type ListJournalsParams struct {
	Limit  int `form:"limit,default=20"`
	Offset int `form:"offset,default=0"`
	// TODO: Add filtering options like date range, status?
}

// ListJournalsResponse wraps a list of journal responses.
type ListJournalsResponse struct {
	Journals []JournalResponse `json:"journals"`
	// TODO: Add pagination metadata (total count, limit, offset)?
}

// UpdateJournalRequest defines data for updating a journal entry's details.
type UpdateJournalRequest struct {
	Date        *time.Time `json:"date"`        // Pointer to allow optional update
	Description *string    `json:"description"` // Pointer to allow optional update
}

// --- Transaction DTOs (Separate for potential future use) ---

// TransactionResponse defines the data returned for a transaction entry.
type TransactionResponse struct {
	TransactionID   string                 `json:"transactionID"`
	JournalID       string                 `json:"journalID"`
	AccountID       string                 `json:"accountID"`
	Amount          decimal.Decimal        `json:"amount"` // Always positive
	TransactionType domain.TransactionType `json:"transactionType"`
	CurrencyCode    string                 `json:"currencyCode"`
	Notes           string                 `json:"notes"`
	CreatedAt       time.Time              `json:"createdAt"`
	CreatedBy       string                 `json:"createdBy"`
}

// ToTransactionResponse converts domain.Transaction to TransactionResponse DTO.
func ToTransactionResponse(t *domain.Transaction) TransactionResponse {
	return TransactionResponse{
		TransactionID:   t.TransactionID,
		JournalID:       t.JournalID,
		AccountID:       t.AccountID,
		Amount:          t.Amount, // Already positive in domain
		TransactionType: t.TransactionType,
		CurrencyCode:    t.CurrencyCode,
		Notes:           t.Notes,
		CreatedAt:       t.CreatedAt,
		CreatedBy:       t.CreatedBy,
	}
}

// ToTransactionResponses converts a slice of domain.Transaction to DTOs.
func ToTransactionResponses(ts []domain.Transaction) []TransactionResponse {
	list := make([]TransactionResponse, len(ts))
	for i, t := range ts {
		list[i] = ToTransactionResponse(&t)
	}
	return list
}

// --- Old DTOs (Marked for removal or refactoring) ---

/*
// CreateJournalAndTxn was used by the old PersistJournal endpoint.
// Replaced by CreateJournalRequest which embeds CreateTransactionRequest.
type CreateJournalAndTxn struct {
	Journal      models.Journal      `json:"journal" binding:"required"`
	Transactions []models.Transaction `json:"transactions" binding:"required,min=2,dive"` // Ensure at least two entries, validate each
}
*/

/*
// GetJournalResponse combined Journal and Transactions.
// The new approach might involve separate calls or a different combined DTO if needed.
type GetJournalResponse struct {
	Journal      JournalResponse       `json:"journal"`
	Transactions []TransactionResponse `json:"transactions"`
}
*/

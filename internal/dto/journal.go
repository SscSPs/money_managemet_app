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
	Amount          decimal.Decimal        `json:"amount" binding:"required,decimal_gtz"` // Use custom validator
	TransactionType domain.TransactionType `json:"transactionType" binding:"required,oneof=DEBIT CREDIT"`
	Notes           string                 `json:"notes"`
	// CurrencyCode is inherited from the Journal
}

// JournalResponse defines the data returned for a journal entry.
type JournalResponse struct {
	JournalID    string    `json:"journalID"`
	WorkplaceID  string    `json:"workplaceID"`
	Date         time.Time `json:"date"`
	Description  string    `json:"description"`
	CurrencyCode string    `json:"currencyCode"`
	// Status domain.JournalStatus `json:"status"` // Status might not be needed/settable directly via CRUD
	CreatedAt     time.Time             `json:"createdAt"`
	CreatedBy     string                `json:"createdBy"`
	LastUpdatedAt time.Time             `json:"lastUpdatedAt"`
	LastUpdatedBy string                `json:"lastUpdatedBy"`
	Transactions  []TransactionResponse `json:"transactions,omitempty"` // Added transactions
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
		Transactions:  ToTransactionResponses(j.Transactions), // Map transactions
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
	RunningBalance  decimal.Decimal        `json:"runningBalance,omitempty"` // Added running balance
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
		RunningBalance:  t.RunningBalance, // Added running balance
	}
}

// ToTransactionResponses converts a slice of domain.Transaction to DTOs.
func ToTransactionResponses(ts []domain.Transaction) []TransactionResponse {
	list := make([]TransactionResponse, len(ts))
	// Use index directly instead of ranging over value
	for i := range ts {
		// Create a pointer to the transaction in the slice
		tPtr := &ts[i] // Get address of element directly
		list[i] = ToTransactionResponse(tPtr)
	}
	return list
}

// ListTransactionsParams defines query parameters for listing transactions.
type ListTransactionsParams struct {
	Limit  int `form:"limit,default=20"`
	Offset int `form:"offset,default=0"`
	// TODO: Add filtering options like date range?
}

// ListTransactionsResponse wraps a list of transaction responses.
type ListTransactionsResponse struct {
	Transactions []TransactionResponse `json:"transactions"`
	// TODO: Add pagination metadata (total count, limit, offset)?
}

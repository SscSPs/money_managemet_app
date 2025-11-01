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
	AccountID        string                 `json:"accountID" binding:"required,uuid"`
	Amount           decimal.Decimal        `json:"amount" binding:"required,decimal_gtz"` // Amount in journal's base currency
	OriginalAmount   *decimal.Decimal       `json:"originalAmount,omitempty"`              // Original amount in transaction's currency
	OriginalCurrency *string                `json:"originalCurrency,omitempty"`            // Original currency code (ISO 4217)
	ExchangeRateID   *string                `json:"exchangeRateId,omitempty"`              // Reference to exchange rate used
	TransactionType  domain.TransactionType `json:"transactionType" binding:"required,oneof=DEBIT CREDIT"`
	TransactionDate  *time.Time             `json:"transactionDate,omitempty"` // Optional, defaults to journal date if not provided
	Notes            string                 `json:"notes"`
	// CurrencyCode is inherited from the Journal
}

// JournalResponse defines the data returned for a journal entry.
type JournalResponse struct {
	JournalID          string                `json:"journalID"`
	WorkplaceID        string                `json:"workplaceID"`
	Date               time.Time             `json:"date"`
	Description        string                `json:"description"`
	CurrencyCode       string                `json:"currencyCode"`
	Status             domain.JournalStatus  `json:"status"` // Status (e.g., POSTED, REVERSED)
	OriginalJournalID  *string               `json:"originalJournalID,omitempty"`
	ReversingJournalID *string               `json:"reversingJournalID,omitempty"`
	Amount             decimal.Decimal       `json:"amount,omitempty"` // Total movement amount in the journal
	CreatedAt          time.Time             `json:"createdAt"`
	CreatedBy          string                `json:"createdBy"`
	LastUpdatedAt      time.Time             `json:"lastUpdatedAt"`
	LastUpdatedBy      string                `json:"lastUpdatedBy"`
	Transactions       []TransactionResponse `json:"transactions,omitempty"` // Added transactions
}

// ToJournalResponse converts domain.Journal to JournalResponse DTO.
func ToJournalResponse(j *domain.Journal) JournalResponse {
	return JournalResponse{
		JournalID:          j.JournalID,
		WorkplaceID:        j.WorkplaceID,
		Date:               j.JournalDate,
		Description:        j.Description,
		CurrencyCode:       j.CurrencyCode,
		Status:             j.Status,             // Map status
		OriginalJournalID:  j.OriginalJournalID,  // Map link
		ReversingJournalID: j.ReversingJournalID, // Map link
		Amount:             j.Amount,             // Map amount
		CreatedAt:          j.CreatedAt,
		CreatedBy:          j.CreatedBy,
		LastUpdatedAt:      j.LastUpdatedAt,
		LastUpdatedBy:      j.LastUpdatedBy,
		Transactions:       ToTransactionResponses(j.Transactions), // Map transactions
	}
}

// ListJournalsParams defines query parameters for listing journals.
// Uses token-based pagination.
type ListJournalsParams struct {
	Limit               int     `form:"limit" binding:"omitempty,gte=1,lte=100"` // Limit results, default 20, max 100
	NextToken           *string `form:"nextToken"`                               // Token for the next page
	IncludeReversals    bool    `form:"includeReversals"`                        // Whether to include reversed and reversing journals
	IncludeTransactions bool    `form:"includeTxn"`                              // Whether to include transactions in the response
}

// ListJournalsResponse wraps a list of journal responses.
// Uses token-based pagination.
type ListJournalsResponse struct {
	Journals  []JournalResponse `json:"journals"`
	NextToken *string           `json:"nextToken,omitempty"` // Token to fetch the next page
}

// UpdateJournalRequest defines data for updating a journal entry's details.
type UpdateJournalRequest struct {
	Date        *time.Time `json:"date"`        // Pointer to allow optional update
	Description *string    `json:"description"` // Pointer to allow optional update
}

// --- Transaction DTOs (Separate for potential future use) ---

// TransactionResponse defines the data returned for a transaction entry.
type TransactionResponse struct {
	TransactionID      string                 `json:"transactionID"`
	JournalID          string                 `json:"journalID"`
	AccountID          string                 `json:"accountID"`
	Amount             decimal.Decimal        `json:"amount"`
	OriginalAmount     *decimal.Decimal       `json:"originalAmount,omitempty"`
	OriginalCurrency   *string                `json:"originalCurrency,omitempty"`
	ExchangeRate       *decimal.Decimal       `json:"exchangeRate,omitempty"`
	ExchangeRateID     *string                `json:"exchangeRateId,omitempty"`
	TransactionType    domain.TransactionType `json:"transactionType"`
	Notes              string                 `json:"notes"`
	TransactionDate    time.Time              `json:"transactionDate"`
	RunningBalance     decimal.Decimal        `json:"runningBalance,omitempty"`
	JournalDate        time.Time              `json:"journalDate"`
	JournalDescription string                 `json:"journalDescription"`
}

// ToTransactionResponse converts domain.Transaction to TransactionResponse DTO.
func ToTransactionResponse(t *domain.Transaction) TransactionResponse {
	txnResponse := TransactionResponse{
		TransactionID:      t.TransactionID,
		JournalID:          t.JournalID,
		AccountID:          t.AccountID,
		Amount:             t.Amount,
		TransactionType:    t.TransactionType,
		Notes:              t.Notes,
		TransactionDate:    t.TransactionDate,
		RunningBalance:     t.RunningBalance,
		JournalDate:        t.JournalDate,
		JournalDescription: t.JournalDescription,
	}

	// Add multi-currency fields if available
	if t.OriginalAmount != nil && t.OriginalCurrency != nil {
		txnResponse.OriginalAmount = t.OriginalAmount
		txnResponse.OriginalCurrency = t.OriginalCurrency
		txnResponse.ExchangeRateID = t.ExchangeRateID

		// Calculate exchange rate if both original and converted amounts are available
		if !t.Amount.IsZero() && !t.OriginalAmount.IsZero() {
			exchangeRate := t.Amount.Div(*t.OriginalAmount)
			txnResponse.ExchangeRate = &exchangeRate
		}
	}

	return txnResponse
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
	Limit     int     `form:"limit" binding:"omitempty,gte=1,lte=100"` // Limit results, default 20, max 100
	NextToken *string `form:"nextToken"`                               // Token for the next page
}

// ListTransactionsResponse wraps a list of transaction responses.
type ListTransactionsResponse struct {
	Transactions []TransactionResponse `json:"transactions"`
	NextToken    *string               `json:"nextToken,omitempty"` // Token to fetch the next page
}

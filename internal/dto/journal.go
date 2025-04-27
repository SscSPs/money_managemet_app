package dto

import (
	"time"

	"github.com/SscSPs/money_managemet_app/internal/core/domain"
	"github.com/shopspring/decimal"
)

// TransactionResponse defines the data returned for a transaction.
type TransactionResponse struct {
	TransactionID string          `json:"transactionID"`
	AccountID     string          `json:"accountID"`
	Amount        decimal.Decimal `json:"amount"`
	Type          string          `json:"type"` // DEBIT or CREDIT
	// Maybe add link back to JournalID if needed?
}

// JournalResponse defines the data returned for a journal entry.
type JournalResponse struct {
	JournalID   string    `json:"journalID"`
	Date        time.Time `json:"date"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"createdAt"`
	CreatedBy   string    `json:"createdBy"`
}

// GetJournalResponse defines the combined response for getting a journal and its transactions.
type GetJournalResponse struct {
	Journal      JournalResponse       `json:"journal"`
	Transactions []TransactionResponse `json:"transactions"`
}

// ToTransactionResponse converts a domain.Transaction to TransactionResponse DTO.
func ToTransactionResponse(txn *domain.Transaction) TransactionResponse {
	return TransactionResponse{
		TransactionID: txn.TransactionID,
		AccountID:     txn.AccountID,
		Amount:        txn.Amount,
		Type:          string(txn.TransactionType),
	}
}

// ToTransactionResponses converts a slice of domain.Transaction to []TransactionResponse.
func ToTransactionResponses(txns []domain.Transaction) []TransactionResponse {
	responses := make([]TransactionResponse, len(txns))
	for i, txn := range txns {
		responses[i] = ToTransactionResponse(&txn)
	}
	return responses
}

// ToJournalResponse converts a domain.Journal to JournalResponse DTO.
func ToJournalResponse(j *domain.Journal) JournalResponse {
	return JournalResponse{
		JournalID:   j.JournalID,
		Date:        j.JournalDate,
		Description: j.Description,
		CreatedAt:   j.CreatedAt,
		CreatedBy:   j.CreatedBy,
	}
}

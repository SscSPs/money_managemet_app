package dto

import (
	"time"

	"github.com/SscSPs/money_managemet_app/internal/models"
	"github.com/shopspring/decimal"
)

// CreateAccountRequest defines the data needed to create a new account.
type CreateAccountRequest struct {
	Name            string             `json:"name" binding:"required"`
	AccountType     models.AccountType `json:"accountType" binding:"required,oneof=ASSET LIABILITY EQUITY INCOME EXPENSE"`
	CurrencyCode    string             `json:"currencyCode" binding:"required"`
	ParentAccountID *string            `json:"parentAccountID"` // Optional, use pointer for nullability
	Description     string             `json:"description"`     // Optional
	UserID          string             `json:"userID"`          // needed for audit fields
}

// AccountResponse defines the data returned for an account.
// It mirrors models.Account but ensures AuditFields are included.
type AccountResponse struct {
	AccountID       string             `json:"accountID"`
	Name            string             `json:"name"`
	AccountType     models.AccountType `json:"accountType"`
	CurrencyCode    string             `json:"currencyCode"`
	ParentAccountID string             `json:"parentAccountID"` // Note: Empty string if null in DB
	Description     string             `json:"description"`
	IsActive        bool               `json:"isActive"`
	CreatedAt       time.Time          `json:"createdAt"`
	CreatedBy       string             `json:"createdBy"`
	LastUpdatedAt   time.Time          `json:"lastUpdatedAt"`
	LastUpdatedBy   string             `json:"lastUpdatedBy"`
}

// ToAccountResponse converts a models.Account to AccountResponse DTO
func ToAccountResponse(acc *models.Account) AccountResponse {
	return AccountResponse{
		AccountID:       acc.AccountID,
		Name:            acc.Name,
		AccountType:     acc.AccountType,
		CurrencyCode:    acc.CurrencyCode,
		ParentAccountID: acc.ParentAccountID,
		Description:     acc.Description,
		IsActive:        acc.IsActive,
		CreatedAt:       acc.CreatedAt,
		CreatedBy:       acc.CreatedBy,
		LastUpdatedAt:   acc.LastUpdatedAt,
		LastUpdatedBy:   acc.LastUpdatedBy,
	}
}

// ToListAccountResponse converts a slice of models.Account to a slice of AccountResponse DTOs
func ToListAccountResponse(accounts []models.Account) []AccountResponse {
	res := make([]AccountResponse, len(accounts))
	for i, acc := range accounts {
		res[i] = ToAccountResponse(&acc) // Reuse the single converter
	}
	return res
}

// AccountBalanceResponse defines the data returned for an account balance query.
type AccountBalanceResponse struct {
	AccountID string          `json:"accountID"`
	Balance   decimal.Decimal `json:"balance"`
	// Could add currency code here if needed
}

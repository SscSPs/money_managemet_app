package dto

import (
	"time"

	"github.com/SscSPs/money_managemet_app/internal/core/domain"
	"github.com/shopspring/decimal"
)

// CreateAccountRequest defines the data needed to create a new account.
type CreateAccountRequest struct {
	Name            string             `json:"name" binding:"required"`
	AccountType     domain.AccountType `json:"accountType" binding:"required,oneof=ASSET LIABILITY EQUITY INCOME EXPENSE"`
	CurrencyCode    string             `json:"currencyCode" binding:"required"`
	ParentAccountID *string            `json:"parentAccountID"` // Optional, use pointer for nullability
	Description     string             `json:"description"`     // Optional
	UserID          string             `json:"userID"`          // needed for audit fields
}

// AccountResponse defines the data returned for an account.
// Mirrors domain.Account.
type AccountResponse struct {
	AccountID       string             `json:"accountID"`
	Name            string             `json:"name"`
	AccountType     domain.AccountType `json:"accountType"`
	CurrencyCode    string             `json:"currencyCode"`
	ParentAccountID string             `json:"parentAccountID"` // Note: Empty string if null in DB
	Description     string             `json:"description"`
	IsActive        bool               `json:"isActive"`
	CreatedAt       time.Time          `json:"createdAt"`
	CreatedBy       string             `json:"createdBy"`
	LastUpdatedAt   time.Time          `json:"lastUpdatedAt"`
	LastUpdatedBy   string             `json:"lastUpdatedBy"`
}

// UpdateAccountRequest defines the data allowed for updating an account.
// Use pointers to distinguish between zero-value updates and fields not provided.
type UpdateAccountRequest struct {
	Name        *string `json:"name"`        // Optional: New name
	Description *string `json:"description"` // Optional: New description
	IsActive    *bool   `json:"isActive"`    // Optional: New active status
}

// ToAccountResponse converts a domain.Account to AccountResponse DTO
func ToAccountResponse(acc *domain.Account) AccountResponse {
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

// ToListAccountResponse converts a slice of domain.Account to a slice of AccountResponse DTOs
func ToListAccountResponse(accounts []domain.Account) []AccountResponse {
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

// ListAccountsParams defines query parameters for listing accounts.
type ListAccountsParams struct {
	Limit  int `form:"limit,default=20"`
	Offset int `form:"offset,default=0"`
	// Add filters later (e.g., type, currency, name)?
}

// ListAccountsResponse wraps the list of accounts.
type ListAccountsResponse struct {
	Accounts []AccountResponse `json:"accounts"`
	// TODO: Add pagination metadata (total count, limit, offset) later
}

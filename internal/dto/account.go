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
	CurrencyCode    string             `json:"currencyCode" binding:"required,iso4217"`
	Description     string             `json:"description"`
	ParentAccountID *string            `json:"parentAccountID,omitempty" binding:"omitempty,uuid"` // Optional, must be UUID if provided
	// UserID is extracted from the context, not part of the request body
	// UserID string `json:"userID" binding:"required"` // Removed from here
}

// AccountResponse defines the data returned for an account.
// Mirrors domain.Account.
type AccountResponse struct {
	AccountID       string             `json:"accountID"`
	WorkplaceID     string             `json:"workplaceID"`
	Name            string             `json:"name"`
	AccountType     domain.AccountType `json:"accountType"`
	CurrencyCode    string             `json:"currencyCode"`
	ParentAccountID string             `json:"parentAccountID,omitempty"`
	Description     string             `json:"description"`
	IsActive        bool               `json:"isActive"`
	CreatedAt       time.Time          `json:"createdAt"`
	CreatedBy       string             `json:"createdBy"` // UserID
	LastUpdatedAt   time.Time          `json:"lastUpdatedAt"`
	LastUpdatedBy   string             `json:"lastUpdatedBy"` // UserID
	// Balance is not typically included directly; might be a separate endpoint or calculation
}

// UpdateAccountRequest defines the data allowed for updating an account.
// Use pointers to distinguish between zero-value updates and fields not provided.
type UpdateAccountRequest struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	IsActive    *bool   `json:"isActive,omitempty"`
	// Note: AccountType, CurrencyCode, ParentAccountID are usually not updatable.
}

// ToAccountResponse converts a domain.Account to AccountResponse DTO
func ToAccountResponse(acc *domain.Account) AccountResponse {
	return AccountResponse{
		AccountID:       acc.AccountID,
		WorkplaceID:     acc.WorkplaceID,
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
	// TODO: Add filtering options like name, type, isActive?
}

// ListAccountsResponse wraps the list of accounts.
type ListAccountsResponse struct {
	Accounts []AccountResponse `json:"accounts"`
	// TODO: Add pagination metadata (total count, limit, offset) later
}

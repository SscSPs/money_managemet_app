package dto

import (
	"time"

	"github.com/SscSPs/money_managemet_app/internal/models"
)

// CreateCurrencyRequest defines the data needed to create a new currency.
type CreateCurrencyRequest struct {
	CurrencyCode string `json:"currencyCode" binding:"required,uppercase,len=3"`
	Symbol       string `json:"symbol" binding:"required"`
	Name         string `json:"name" binding:"required"`
	UserID       string `json:"userID" binding:"required"`
}

// CurrencyResponse defines the data returned for a currency.
type CurrencyResponse struct {
	CurrencyCode  string    `json:"currencyCode"`
	Symbol        string    `json:"symbol"`
	Name          string    `json:"name"`
	CreatedAt     time.Time `json:"createdAt"`
	CreatedBy     string    `json:"createdBy"`
	LastUpdatedAt time.Time `json:"lastUpdatedAt"`
	LastUpdatedBy string    `json:"lastUpdatedBy"`
}

// ToCurrencyResponse converts a models.Currency to CurrencyResponse DTO
func ToCurrencyResponse(curr *models.Currency) CurrencyResponse {
	return CurrencyResponse{
		CurrencyCode:  curr.CurrencyCode,
		Symbol:        curr.Symbol,
		Name:          curr.Name,
		CreatedAt:     curr.CreatedAt,
		CreatedBy:     curr.CreatedBy,
		LastUpdatedAt: curr.LastUpdatedAt,
		LastUpdatedBy: curr.LastUpdatedBy,
	}
}

// ToListCurrencyResponse converts a slice of models.Currency to a slice of CurrencyResponse DTOs
func ToListCurrencyResponse(currencies []models.Currency) []CurrencyResponse {
	res := make([]CurrencyResponse, len(currencies))
	for i, curr := range currencies {
		res[i] = ToCurrencyResponse(&curr) // Reuse the single converter
	}
	return res
}

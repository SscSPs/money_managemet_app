package dto

import (
	"github.com/SscSPs/money_managemet_app/internal/core/domain"
)

// CreateCurrencyRequest defines the data needed to create a new currency.
type CreateCurrencyRequest struct {
	CurrencyCode string `json:"currencyCode" binding:"required,uppercase,len=3"`
	Symbol       string `json:"symbol" binding:"required"`
	Name         string `json:"name" binding:"required"`
}

// CurrencyResponse defines the data returned for a currency.
type CurrencyResponse struct {
	CurrencyCode string `json:"currencyCode"`
	Symbol       string `json:"symbol"`
	Name         string `json:"name"`
}

// ToCurrencyResponse converts a domain.Currency to CurrencyResponse DTO
func ToCurrencyResponse(curr *domain.Currency) CurrencyResponse {
	return CurrencyResponse{
		CurrencyCode: curr.CurrencyCode,
		Symbol:       curr.Symbol,
		Name:         curr.Name,
	}
}

// ToListCurrencyResponse converts a slice of domain.Currency to a slice of CurrencyResponse DTOs
func ToListCurrencyResponse(currencies []domain.Currency) []CurrencyResponse {
	res := make([]CurrencyResponse, len(currencies))
	for i, curr := range currencies {
		res[i] = ToCurrencyResponse(&curr) // Reuse the single converter
	}
	return res
}

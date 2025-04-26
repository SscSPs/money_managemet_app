package dto

import (
	"time"

	"github.com/SscSPs/money_managemet_app/internal/core/domain"
	"github.com/shopspring/decimal"
)

// CreateExchangeRateRequest defines the structure for creating a new exchange rate.
type CreateExchangeRateRequest struct {
	FromCurrencyCode string          `json:"fromCurrencyCode" binding:"required,len=3,uppercase"`
	ToCurrencyCode   string          `json:"toCurrencyCode" binding:"required,len=3,uppercase"`
	Rate             decimal.Decimal `json:"rate" binding:"required"` // Consider adding validation for > 0
	DateEffective    time.Time       `json:"dateEffective" binding:"required"`
}

// ExchangeRateResponse defines the structure for API responses containing exchange rate details.
type ExchangeRateResponse struct {
	ExchangeRateID   string          `json:"exchangeRateID"`
	FromCurrencyCode string          `json:"fromCurrencyCode"`
	ToCurrencyCode   string          `json:"toCurrencyCode"`
	Rate             decimal.Decimal `json:"rate"`
	DateEffective    time.Time       `json:"dateEffective"`
	CreatedAt        time.Time       `json:"createdAt"`
	CreatedBy        string          `json:"createdBy"`
	LastUpdatedAt    time.Time       `json:"lastUpdatedAt"`
	LastUpdatedBy    string          `json:"lastUpdatedBy"`
}

// ToExchangeRateResponse converts a domain.ExchangeRate to ExchangeRateResponse DTO
func ToExchangeRateResponse(rate *domain.ExchangeRate) ExchangeRateResponse {
	return ExchangeRateResponse{
		ExchangeRateID:   rate.ExchangeRateID,
		FromCurrencyCode: rate.FromCurrencyCode,
		ToCurrencyCode:   rate.ToCurrencyCode,
		Rate:             rate.Rate,
		DateEffective:    rate.DateEffective,
		CreatedAt:        rate.CreatedAt,
		CreatedBy:        rate.CreatedBy,
		LastUpdatedAt:    rate.LastUpdatedAt,
		LastUpdatedBy:    rate.LastUpdatedBy,
	}
}

// ToListExchangeRateResponse converts a slice of models.ExchangeRate to a slice of ExchangeRateResponse DTOs.
func ToListExchangeRateResponse(rates []*domain.ExchangeRate) []ExchangeRateResponse {
	responses := make([]ExchangeRateResponse, len(rates))
	for i, rate := range rates {
		responses[i] = ToExchangeRateResponse(rate)
	}
	return responses
}

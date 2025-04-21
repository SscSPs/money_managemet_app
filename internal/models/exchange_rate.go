package models

import (
	"time"

	"github.com/shopspring/decimal"
)

// ExchangeRate stores the conversion rate between two currencies for a specific date.
// Note: Rate should use a precise decimal type like github.com/shopspring/decimal
type ExchangeRate struct {
	ExchangeRateID   string          `json:"exchangeRateID"`   // Primary Key (e.g., UUID)
	FromCurrencyCode string          `json:"fromCurrencyCode"` // FK -> Currency.currencyCode
	ToCurrencyCode   string          `json:"toCurrencyCode"`   // FK -> Currency.currencyCode
	Rate             decimal.Decimal `json:"rate"`             // Precise decimal type
	DateEffective    time.Time       `json:"dateEffective"`
	AuditFields
}

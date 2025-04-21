package models

// Currency represents a supported currency.
type Currency struct {
	CurrencyCode string `json:"currencyCode"` // Primary Key (e.g., "USD")
	Symbol       string `json:"symbol"`       // e.g., "$"
	Name         string `json:"name"`         // e.g., "US Dollar"
	AuditFields
}

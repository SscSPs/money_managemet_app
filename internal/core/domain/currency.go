package domain

// Currency represents a supported currency in the domain.
type Currency struct {
	CurrencyCode string `json:"currencyCode"` // Primary Key (e.g., "USD")
	Symbol       string `json:"symbol"`       // e.g., "$"
	Name         string `json:"name"`         // e.g., "US Dollar"
	AuditFields
}

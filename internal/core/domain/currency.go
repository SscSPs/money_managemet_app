package domain

// Currency represents a supported currency in the domain.
type Currency struct {
	CurrencyCode string `json:"currencyCode"` // Primary Key (e.g., "USD")
	Symbol       string `json:"symbol"`       // e.g., "$"
	Name         string `json:"name"`         // e.g., "US Dollar"
	Precision    int    `json:"precision"`    // Number of decimal places (e.g., 2 for USD, 0 for JPY)
	AuditFields
}

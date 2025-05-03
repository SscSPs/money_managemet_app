package utils

import (
	"github.com/SscSPs/money_managemet_app/internal/core/domain"
	"github.com/shopspring/decimal"
)

// FormatWithCurrencyPrecision formats an amount with the correct precision for a given currency
// Example: amount 12.3456 with USD (precision 2) returns "12.35"
// Example: amount 12.3456 with JPY (precision 0) returns "12"
// Example: amount 12.3456789012345678 with ETH (precision 18) returns "12.3456789012345678"
func FormatWithCurrencyPrecision(amount decimal.Decimal, currency domain.Currency) string {
	return amount.Round(int32(currency.Precision)).String()
}

// FormatWithPrecision formats an amount with the given precision
// This is a convenience function when you only have the precision value
func FormatWithPrecision(amount decimal.Decimal, precision int) string {
	return amount.Round(int32(precision)).String()
}

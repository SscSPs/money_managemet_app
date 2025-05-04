package repositories

import (
	"context"

	"github.com/SscSPs/money_managemet_app/internal/core/domain"
)

// ExchangeRateReader defines read operations for exchange rate data
type ExchangeRateReader interface {
	// FindExchangeRate retrieves an exchange rate between two currencies.
	FindExchangeRate(ctx context.Context, fromCurrencyCode, toCurrencyCode string) (*domain.ExchangeRate, error)
}

// ExchangeRateWriter defines write operations for exchange rate data
type ExchangeRateWriter interface {
	// SaveExchangeRate persists a new exchange rate.
	SaveExchangeRate(ctx context.Context, rate domain.ExchangeRate) error
}

// ExchangeRateRepositoryFacade combines all exchange rate-related repository interfaces
// This is a facade for clients that need access to all operations
type ExchangeRateRepositoryFacade interface {
	ExchangeRateReader
	ExchangeRateWriter
}

// ExchangeRateRepositoryWithTx extends ExchangeRateRepositoryFacade with transaction capabilities
type ExchangeRateRepositoryWithTx interface {
	ExchangeRateRepositoryFacade
	TransactionManager
}

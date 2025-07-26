package repositories

import (
	"context"
	"time"

	"github.com/SscSPs/money_managemet_app/internal/core/domain"
)

// ExchangeRateReader defines read operations for exchange rate data
type ExchangeRateReader interface {
	// FindExchangeRate retrieves an exchange rate between two currencies.
	FindExchangeRate(ctx context.Context, fromCurrencyCode, toCurrencyCode string) (*domain.ExchangeRate, error)
	// FindExchangeRateByID retrieves an exchange rate by its ID.
	FindExchangeRateByID(ctx context.Context, rateID string) (*domain.ExchangeRate, error)
	// FindExchangeRateByIDs retrieves exchange rates by their IDs.
	FindExchangeRateByIDs(ctx context.Context, rateIDs []string) ([]domain.ExchangeRate, error)
	// ListExchangeRates retrieves all exchange rates with optional filtering.
	ListExchangeRates(ctx context.Context, fromCurrency, toCurrency *string, effectiveDate *time.Time, page, pageSize int) ([]domain.ExchangeRate, int, error)
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

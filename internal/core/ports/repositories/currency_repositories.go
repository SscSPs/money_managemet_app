package repositories

import (
	"context"

	"github.com/SscSPs/money_managemet_app/internal/core/domain"
)

// CurrencyReader defines read operations for currency data
type CurrencyReader interface {
	// FindCurrencyByCode retrieves a specific currency by its code.
	FindCurrencyByCode(ctx context.Context, currencyCode string) (*domain.Currency, error)

	// ListCurrencies retrieves all available currencies.
	ListCurrencies(ctx context.Context) ([]domain.Currency, error)
}

// CurrencyWriter defines write operations for currency data
type CurrencyWriter interface {
	// SaveCurrency persists a new currency.
	SaveCurrency(ctx context.Context, currency domain.Currency) error
}

// CurrencyRepositoryFacade combines all currency-related repository interfaces
// This is a facade for clients that need access to all operations
type CurrencyRepositoryFacade interface {
	CurrencyReader
	CurrencyWriter
}

// CurrencyRepositoryWithTx extends CurrencyRepositoryFacade with transaction capabilities
type CurrencyRepositoryWithTx interface {
	CurrencyRepositoryFacade
	TransactionManager
}

package services

import (
	"context"

	"github.com/SscSPs/money_managemet_app/internal/core/domain"
	"github.com/SscSPs/money_managemet_app/internal/dto"
)

// CurrencyReaderSvc defines read operations for currency data
type CurrencyReaderSvc interface {
	// GetCurrencyByCode retrieves a specific currency by its code.
	GetCurrencyByCode(ctx context.Context, currencyCode string) (*domain.Currency, error)

	// ListCurrencies retrieves all available currencies.
	ListCurrencies(ctx context.Context) ([]domain.Currency, error)
}

// CurrencyWriterSvc defines write operations for currency data
type CurrencyWriterSvc interface {
	// CreateCurrency persists a new currency.
	CreateCurrency(ctx context.Context, req dto.CreateCurrencyRequest, creatorUserID string) (*domain.Currency, error)
}

// CurrencySvcFacade combines all currency-related service interfaces
type CurrencySvcFacade interface {
	CurrencyReaderSvc
	CurrencyWriterSvc
}

// ExchangeRateReaderSvc defines read operations for exchange rate data
type ExchangeRateReaderSvc interface {
	// GetExchangeRate retrieves an exchange rate between two currencies.
	GetExchangeRate(ctx context.Context, fromCode, toCode string) (*domain.ExchangeRate, error)
}

// ExchangeRateWriterSvc defines write operations for exchange rate data
type ExchangeRateWriterSvc interface {
	// CreateExchangeRate persists a new exchange rate.
	CreateExchangeRate(ctx context.Context, req dto.CreateExchangeRateRequest, creatorUserID string) (*domain.ExchangeRate, error)
}

// ExchangeRateSvcFacade combines all exchange rate-related service interfaces
type ExchangeRateSvcFacade interface {
	ExchangeRateReaderSvc
	ExchangeRateWriterSvc
}

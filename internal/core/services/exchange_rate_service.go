package services

import (
	"context"
	"errors" // Import errors for Is checking
	"fmt"
	"log/slog" // Import slog
	"strings"
	"time"

	"github.com/SscSPs/money_managemet_app/internal/apperrors"
	"github.com/SscSPs/money_managemet_app/internal/core/domain" // Use domain
	portsrepo "github.com/SscSPs/money_managemet_app/internal/core/ports/repositories"
	portssvc "github.com/SscSPs/money_managemet_app/internal/core/ports/services" // Added portssvc import
	"github.com/SscSPs/money_managemet_app/internal/dto"
	"github.com/SscSPs/money_managemet_app/internal/middleware" // Import middleware
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// exchangeRateService handles exchange rate operations.
type exchangeRateService struct {
	exchangeRateRepo portsrepo.ExchangeRateRepositoryFacade
	currencyService  portssvc.CurrencySvcFacade
}

// NewExchangeRateService creates a new exchange rate service.
func NewExchangeRateService(exchangeRateRepo portsrepo.ExchangeRateRepositoryFacade, currencyService portssvc.CurrencySvcFacade) portssvc.ExchangeRateSvcFacade {
	return &exchangeRateService{
		exchangeRateRepo: exchangeRateRepo,
		currencyService:  currencyService,
	}
}

// CreateExchangeRate handles the creation of a new exchange rate.
func (s *exchangeRateService) CreateExchangeRate(ctx context.Context, req dto.CreateExchangeRateRequest, creatorUserID string) (*domain.ExchangeRate, error) {
	logger := middleware.GetLoggerFromCtx(ctx) // Get logger from context

	// Input validation (basic format) is handled by DTO binding tags.

	// Additional Service-Level Validations
	if req.Rate.LessThanOrEqual(decimal.Zero) {
		logger.Warn("Validation Error: Exchange rate must be positive", slog.Any("rate", req.Rate))
		return nil, fmt.Errorf("%w: exchange rate must be positive", apperrors.ErrValidation)
	}
	if req.FromCurrencyCode == req.ToCurrencyCode {
		logger.Warn("Validation Error: From and to currency codes cannot be the same", slog.String("code", req.FromCurrencyCode))
		return nil, fmt.Errorf("%w: from and to currency codes cannot be the same", apperrors.ErrValidation)
	}

	// Check if currencies exist
	_, errFrom := s.currencyService.GetCurrencyByCode(ctx, req.FromCurrencyCode)
	if errFrom != nil {
		if errors.Is(errFrom, apperrors.ErrNotFound) {
			logger.Warn("Validation Error: 'from' currency code not found", slog.String("currency_code", req.FromCurrencyCode))
			return nil, fmt.Errorf("%w: 'from' currency code '%s' not found", apperrors.ErrValidation, req.FromCurrencyCode)
		}
		logger.Error("Failed to validate 'from' currency", slog.String("currency_code", req.FromCurrencyCode), slog.String("error", errFrom.Error()))
		return nil, fmt.Errorf("failed to validate 'from' currency '%s': %w", req.FromCurrencyCode, errFrom)
	}

	_, errTo := s.currencyService.GetCurrencyByCode(ctx, req.ToCurrencyCode)
	if errTo != nil {
		if errors.Is(errTo, apperrors.ErrNotFound) {
			logger.Warn("Validation Error: 'to' currency code not found", slog.String("currency_code", req.ToCurrencyCode))
			return nil, fmt.Errorf("%w: 'to' currency code '%s' not found", apperrors.ErrValidation, req.ToCurrencyCode)
		}
		logger.Error("Failed to validate 'to' currency", slog.String("currency_code", req.ToCurrencyCode), slog.String("error", errTo.Error()))
		return nil, fmt.Errorf("failed to validate 'to' currency '%s': %w", req.ToCurrencyCode, errTo)
	}

	now := time.Now()
	newRateID := uuid.NewString()

	rate := domain.ExchangeRate{
		ExchangeRateID:   newRateID,
		FromCurrencyCode: req.FromCurrencyCode,
		ToCurrencyCode:   req.ToCurrencyCode,
		Rate:             req.Rate,
		DateEffective:    req.DateEffective,
		AuditFields: domain.AuditFields{
			CreatedAt:     now,
			CreatedBy:     creatorUserID,
			LastUpdatedAt: now,
			LastUpdatedBy: creatorUserID,
		},
	}

	err := s.exchangeRateRepo.SaveExchangeRate(ctx, rate)
	if err != nil {
		// Check for duplicate error from repository
		if errors.Is(err, apperrors.ErrDuplicate) {
			logger.Warn("Attempted to create duplicate exchange rate",
				slog.String("from", rate.FromCurrencyCode),
				slog.String("to", rate.ToCurrencyCode),
				slog.Time("date", rate.DateEffective),
			)
			// Map to a validation error for the client
			return nil, fmt.Errorf("%w: exchange rate for this pair and date already exists", apperrors.ErrValidation)
		}
		logger.Error("Failed to save exchange rate in repository", slog.String("error", err.Error()), slog.String("rate_id", rate.ExchangeRateID))
		return nil, fmt.Errorf("failed to create exchange rate in service: %w", err)
	}

	logger.Info("Exchange rate created successfully in service", slog.String("rate_id", rate.ExchangeRateID))
	return &rate, nil
}

// GetExchangeRate retrieves a specific exchange rate for a given currency pair and date.
func (s *exchangeRateService) GetExchangeRate(ctx context.Context, fromCode, toCode string) (*domain.ExchangeRate, error) {
	logger := middleware.GetLoggerFromCtx(ctx) // Get logger from context

	fromCode = strings.ToUpper(fromCode)
	toCode = strings.ToUpper(toCode)
	if len(fromCode) != 3 || len(toCode) != 3 {
		logger.Warn("Validation Error: Invalid currency code length", slog.String("from_code", fromCode), slog.String("to_code", toCode))
		return nil, fmt.Errorf("%w: currency codes must be 3 letters", apperrors.ErrValidation)
	}

	rate, err := s.exchangeRateRepo.FindExchangeRate(ctx, fromCode, toCode)
	if err != nil {
		logger.Error("Failed to find exchange rate in repository", slog.String("error", err.Error()), slog.String("from_code", fromCode), slog.String("to_code", toCode))
		return nil, fmt.Errorf("failed to get exchange rate in service: %w", err)
	}

	logger.Debug("Exchange rate retrieved successfully from service", slog.String("rate_id", rate.ExchangeRateID))
	return rate, nil
}

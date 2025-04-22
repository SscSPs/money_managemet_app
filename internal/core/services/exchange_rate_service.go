package services

import (
	"context"
	"errors" // Import errors for Is checking
	"fmt"
	"strings"
	"time"

	"github.com/SscSPs/money_managemet_app/internal/apperrors"
	"github.com/SscSPs/money_managemet_app/internal/core/ports"
	"github.com/SscSPs/money_managemet_app/internal/dto"
	"github.com/SscSPs/money_managemet_app/internal/models"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// ExchangeRateService provides business logic for exchange rates.
type ExchangeRateService struct {
	rateRepo        ports.ExchangeRateRepository
	currencyService *CurrencyService // Added CurrencyService dependency
}

// NewExchangeRateService creates a new ExchangeRateService.
func NewExchangeRateService(rateRepo ports.ExchangeRateRepository, currencyService *CurrencyService) *ExchangeRateService {
	return &ExchangeRateService{
		rateRepo:        rateRepo,
		currencyService: currencyService, // Store CurrencyService
	}
}

// CreateExchangeRate handles the creation of a new exchange rate.
func (s *ExchangeRateService) CreateExchangeRate(ctx context.Context, req dto.CreateExchangeRateRequest, creatorUserID string) (*models.ExchangeRate, error) {
	// Input validation (basic format) is handled by DTO binding tags.

	// Additional Service-Level Validations
	if req.Rate.LessThanOrEqual(decimal.Zero) {
		return nil, fmt.Errorf("%w: exchange rate must be positive", apperrors.ErrValidation)
	}
	if req.FromCurrencyCode == req.ToCurrencyCode {
		return nil, fmt.Errorf("%w: from and to currency codes cannot be the same", apperrors.ErrValidation)
	}

	// Check if currencies exist
	_, errFrom := s.currencyService.GetCurrencyByCode(ctx, req.FromCurrencyCode)
	if errFrom != nil {
		if errors.Is(errFrom, apperrors.ErrNotFound) {
			return nil, fmt.Errorf("%w: 'from' currency code '%s' not found", apperrors.ErrValidation, req.FromCurrencyCode)
		}
		// Log other errors from currency service? For now, wrap and return
		return nil, fmt.Errorf("failed to validate 'from' currency '%s': %w", req.FromCurrencyCode, errFrom)
	}

	_, errTo := s.currencyService.GetCurrencyByCode(ctx, req.ToCurrencyCode)
	if errTo != nil {
		if errors.Is(errTo, apperrors.ErrNotFound) {
			return nil, fmt.Errorf("%w: 'to' currency code '%s' not found", apperrors.ErrValidation, req.ToCurrencyCode)
		}
		return nil, fmt.Errorf("failed to validate 'to' currency '%s': %w", req.ToCurrencyCode, errTo)
	}

	now := time.Now()
	newRateID := uuid.NewString()

	rate := models.ExchangeRate{
		ExchangeRateID:   newRateID,
		FromCurrencyCode: req.FromCurrencyCode,
		ToCurrencyCode:   req.ToCurrencyCode,
		Rate:             req.Rate,
		DateEffective:    req.DateEffective,
		AuditFields: models.AuditFields{
			CreatedAt:     now,
			CreatedBy:     creatorUserID,
			LastUpdatedAt: now,
			LastUpdatedBy: creatorUserID,
		},
	}

	err := s.rateRepo.SaveExchangeRate(ctx, rate)
	if err != nil {
		// TODO: Add structured logging
		return nil, fmt.Errorf("failed to create exchange rate in service: %w", err)
	}

	return &rate, nil
}

// GetExchangeRate retrieves a specific exchange rate for a given currency pair and date.
func (s *ExchangeRateService) GetExchangeRate(ctx context.Context, fromCode, toCode string) (*models.ExchangeRate, error) {
	// Basic validation for codes (length, case)
	fromCode = strings.ToUpper(fromCode)
	toCode = strings.ToUpper(toCode)
	if len(fromCode) != 3 || len(toCode) != 3 {
		return nil, fmt.Errorf("%w: currency codes must be 3 letters", apperrors.ErrValidation)
	}

	rate, err := s.rateRepo.FindExchangeRate(ctx, fromCode, toCode)
	if err != nil {
		// Repository layer handles ErrNotFound mapping
		// TODO: Add structured logging
		return nil, fmt.Errorf("failed to get exchange rate in service: %w", err)
	}

	// No need to check for nil if repository correctly returns apperrors.ErrNotFound
	return rate, nil
}

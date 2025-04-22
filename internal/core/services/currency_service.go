package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/SscSPs/money_managemet_app/internal/apperrors"
	"github.com/SscSPs/money_managemet_app/internal/core/ports"
	"github.com/SscSPs/money_managemet_app/internal/dto"
	"github.com/SscSPs/money_managemet_app/internal/middleware" // Import middleware
	"github.com/SscSPs/money_managemet_app/internal/models"
)

type CurrencyService struct {
	currencyRepo ports.CurrencyRepository
}

func NewCurrencyService(currencyRepo ports.CurrencyRepository) *CurrencyService {
	return &CurrencyService{currencyRepo: currencyRepo}
}

func (s *CurrencyService) CreateCurrency(ctx context.Context, req dto.CreateCurrencyRequest, creatorUserID string) (*models.Currency, error) {
	logger := middleware.GetLoggerFromCtx(ctx) // Get logger from context
	now := time.Now()

	currency := models.Currency{
		CurrencyCode: req.CurrencyCode,
		Symbol:       req.Symbol,
		Name:         req.Name,
		AuditFields: models.AuditFields{
			CreatedAt:     now,
			CreatedBy:     creatorUserID,
			LastUpdatedAt: now,
			LastUpdatedBy: creatorUserID,
		},
	}

	err := s.currencyRepo.SaveCurrency(ctx, currency)
	if err != nil {
		logger.Error("Failed to save currency in repository", slog.String("error", err.Error()), slog.String("currency_code", currency.CurrencyCode))
		return nil, fmt.Errorf("failed to create currency in service: %w", err)
	}

	logger.Info("Currency created successfully in service", slog.String("currency_code", currency.CurrencyCode))
	return &currency, nil
}

func (s *CurrencyService) GetCurrencyByCode(ctx context.Context, currencyCode string) (*models.Currency, error) {
	logger := middleware.GetLoggerFromCtx(ctx)
	currency, err := s.currencyRepo.FindCurrencyByCode(ctx, currencyCode)
	if err != nil {
		if !errors.Is(err, apperrors.ErrNotFound) {
			logger.Error("Failed to find currency by code in repository", slog.String("error", err.Error()), slog.String("currency_code", currencyCode))
		}
		return nil, err
	}

	logger.Debug("Currency retrieved successfully by code from service", slog.String("currency_code", currency.CurrencyCode))
	return currency, nil
}

func (s *CurrencyService) ListCurrencies(ctx context.Context) ([]models.Currency, error) {
	logger := middleware.GetLoggerFromCtx(ctx) // Get logger from context
	currencies, err := s.currencyRepo.ListCurrencies(ctx)
	if err != nil {
		logger.Error("Failed to list currencies in repository", slog.String("error", err.Error()))
		return nil, fmt.Errorf("failed to list currencies in service: %w", err)
	}

	if currencies == nil {
		logger.Debug("No currencies found, returning empty list.")
		return []models.Currency{}, nil
	}

	logger.Debug("Currencies listed successfully from service", slog.Int("count", len(currencies)))
	return currencies, nil
}

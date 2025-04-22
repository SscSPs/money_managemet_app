package services

import (
	"context"
	"fmt"
	"time" // Needed for AuditFields

	"github.com/SscSPs/money_managemet_app/internal/core/ports"
	"github.com/SscSPs/money_managemet_app/internal/dto"
	"github.com/SscSPs/money_managemet_app/internal/models"
)

type CurrencyService struct {
	currencyRepo ports.CurrencyRepository
}

func NewCurrencyService(currencyRepo ports.CurrencyRepository) *CurrencyService {
	return &CurrencyService{currencyRepo: currencyRepo}
}

func (s *CurrencyService) CreateCurrency(ctx context.Context, req dto.CreateCurrencyRequest, creatorUserID string) (*models.Currency, error) {
	// Basic validation already handled by DTO binding (required, len=3, uppercase)
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
		// TODO: Add structured logging
		// Could check for specific DB errors (e.g., constraint violations)
		return nil, fmt.Errorf("failed to create currency in service: %w", err)
	}

	return &currency, nil
}

func (s *CurrencyService) GetCurrencyByCode(ctx context.Context, currencyCode string) (*models.Currency, error) {
	currency, err := s.currencyRepo.FindCurrencyByCode(ctx, currencyCode)
	if err != nil {
		// TODO: Add structured logging
		return nil, fmt.Errorf("failed to get currency by code in service: %w", err)
	}
	if currency == nil {
		// Service layer could return a specific "not found" error type here
		return nil, nil // Or return a custom error e.g., ErrNotFound
	}
	return currency, nil
}

func (s *CurrencyService) ListCurrencies(ctx context.Context) ([]models.Currency, error) {
	currencies, err := s.currencyRepo.ListCurrencies(ctx)
	if err != nil {
		// TODO: Add structured logging
		return nil, fmt.Errorf("failed to list currencies in service: %w", err)
	}
	// Return empty slice if no currencies found, not nil
	if currencies == nil {
		return []models.Currency{}, nil
	}
	return currencies, nil
}

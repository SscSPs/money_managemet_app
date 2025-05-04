package mapping

import (
	"github.com/SscSPs/money_managemet_app/internal/core/domain"
	"github.com/SscSPs/money_managemet_app/internal/models"
)

// ToModelCurrency converts a domain Currency to a model Currency
func ToModelCurrency(d domain.Currency) models.Currency {
	return models.Currency{
		CurrencyCode: d.CurrencyCode,
		Symbol:       d.Symbol,
		Name:         d.Name,
		Precision:    d.Precision,
		AuditFields: models.AuditFields{
			CreatedAt:     d.CreatedAt,
			CreatedBy:     d.CreatedBy,
			LastUpdatedAt: d.LastUpdatedAt,
			LastUpdatedBy: d.LastUpdatedBy,
		},
	}
}

// ToDomainCurrency converts a model Currency to a domain Currency
func ToDomainCurrency(m models.Currency) domain.Currency {
	return domain.Currency{
		CurrencyCode: m.CurrencyCode,
		Symbol:       m.Symbol,
		Name:         m.Name,
		Precision:    m.Precision,
		AuditFields: domain.AuditFields{
			CreatedAt:     m.CreatedAt,
			CreatedBy:     m.CreatedBy,
			LastUpdatedAt: m.LastUpdatedAt,
			LastUpdatedBy: m.LastUpdatedBy,
		},
	}
}

// ToDomainCurrencySlice converts a slice of model Currencies to a slice of domain Currencies
func ToDomainCurrencySlice(ms []models.Currency) []domain.Currency {
	ds := make([]domain.Currency, len(ms))
	for i, m := range ms {
		ds[i] = ToDomainCurrency(m)
	}
	return ds
}

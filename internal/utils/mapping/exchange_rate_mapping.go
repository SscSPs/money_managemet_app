package mapping

import (
	"github.com/SscSPs/money_managemet_app/internal/core/domain"
	"github.com/SscSPs/money_managemet_app/internal/models"
)

// ToModelExchangeRate converts a domain ExchangeRate to a model ExchangeRate
func ToModelExchangeRate(d domain.ExchangeRate) models.ExchangeRate {
	return models.ExchangeRate{
		ExchangeRateID:   d.ExchangeRateID,
		FromCurrencyCode: d.FromCurrencyCode,
		ToCurrencyCode:   d.ToCurrencyCode,
		Rate:             d.Rate,
		DateEffective:    d.DateEffective,
		AuditFields:      ToModelAuditFields(d.AuditFields),
	}
}

// ToDomainExchangeRate converts a model ExchangeRate to a domain ExchangeRate
func ToDomainExchangeRate(m models.ExchangeRate) domain.ExchangeRate {
	return domain.ExchangeRate{
		ExchangeRateID:   m.ExchangeRateID,
		FromCurrencyCode: m.FromCurrencyCode,
		ToCurrencyCode:   m.ToCurrencyCode,
		Rate:             m.Rate,
		DateEffective:    m.DateEffective,
		AuditFields:      ToDomainAuditFields(m.AuditFields),
	}
}

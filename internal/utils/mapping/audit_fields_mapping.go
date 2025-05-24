package mapping

import (
	"github.com/SscSPs/money_managemet_app/internal/core/domain"
	"github.com/SscSPs/money_managemet_app/internal/models"
)

// ToModelAuditFields converts a domain AuditFields to a model AuditFields
func ToModelAuditFields(d domain.AuditFields) models.AuditFields {
	return models.AuditFields{
		CreatedAt:     d.CreatedAt,
		CreatedBy:     d.CreatedBy,
		LastUpdatedAt: d.LastUpdatedAt,
		LastUpdatedBy: d.LastUpdatedBy,
		Version:       d.Version,
	}
}

// ToDomainAuditFields converts a model AuditFields to a domain AuditFields
func ToDomainAuditFields(m models.AuditFields) domain.AuditFields {
	return domain.AuditFields{
		CreatedAt:     m.CreatedAt,
		CreatedBy:     m.CreatedBy,
		LastUpdatedAt: m.LastUpdatedAt,
		LastUpdatedBy: m.LastUpdatedBy,
		Version:       m.Version,
	}
}

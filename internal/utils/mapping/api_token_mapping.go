package mapping

import (
	"github.com/SscSPs/money_managemet_app/internal/core/domain"
	"github.com/SscSPs/money_managemet_app/internal/models"
)

// ToModelAPIToken converts a domain APIToken to a model APIToken
func ToModelAPIToken(d domain.APIToken) models.APIToken {
	return models.APIToken{
		ID:         d.ID,
		UserID:     d.UserID,
		Name:       d.Name,
		TokenHash:  d.TokenHash,
		LastUsedAt: d.LastUsedAt,
		ExpiresAt:  d.ExpiresAt,
		CreatedAt:  d.CreatedAt,
		UpdatedAt:  d.UpdatedAt,
	}
}

// ToDomainAPIToken converts a model APIToken to a domain APIToken
func ToDomainAPIToken(m models.APIToken) domain.APIToken {
	return domain.APIToken{
		ID:         m.ID,
		UserID:     m.UserID,
		Name:       m.Name,
		TokenHash:  m.TokenHash,
		LastUsedAt: m.LastUsedAt,
		ExpiresAt:  m.ExpiresAt,
		CreatedAt:  m.CreatedAt,
		UpdatedAt:  m.UpdatedAt,
	}
}

// ToDomainAPITokenSlice converts a slice of model APITokens to a slice of domain APITokens
func ToDomainAPITokenSlice(ms []models.APIToken) []domain.APIToken {
	ds := make([]domain.APIToken, len(ms))
	for i, m := range ms {
		ds[i] = ToDomainAPIToken(m)
	}
	return ds
}

// MapAPITokenModelToDomain is an alias for ToDomainAPIToken for backward compatibility
func MapAPITokenModelToDomain(m models.APIToken) domain.APIToken {
	return ToDomainAPIToken(m)
}

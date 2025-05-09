package mapping

import (
	"database/sql"
	"github.com/SscSPs/money_managemet_app/internal/core/domain"
	"github.com/SscSPs/money_managemet_app/internal/models"
)

// ToModelUser converts a domain User to a model User
func ToModelUser(d domain.User) models.User {
	model := models.User{
		UserID:       d.UserID,
		Name:         d.Name,
		Username:     d.Username,
		PasswordHash: d.PasswordHash,
		AuditFields: models.AuditFields{
			CreatedAt:     d.CreatedAt,
			CreatedBy:     d.CreatedBy,
			LastUpdatedAt: d.LastUpdatedAt,
			LastUpdatedBy: d.LastUpdatedBy,
		},
		DeletedAt: d.DeletedAt,
	}
	if d.RefreshTokenHash != "" {
		model.RefreshTokenHash = sql.NullString{String: d.RefreshTokenHash, Valid: true}
	}
	if d.RefreshTokenExpiryTime != nil {
		model.RefreshTokenExpiryTime = sql.NullTime{Time: *d.RefreshTokenExpiryTime, Valid: true}
	}
	return model
}

// ToDomainUser converts a model User to a domain User
func ToDomainUser(m models.User) domain.User {
	domainUser := domain.User{
		UserID:       m.UserID,
		Name:         m.Name,
		Username:     m.Username,
		PasswordHash: m.PasswordHash,
		AuditFields: domain.AuditFields{
			CreatedAt:     m.CreatedAt,
			CreatedBy:     m.CreatedBy,
			LastUpdatedAt: m.LastUpdatedAt,
			LastUpdatedBy: m.LastUpdatedBy,
		},
		DeletedAt: m.DeletedAt,
	}
	if m.RefreshTokenHash.Valid {
		domainUser.RefreshTokenHash = m.RefreshTokenHash.String
	}
	if m.RefreshTokenExpiryTime.Valid {
		domainUser.RefreshTokenExpiryTime = &m.RefreshTokenExpiryTime.Time
	}
	return domainUser
}

// ToDomainUserSlice converts a slice of model Users to a slice of domain Users
func ToDomainUserSlice(ms []models.User) []domain.User {
	ds := make([]domain.User, len(ms))
	for i, m := range ms {
		ds[i] = ToDomainUser(m)
	}
	return ds
}

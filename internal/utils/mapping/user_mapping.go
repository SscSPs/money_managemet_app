package mapping

import (
	"database/sql"

	"github.com/SscSPs/money_managemet_app/internal/core/domain"
	"github.com/SscSPs/money_managemet_app/internal/dto"
	"github.com/SscSPs/money_managemet_app/internal/models"
)

// ToModelUser converts a domain User to a model User
func ToModelUser(d domain.User) models.User {
	model := models.User{
		UserID:           d.UserID,
		Username:         d.Username,
		Email:            sql.NullString{String: d.Email, Valid: d.Email != ""}, // Email is valid if not an empty string
		Name:             d.Name,
		AuthProvider:     sql.NullString{String: string(d.AuthProvider), Valid: string(d.AuthProvider) != ""}, // AuthProvider is valid if not an empty string
		ProviderUserID:   sql.NullString{String: d.ProviderUserID, Valid: d.ProviderUserID != ""},             // ProviderUserID is valid if not an empty string
		RefreshTokenHash: sql.NullString{String: d.RefreshTokenHash, Valid: d.RefreshTokenHash != ""},         // RefreshTokenHash is valid if not an empty string
		AuditFields: models.AuditFields{
			CreatedAt:     d.CreatedAt,
			CreatedBy:     d.CreatedBy,
			LastUpdatedAt: d.LastUpdatedAt,
			LastUpdatedBy: d.LastUpdatedBy,
			Version:       d.Version,
		},
		DeletedAt: d.DeletedAt,
	}

	if d.PasswordHash != nil {
		model.PasswordHash = sql.NullString{String: *d.PasswordHash, Valid: true}
	} else {
		model.PasswordHash = sql.NullString{Valid: false} // Represents SQL NULL
	}

	if d.RefreshTokenExpiryTime != nil {
		model.RefreshTokenExpiryTime = sql.NullTime{Time: *d.RefreshTokenExpiryTime, Valid: true}
	} else {
		model.RefreshTokenExpiryTime = sql.NullTime{Valid: false} // Represents SQL NULL
	}

	return model
}

// ToDomainUser converts a model User to a domain User
func ToDomainUser(m models.User) domain.User {
	domainUser := domain.User{
		UserID:                 m.UserID,
		Username:               m.Username,
		Email:                  m.Email.String,
		Name:                   m.Name,
		PasswordHash:           &m.PasswordHash.String,
		AuthProvider:           domain.AuthProviderType(m.AuthProvider.String),
		ProviderUserID:         m.ProviderUserID.String,
		RefreshTokenHash:       m.RefreshTokenHash.String,
		RefreshTokenExpiryTime: &m.RefreshTokenExpiryTime.Time,
		AuditFields: domain.AuditFields{
			CreatedAt:     m.CreatedAt,
			CreatedBy:     m.CreatedBy,
			LastUpdatedAt: m.LastUpdatedAt,
			LastUpdatedBy: m.LastUpdatedBy,
			Version:       m.Version,
		},
		DeletedAt: m.DeletedAt,
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

// UserToUserResponse maps a domain.User to a dto.UserResponse.
func UserToUserResponse(user *domain.User) *dto.UserResponse {
	if user == nil {
		return nil
	}
	return &dto.UserResponse{
		UserID:   user.UserID,
		Username: user.Username,
		Name:     user.Name,
	}
}

// UsersToUserResponses maps a slice of domain.User to a slice of dto.UserResponse.
func UsersToUserResponses(users []*domain.User) []*dto.UserResponse {
	if users == nil {
		return nil
	}
	userResponses := make([]*dto.UserResponse, len(users))
	for i, user := range users {
		userResponses[i] = UserToUserResponse(user)
	}
	return userResponses
}

// ToUserMeResponse maps a domain.User to a dto.UserMeResponse.
func ToUserMeResponse(user *domain.User) *dto.UserMeResponse {
	if user == nil {
		return nil
	}
	return &dto.UserMeResponse{
		UserID:    user.UserID,
		Username:  user.Username,
		Email:     &user.Email,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.LastUpdatedAt,
	}
}

// CreateUserRequestToUser maps a dto.CreateUserRequest to a domain.User.
// Note: Password hashing should be handled by the service layer.
func CreateUserRequestToUser(req *dto.CreateUserRequest) *domain.User {
	if req == nil {
		return nil
	}
	user := &domain.User{
		Username: req.Username,
		Name:     req.Name,
		// Password field from request is used by service to create PasswordHash
	}
	return user
}

// UpdateUserRequestToUser maps a dto.UpdateUserRequest to a domain.User.
// It's important to handle partial updates appropriately in the service layer.
func UpdateUserRequestToUser(userID string, req *dto.UpdateUserRequest) *domain.User {
	user := &domain.User{
		UserID: userID,
	}

	if req.Name != nil {
		user.Name = *req.Name
	}

	return user
}

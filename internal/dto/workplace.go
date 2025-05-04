package dto

import (
	"time"

	"github.com/SscSPs/money_managemet_app/internal/core/domain"
)

// --- Workplace DTOs ---

// CreateWorkplaceRequest defines data for creating a new workplace.
type CreateWorkplaceRequest struct {
	Name                string `json:"name" binding:"required"`
	Description         string `json:"description"`
	DefaultCurrencyCode string `json:"defaultCurrencyCode" binding:"required,iso4217"`
}

// WorkplaceResponse defines data returned for a workplace.
type WorkplaceResponse struct {
	WorkplaceID         string    `json:"workplaceID"`
	Name                string    `json:"name"`
	Description         string    `json:"description"`
	DefaultCurrencyCode *string   `json:"defaultCurrencyCode,omitempty"`
	IsActive            bool      `json:"isActive"`
	CreatedAt           time.Time `json:"createdAt"`
	CreatedBy           string    `json:"createdBy"` // UserID
	LastUpdatedAt       time.Time `json:"lastUpdatedAt"`
	LastUpdatedBy       string    `json:"lastUpdatedBy"` // UserID
}

// ToWorkplaceResponse converts domain.Workplace to DTO.
func ToWorkplaceResponse(w *domain.Workplace) WorkplaceResponse {
	return WorkplaceResponse{
		WorkplaceID:         w.WorkplaceID,
		Name:                w.Name,
		Description:         w.Description,
		DefaultCurrencyCode: w.DefaultCurrencyCode,
		IsActive:            w.IsActive,
		CreatedAt:           w.CreatedAt,
		CreatedBy:           w.CreatedBy,
		LastUpdatedAt:       w.LastUpdatedAt,
		LastUpdatedBy:       w.LastUpdatedBy,
	}
}

// ListWorkplacesResponse wraps a list of workplaces.
type ListWorkplacesResponse struct {
	Workplaces []WorkplaceResponse `json:"workplaces"`
}

// ToListWorkplacesResponse converts a slice of domain.Workplace to DTO.
func ToListWorkplacesResponse(ws []domain.Workplace) ListWorkplacesResponse {
	list := make([]WorkplaceResponse, len(ws))
	for i, w := range ws {
		list[i] = ToWorkplaceResponse(&w)
	}
	return ListWorkplacesResponse{Workplaces: list}
}

// --- User Workplace Membership DTOs ---

// AddUserToWorkplaceRequest defines data for adding a user to a workplace.
type AddUserToWorkplaceRequest struct {
	UserID string                   `json:"userID" binding:"required"`
	Role   domain.UserWorkplaceRole `json:"role" binding:"required,oneof=ADMIN MEMBER REMOVED"`
}

// UserWorkplaceResponse defines data returned about a user's membership.
type UserWorkplaceResponse struct {
	UserID      string                   `json:"userID"`
	UserName    string                   `json:"userName"` // User's name
	WorkplaceID string                   `json:"workplaceID"`
	Role        domain.UserWorkplaceRole `json:"role"`
	JoinedAt    time.Time                `json:"joinedAt"`
}

// ToUserWorkplaceResponse converts domain.UserWorkplace to DTO.
func ToUserWorkplaceResponse(uw *domain.UserWorkplace) UserWorkplaceResponse {
	return UserWorkplaceResponse{
		UserID:      uw.UserID,
		UserName:    uw.UserName,
		WorkplaceID: uw.WorkplaceID,
		Role:        uw.Role,
		JoinedAt:    uw.JoinedAt,
	}
}

// ListUserWorkplacesParams defines parameters for listing user workplaces.
type ListUserWorkplacesParams struct {
	IncludeDisabled bool `form:"includeDisabled" json:"includeDisabled"`
}

// DeactivateWorkplaceRequest defines data for deactivating a workplace.
type DeactivateWorkplaceRequest struct {
	// Can be expanded later to include additional information
	// such as a reason for deactivation or a timestamp until when
	// the workplace should remain inactive
}

// ListWorkplaceUsersResponse wraps a list of users for a workplace.
type ListWorkplaceUsersResponse struct {
	Users []UserWorkplaceResponse `json:"users"`
}

// ToListWorkplaceUsersResponse converts a slice of domain.UserWorkplace to DTO.
func ToListWorkplaceUsersResponse(userWorkplaces []domain.UserWorkplace) ListWorkplaceUsersResponse {
	list := make([]UserWorkplaceResponse, len(userWorkplaces))
	for i, uw := range userWorkplaces {
		list[i] = ToUserWorkplaceResponse(&uw)
	}
	return ListWorkplaceUsersResponse{Users: list}
}

// UpdateUserRoleRequest defines the request structure for updating a user's role in a workplace.
type UpdateUserRoleRequest struct {
	Role domain.UserWorkplaceRole `json:"role" binding:"required,oneof=ADMIN MEMBER REMOVED"`
}

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
	Role   domain.UserWorkplaceRole `json:"role" binding:"required,oneof=ADMIN MEMBER"`
}

// UserWorkplaceResponse defines data returned about a user's membership.
type UserWorkplaceResponse struct {
	UserID      string                   `json:"userID"`
	WorkplaceID string                   `json:"workplaceID"`
	Role        domain.UserWorkplaceRole `json:"role"`
	JoinedAt    time.Time                `json:"joinedAt"`
}

// ToUserWorkplaceResponse converts domain.UserWorkplace to DTO.
func ToUserWorkplaceResponse(uw *domain.UserWorkplace) UserWorkplaceResponse {
	return UserWorkplaceResponse{
		UserID:      uw.UserID,
		WorkplaceID: uw.WorkplaceID,
		Role:        uw.Role,
		JoinedAt:    uw.JoinedAt,
	}
}

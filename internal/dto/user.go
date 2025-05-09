package dto

import (
	"github.com/SscSPs/money_managemet_app/internal/core/domain"
)

// UpdateUserRequest defines the data allowed for updating a user.
// Using pointers to differentiate between omitted fields and zero-value fields.
type UpdateUserRequest struct {
	Name *string `json:"name"` // Only name is updatable for now
}

// ListUsersParams defines query parameters for listing users.
type ListUsersParams struct {
	Limit  int `form:"limit,default=20"`
	Offset int `form:"offset,default=0"`
}

// ListUsersResponse wraps the list of users.
type ListUsersResponse struct {
	Users []UserResponse `json:"users"`
	// TODO: Add pagination metadata (total count, limit, offset) later
}

// ToListUserResponse converts a slice of domain.User to ListUsersResponse DTO
func ToListUserResponse(users []domain.User) ListUsersResponse {
	userResponses := make([]UserResponse, len(users))
	for i, user := range users {
		userResponses[i] = ToUserResponse(&user)
	}
	return ListUsersResponse{
		Users: userResponses,
	}
}

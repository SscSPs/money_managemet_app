package dto

import "time"

type UserResponse struct {
	UserID   string `json:"userID"`
	Username string `json:"username"`
	Name     string `json:"name"`
}

// UserMeResponse defines the structure for the /auth/me endpoint response.
// It should contain non-sensitive user information.
type UserMeResponse struct {
	UserID    string    `json:"user_id"`
	Username  string    `json:"username"`
	Email     *string   `json:"email,omitempty"` // Pointer to allow null/omitted if not set
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func ToUserResponse(user interface {
	GetUserID() string
	GetUsername() string
	GetName() string
}) UserResponse {
	return UserResponse{
		UserID:   user.GetUserID(),
		Username: user.GetUsername(),
		Name:     user.GetName(),
	}
}

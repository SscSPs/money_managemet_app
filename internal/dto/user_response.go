package dto

type UserResponse struct {
	UserID   string `json:"userID"`
	Username string `json:"username"`
	Name     string `json:"name"`
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

package dto

// LoginResponse represents the response for a successful login.
type LoginResponse struct {
	Token string `json:"token"`
}

// RefreshTokenResponse represents the response for a successful token refresh.
type RefreshTokenResponse struct {
	Token string `json:"token"`
}

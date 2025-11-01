package dto

import (
	"time"

	"github.com/SscSPs/money_managemet_app/internal/core/domain"
)

// CreateAPITokenRequest represents the request body for creating a new API token
type CreateAPITokenRequest struct {
	Name      string         `json:"name" binding:"required,min=3,max=100"`
	ExpiresIn *time.Duration `json:"expiresIn,omitempty"` // Duration in seconds
}

// APITokenResponse represents an API token in the API responses
type APITokenResponse struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	LastUsedAt *time.Time `json:"lastUsedAt,omitempty"`
	ExpiresAt  *time.Time `json:"expiresAt,omitempty"`
	CreatedAt  time.Time  `json:"createdAt"`
}

// CreateAPITokenResponse represents the response when creating a new API token
type CreateAPITokenResponse struct {
	TokenString string           `json:"token"` // Only shown once when created
	Details     APITokenResponse `json:"details"`
}

// ListAPITokensResponse represents a list of API tokens
type ListAPITokensResponse []APITokenResponse

// ToAPITokenResponse converts a domain.APIToken to an APITokenResponse
func ToAPITokenResponse(token domain.APIToken) APITokenResponse {
	return APITokenResponse{
		ID:         token.ID,
		Name:       token.Name,
		LastUsedAt: token.LastUsedAt,
		ExpiresAt:  token.ExpiresAt,
		CreatedAt:  token.CreatedAt,
	}
}

// ToAPITokenResponseList converts a slice of domain.APIToken to ListAPITokensResponse
func ToAPITokenResponseList(tokens []domain.APIToken) ListAPITokensResponse {
	result := make(ListAPITokensResponse, len(tokens))
	for i, token := range tokens {
		result[i] = ToAPITokenResponse(token)
	}
	return result
}

// ToCreateAPITokenResponse converts a token string and domain.APIToken to CreateAPITokenResponse
func ToCreateAPITokenResponse(tokenStr string, token domain.APIToken) CreateAPITokenResponse {
	return CreateAPITokenResponse{
		TokenString: tokenStr,
		Details:     ToAPITokenResponse(token),
	}
}

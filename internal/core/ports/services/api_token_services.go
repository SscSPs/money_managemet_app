package services

import (
	"context"
	"time"

	"github.com/SscSPs/money_managemet_app/internal/core/domain"
)

// APITokenSvc defines operations for API token management
type APITokenSvc interface {
	// CreateToken generates a new API token for the user
	// Returns the plaintext token (only shown once) and the token details
	CreateToken(ctx context.Context, userID, name string, expiresIn *time.Duration) (string, *domain.APIToken, error)

	// ListTokens returns all API tokens for a user
	ListTokens(ctx context.Context, userID string) ([]domain.APIToken, error)

	// RevokeToken deletes a specific API token for a user
	RevokeToken(ctx context.Context, userID, tokenID string) error

	// RevokeAllTokens deletes all API tokens for a user
	RevokeAllTokens(ctx context.Context, userID string) error

	// ValidateToken checks if a token is valid and returns the associated user
	// Updates the last_used_at timestamp if the token is valid
	ValidateToken(ctx context.Context, tokenString string) (*domain.User, error)
}

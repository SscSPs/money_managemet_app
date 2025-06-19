package repositories

import (
	"context"
	"time"

	"github.com/SscSPs/money_managemet_app/internal/core/domain"
)

// APITokenRepository defines the interface for API token data access operations
type APITokenRepository interface {
	// Create persists a new API token
	Create(ctx context.Context, token *domain.APIToken) error

	// FindByID retrieves an API token by its ID
	FindByID(ctx context.Context, id string) (*domain.APIToken, error)

	// FindByUserID retrieves all API tokens for a specific user
	FindByUserID(ctx context.Context, userID string) ([]domain.APIToken, error)

	// FindByToken finds a token by its hash (used for validation)
	FindByToken(ctx context.Context, tokenString string) (*domain.APIToken, error)

	// Update updates an existing API token (e.g., to update last_used_at)
	Update(ctx context.Context, token *domain.APIToken) error

	// Delete removes an API token by ID
	Delete(ctx context.Context, id string) error

	// DeleteByUserID removes all API tokens for a specific user
	DeleteByUserID(ctx context.Context, userID string) error

	// DeleteExpired removes all expired API tokens
	DeleteExpired(ctx context.Context, before time.Time) (int64, error)
}

// APITokenRepositoryWithTx extends APITokenRepository with transaction capabilities
type APITokenRepositoryWithTx interface {
	APITokenRepository
	WithTx(tx interface{}) APITokenRepository
}

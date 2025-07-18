package repositories

import (
	"context"
	"time"

	"github.com/SscSPs/money_managemet_app/internal/core/domain"
)

// UserReader defines read operations for user data
type UserReader interface {
	// FindUserByID retrieves a specific user by their ID.
	FindUserByID(ctx context.Context, userID string) (*domain.User, error)

	// FindUserByUsername retrieves a user by their username.
	FindUserByUsername(ctx context.Context, username string) (*domain.User, error)

	// FindUserByEmail retrieves a user by their email address.
	FindUserByEmail(ctx context.Context, email string) (*domain.User, error)

	// FindUserByProviderDetails retrieves a user by their authentication provider and provider-specific ID.
	FindUserByProviderDetails(ctx context.Context, authProvider string, providerUserID string) (*domain.User, error)

	// FindUsers retrieves a paginated list of users.
	FindUsers(ctx context.Context, limit int, offset int) ([]domain.User, error)
}

// UserWriter defines write operations for user data
type UserWriter interface {
	// SaveUser persists a new user.
	SaveUser(ctx context.Context, user *domain.User) error

	// UpdateUser updates an existing user's details.
	UpdateUser(ctx context.Context, user *domain.User) error

	// UpdateRefreshToken updates the refresh token details for a user.
	UpdateRefreshToken(ctx context.Context, existingUser *domain.User, refreshTokenHash string, refreshTokenExpiryTime time.Time) error

	// ClearRefreshToken clears the refresh token for a user.
	ClearRefreshToken(ctx context.Context, existingUser *domain.User) error
}

// UserLifecycleManager defines operations for managing user lifecycle
type UserLifecycleManager interface {
	// MarkUserDeleted marks a user as deleted (soft delete).
	MarkUserDeleted(ctx context.Context, user *domain.User, deletedBy string) error
}

// UserRepositoryFacade combines all user-related repository interfaces
// This is a facade for clients that need access to all operations
type UserRepositoryFacade interface {
	UserReader
	UserWriter
	UserLifecycleManager
}

// UserRepositoryWithTx extends UserRepositoryFacade with transaction capabilities
type UserRepositoryWithTx interface {
	UserRepositoryFacade
	TransactionManager
}

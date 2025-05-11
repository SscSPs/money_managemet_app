package services

import (
	"context"
	"time"

	"github.com/SscSPs/money_managemet_app/internal/core/domain"
	"github.com/SscSPs/money_managemet_app/internal/dto"
)

// UserReaderSvc defines read operations for user data
type UserReaderSvc interface {
	// GetUserByID retrieves a user by ID.
	GetUserByID(ctx context.Context, userID string) (*domain.User, error)

	// GetUserByUsername retrieves a user by username.
	GetUserByUsername(ctx context.Context, username string) (*domain.User, error)

	// FindUserByEmail retrieves a user by email.
	FindUserByEmail(ctx context.Context, email string) (*domain.User, error)

	// FindUserByProviderDetails retrieves a user by authentication provider (using AuthProviderType) and provider-specific user ID.
	FindUserByProvider(ctx context.Context, provider domain.AuthProviderType, providerUserID string) (*domain.User, error)

	// ListUsers retrieves a paginated list of users.
	ListUsers(ctx context.Context, limit, offset int) ([]domain.User, error)
}

// UserWriterSvc defines write operations for user data
type UserWriterSvc interface {
	// CreateUser creates a new user.
	CreateUser(ctx context.Context, req dto.CreateUserRequest) (*domain.User, error)

	// CreateOAuthUser creates a new user authenticating via an OAuth provider.
	// This might be refactorable or removable if CreateUser handles all cases with a domain.User object.
	CreateOAuthUser(ctx context.Context, name, email, authProvider, providerUserID string, emailVerified bool) (*domain.User, error)

	// UpdateUser updates an existing user.
	UpdateUser(ctx context.Context, userID string, req dto.UpdateUserRequest, requestingUserID string) (*domain.User, error)

	// UpdateUserProviderDetails updates a user's provider-specific information.
	UpdateUserProviderDetails(ctx context.Context, userID string, details domain.UpdateUserProviderDetails) (*domain.User, error)

	// UpdateRefreshToken updates the refresh token details for a user.
	UpdateRefreshToken(ctx context.Context, userID string, refreshTokenHash string, refreshTokenExpiryTime time.Time) error

	// ClearRefreshToken clears the refresh token for a user.
	ClearRefreshToken(ctx context.Context, userID string) error
}

// UserLifecycleSvc defines operations for managing user lifecycle
type UserLifecycleSvc interface {
	// DeleteUser marks a user as deleted (soft delete).
	DeleteUser(ctx context.Context, userID string, requestingUserID string) error
}

// UserAuthSvc defines operations for user authentication
type UserAuthSvc interface {
	// AuthenticateUser authenticates a user with email and password.
	AuthenticateUser(ctx context.Context, email, password string) (*domain.User, error)
}

// UserSvcFacade combines all user-related service interfaces
type UserSvcFacade interface {
	UserReaderSvc
	UserWriterSvc
	UserLifecycleSvc
	UserAuthSvc
}

package services

import (
	"context"

	"github.com/SscSPs/money_managemet_app/internal/core/domain"
	"github.com/SscSPs/money_managemet_app/internal/dto"
)

// UserReaderSvc defines read operations for user data
type UserReaderSvc interface {
	// GetUserByID retrieves a user by ID.
	GetUserByID(ctx context.Context, userID string) (*domain.User, error)

	// ListUsers retrieves a paginated list of users.
	ListUsers(ctx context.Context, limit, offset int) ([]domain.User, error)
}

// UserWriterSvc defines write operations for user data
type UserWriterSvc interface {
	// CreateUser creates a new user.
	CreateUser(ctx context.Context, req dto.CreateUserRequest) (*domain.User, error)

	// UpdateUser updates an existing user.
	UpdateUser(ctx context.Context, userID string, req dto.UpdateUserRequest, requestingUserID string) (*domain.User, error)
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

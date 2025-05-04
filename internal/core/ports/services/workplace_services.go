package services

import (
	"context"

	"github.com/SscSPs/money_managemet_app/internal/core/domain"
)

// WorkplaceReaderSvc defines read operations for workplace data
type WorkplaceReaderSvc interface {
	// FindWorkplaceByID retrieves a specific workplace by its ID.
	FindWorkplaceByID(ctx context.Context, workplaceID string) (*domain.Workplace, error)

	// ListUserWorkplaces retrieves workplaces a user belongs to with filtering options.
	// If includeDisabled is true, it includes inactive workplaces.
	// If roleFilter is provided, it only returns workplaces where the user has that specific role.
	ListUserWorkplaces(ctx context.Context, userID string, includeDisabled bool) ([]domain.Workplace, error)

	// ListWorkplaceUsers retrieves all users and their roles for a specific workplace.
	// Only authorized users (members of the workplace) can access this data.
	ListWorkplaceUsers(ctx context.Context, workplaceID string, requestingUserID string) ([]domain.UserWorkplace, error)
}

// WorkplaceWriterSvc defines write operations for workplace data
type WorkplaceWriterSvc interface {
	// CreateWorkplace persists a new workplace.
	CreateWorkplace(ctx context.Context, name, description, defaultCurrencyCode, creatorUserID string) (*domain.Workplace, error)

	// DeactivateWorkplace marks a workplace as inactive.
	DeactivateWorkplace(ctx context.Context, workplaceID string, requestingUserID string) error

	// ActivateWorkplace marks a workplace as active.
	ActivateWorkplace(ctx context.Context, workplaceID string, requestingUserID string) error
}

// WorkplaceMembershipSvc defines operations for managing workplace membership
type WorkplaceMembershipSvc interface {
	// AddUserToWorkplace adds a user to a workplace with a specific role.
	AddUserToWorkplace(ctx context.Context, addingUserID, targetUserID, workplaceID string, role domain.UserWorkplaceRole) error

	// RemoveUserFromWorkplace removes a user from a workplace.
	// Only workplace admins can remove users from a workplace.
	RemoveUserFromWorkplace(ctx context.Context, requestingUserID, targetUserID, workplaceID string) error

	// UpdateUserWorkplaceRole updates a user's role in a workplace.
	// Only workplace admins can update user roles.
	UpdateUserWorkplaceRole(ctx context.Context, requestingUserID, targetUserID, workplaceID string, newRole domain.UserWorkplaceRole) error
}

// WorkplaceAuthorizerSvc defines operations for workplace authorization
type WorkplaceAuthorizerSvc interface {
	// AuthorizeUserAction checks if a user has required permissions for a workplace.
	AuthorizeUserAction(ctx context.Context, userID, workplaceID string, requiredRole domain.UserWorkplaceRole) error
}

// WorkplaceSvcFacade combines all workplace-related service interfaces
// This is a facade for clients that need access to all operations
type WorkplaceSvcFacade interface {
	WorkplaceReaderSvc
	WorkplaceWriterSvc
	WorkplaceMembershipSvc
	WorkplaceAuthorizerSvc
}

package repositories

import (
	"context"

	"github.com/SscSPs/money_managemet_app/internal/core/domain"
)

// WorkplaceReader defines read operations for workplace data
type WorkplaceReader interface {
	// FindWorkplaceByID retrieves a specific workplace by its ID.
	FindWorkplaceByID(ctx context.Context, workplaceID string) (*domain.Workplace, error)

	// ListWorkplacesByUserID retrieves workplaces a user belongs to with filtering options.
	// If includeDisabled is true, it includes inactive workplaces.
	// If role is provided, it only returns workplaces where the user has that specific role.
	ListWorkplacesByUserID(ctx context.Context, userID string, includeDisabled bool, role *domain.UserWorkplaceRole) ([]domain.Workplace, error)

	// ListUsersByWorkplaceID retrieves all users that belong to a specific workplace.
	// By default, it excludes users with the REMOVED role.
	// Set includeRemoved to true to include users with the REMOVED role.
	ListUsersByWorkplaceID(ctx context.Context, workplaceID string, includeRemoved ...bool) ([]domain.UserWorkplace, error)
}

// WorkplaceWriter defines write operations for workplace data
type WorkplaceWriter interface {
	// SaveWorkplace persists a new workplace.
	SaveWorkplace(ctx context.Context, workplace domain.Workplace) error

	// UpdateWorkplaceStatus changes the is_active status of a workplace.
	UpdateWorkplaceStatus(ctx context.Context, workplaceID string, isActive bool, updatedByUserID string) error
}

// WorkplaceMembershipManager defines operations for managing workplace memberships
type WorkplaceMembershipManager interface {
	// AddUserToWorkplace adds a user to a workplace with a specific role.
	AddUserToWorkplace(ctx context.Context, membership domain.UserWorkplace) error

	// FindUserWorkplaceRole retrieves the role of a user in a workplace.
	FindUserWorkplaceRole(ctx context.Context, userID, workplaceID string) (*domain.UserWorkplace, error)

	// RemoveUserFromWorkplace removes a user from a workplace.
	RemoveUserFromWorkplace(ctx context.Context, userID, workplaceID string) error

	// UpdateUserWorkplaceRole updates a user's role in a workplace.
	UpdateUserWorkplaceRole(ctx context.Context, userID, workplaceID string, newRole domain.UserWorkplaceRole) error
}

// WorkplaceRepositoryFacade combines all workplace-related repository interfaces
// This is a facade for clients that need access to all operations
type WorkplaceRepositoryFacade interface {
	WorkplaceReader
	WorkplaceWriter
	WorkplaceMembershipManager
}

// WorkplaceRepositoryWithTx extends WorkplaceRepositoryFacade with transaction capabilities
type WorkplaceRepositoryWithTx interface {
	WorkplaceRepositoryFacade
	TransactionManager
}

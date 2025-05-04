package repositories

import (
	"context"

	"github.com/SscSPs/money_managemet_app/internal/core/domain"
)

// WorkplaceReader defines read operations for workplace data
type WorkplaceReader interface {
	// FindWorkplaceByID retrieves a specific workplace by its ID.
	FindWorkplaceByID(ctx context.Context, workplaceID string) (*domain.Workplace, error)

	// ListWorkplacesByUserID retrieves all workplaces a user belongs to.
	ListWorkplacesByUserID(ctx context.Context, userID string) ([]domain.Workplace, error)
}

// WorkplaceWriter defines write operations for workplace data
type WorkplaceWriter interface {
	// SaveWorkplace persists a new workplace.
	SaveWorkplace(ctx context.Context, workplace domain.Workplace) error
}

// WorkplaceMembershipManager defines operations for managing workplace memberships
type WorkplaceMembershipManager interface {
	// AddUserToWorkplace adds a user to a workplace with a specific role.
	AddUserToWorkplace(ctx context.Context, membership domain.UserWorkplace) error

	// FindUserWorkplaceRole retrieves the role of a user in a workplace.
	FindUserWorkplaceRole(ctx context.Context, userID, workplaceID string) (*domain.UserWorkplace, error)
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

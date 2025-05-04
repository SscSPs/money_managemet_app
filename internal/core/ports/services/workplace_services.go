package services

import (
	"context"

	"github.com/SscSPs/money_managemet_app/internal/core/domain"
)

// WorkplaceReaderSvc defines read operations for workplace data
type WorkplaceReaderSvc interface {
	// FindWorkplaceByID retrieves a specific workplace by its ID.
	FindWorkplaceByID(ctx context.Context, workplaceID string) (*domain.Workplace, error)

	// ListUserWorkplaces retrieves all workplaces a user belongs to.
	ListUserWorkplaces(ctx context.Context, userID string) ([]domain.Workplace, error)
}

// WorkplaceWriterSvc defines write operations for workplace data
type WorkplaceWriterSvc interface {
	// CreateWorkplace persists a new workplace.
	CreateWorkplace(ctx context.Context, name, description, defaultCurrencyCode, creatorUserID string) (*domain.Workplace, error)
}

// WorkplaceMembershipSvc defines operations for managing workplace membership
type WorkplaceMembershipSvc interface {
	// AddUserToWorkplace adds a user to a workplace with a specific role.
	AddUserToWorkplace(ctx context.Context, addingUserID, targetUserID, workplaceID string, role domain.UserWorkplaceRole) error
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

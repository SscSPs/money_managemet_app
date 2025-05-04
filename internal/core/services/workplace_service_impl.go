package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/SscSPs/money_managemet_app/internal/apperrors"
	"github.com/SscSPs/money_managemet_app/internal/core/domain"
	portsrepo "github.com/SscSPs/money_managemet_app/internal/core/ports/repositories"
	portssvc "github.com/SscSPs/money_managemet_app/internal/core/ports/services"
	"github.com/google/uuid"
)

// workplaceService implements the WorkplaceSvcFacade interface
type workplaceService struct {
	BaseService
	workplaceRepo portsrepo.WorkplaceRepositoryFacade
	currencyRepo  portsrepo.CurrencyReader
}

// NewWorkplaceService creates a new workplace service with the provided dependencies
func NewWorkplaceService(
	workplaceRepo portsrepo.WorkplaceRepositoryFacade,
	currencyRepo portsrepo.CurrencyReader,
) portssvc.WorkplaceSvcFacade {
	return &workplaceService{
		workplaceRepo: workplaceRepo,
		currencyRepo:  currencyRepo,
	}
}

// Ensure workplaceService implements the WorkplaceSvcFacade interface
var _ portssvc.WorkplaceSvcFacade = (*workplaceService)(nil)

// FindWorkplaceByID retrieves a workplace by its ID
func (s *workplaceService) FindWorkplaceByID(ctx context.Context, workplaceID string) (*domain.Workplace, error) {
	workplace, err := s.workplaceRepo.FindWorkplaceByID(ctx, workplaceID)
	if err != nil {
		if !errors.Is(err, apperrors.ErrNotFound) {
			s.LogError(ctx, err, "Failed to find workplace by ID",
				slog.String("workplace_id", workplaceID))
		}
		return nil, err
	}

	s.LogDebug(ctx, "Workplace retrieved successfully",
		slog.String("workplace_id", workplace.WorkplaceID))
	return workplace, nil
}

// ListUserWorkplaces retrieves all workplaces a user belongs to
func (s *workplaceService) ListUserWorkplaces(ctx context.Context, userID string) ([]domain.Workplace, error) {
	workplaces, err := s.workplaceRepo.ListWorkplacesByUserID(ctx, userID)
	if err != nil {
		s.LogError(ctx, err, "Failed to list workplaces for user",
			slog.String("user_id", userID))
		return nil, err
	}

	if workplaces == nil {
		return []domain.Workplace{}, nil
	}

	s.LogDebug(ctx, "Workplaces listed successfully",
		slog.Int("count", len(workplaces)),
		slog.String("user_id", userID))
	return workplaces, nil
}

// CreateWorkplace creates a new workplace
func (s *workplaceService) CreateWorkplace(ctx context.Context, name, description, defaultCurrencyCode, creatorUserID string) (*domain.Workplace, error) {
	// Validate currency if specified
	if defaultCurrencyCode != "" && s.currencyRepo != nil {
		_, err := s.currencyRepo.FindCurrencyByCode(ctx, defaultCurrencyCode)
		if err != nil {
			s.LogError(ctx, err, "Invalid default currency code",
				slog.String("currency_code", defaultCurrencyCode))
			return nil, fmt.Errorf("invalid default currency code: %w", err)
		}
	}

	now := time.Now()
	workplaceID := uuid.NewString()

	// Create domain.Workplace
	workplace := domain.Workplace{
		WorkplaceID: workplaceID,
		Name:        name,
		Description: description,
		AuditFields: domain.AuditFields{
			CreatedAt:     now,
			CreatedBy:     creatorUserID,
			LastUpdatedAt: now,
			LastUpdatedBy: creatorUserID,
		},
	}

	if defaultCurrencyCode != "" {
		workplace.DefaultCurrencyCode = &defaultCurrencyCode
	}

	err := s.workplaceRepo.SaveWorkplace(ctx, workplace)
	if err != nil {
		s.LogError(ctx, err, "Failed to save workplace",
			slog.String("workplace_id", workplace.WorkplaceID))
		return nil, err
	}

	// Add creator as an admin to the new workplace
	membershipErr := s.AddUserToWorkplace(ctx, creatorUserID, creatorUserID, workplaceID, domain.RoleAdmin)
	if membershipErr != nil {
		s.LogError(ctx, membershipErr, "Failed to add creator as admin to new workplace",
			slog.String("workplace_id", workplace.WorkplaceID),
			slog.String("user_id", creatorUserID))
		// Note: We don't return this error because the workplace was created successfully
		// In a real app, we might want to handle this more gracefully, perhaps with a transaction
	}

	s.LogInfo(ctx, "Workplace created successfully",
		slog.String("workplace_id", workplace.WorkplaceID),
		slog.String("creator_id", creatorUserID))
	return &workplace, nil
}

// AddUserToWorkplace adds a user to a workplace with a specific role
func (s *workplaceService) AddUserToWorkplace(ctx context.Context, addingUserID, targetUserID, workplaceID string, role domain.UserWorkplaceRole) error {
	// Check if adding user has permission (must be admin)
	if addingUserID != targetUserID { // Self-assignment is permitted (e.g., creator adding self as admin)
		err := s.AuthorizeUserAction(ctx, addingUserID, workplaceID, domain.RoleAdmin)
		if err != nil {
			s.LogError(ctx, err, "User not authorized to add members to workplace",
				slog.String("adding_user_id", addingUserID),
				slog.String("workplace_id", workplaceID))
			return err
		}
	}

	// Create membership
	membership := domain.UserWorkplace{
		UserID:      targetUserID,
		WorkplaceID: workplaceID,
		Role:        role,
		JoinedAt:    time.Now(),
	}

	err := s.workplaceRepo.AddUserToWorkplace(ctx, membership)
	if err != nil {
		s.LogError(ctx, err, "Failed to add user to workplace",
			slog.String("target_user_id", targetUserID),
			slog.String("workplace_id", workplaceID))
		return err
	}

	s.LogInfo(ctx, "User added to workplace successfully",
		slog.String("target_user_id", targetUserID),
		slog.String("workplace_id", workplaceID),
		slog.String("role", string(role)))
	return nil
}

// AuthorizeUserAction checks if a user has required permissions for a workplace
func (s *workplaceService) AuthorizeUserAction(ctx context.Context, userID, workplaceID string, requiredRole domain.UserWorkplaceRole) error {
	membership, err := s.workplaceRepo.FindUserWorkplaceRole(ctx, userID, workplaceID)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			s.LogDebug(ctx, "User not a member of workplace",
				slog.String("user_id", userID),
				slog.String("workplace_id", workplaceID))
			return apperrors.ErrForbidden
		}
		s.LogError(ctx, err, "Failed to find user workplace role",
			slog.String("user_id", userID),
			slog.String("workplace_id", workplaceID))
		return err
	}

	// Check if user has required role or higher
	if !hasRequiredRole(membership.Role, requiredRole) {
		s.LogDebug(ctx, "User does not have required role",
			slog.String("user_id", userID),
			slog.String("workplace_id", workplaceID),
			slog.String("user_role", string(membership.Role)),
			slog.String("required_role", string(requiredRole)))
		return apperrors.ErrForbidden
	}

	return nil
}

// hasRequiredRole checks if the user's role meets or exceeds the required role
func hasRequiredRole(userRole, requiredRole domain.UserWorkplaceRole) bool {
	// Simple role hierarchy check
	switch requiredRole {
	case domain.RoleMember: // Use only roles defined in the domain package
		return userRole == domain.RoleMember || userRole == domain.RoleAdmin
	case domain.RoleAdmin:
		return userRole == domain.RoleAdmin
	default:
		return false
	}
}

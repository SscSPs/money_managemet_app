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
// If includeDisabled is true, inactive workplaces are also included in the results.
// For inactive workplaces, only those where the user is an admin are included.
func (s *workplaceService) ListUserWorkplaces(ctx context.Context, userID string, includeDisabled bool) ([]domain.Workplace, error) {
	workplaces, err := s.workplaceRepo.ListWorkplacesByUserID(ctx, userID, includeDisabled, nil)
	if err != nil {
		s.LogError(ctx, err, "Failed to list workplaces for user",
			slog.String("user_id", userID),
			slog.Bool("include_disabled", includeDisabled))
		return nil, err
	}

	if workplaces == nil {
		return []domain.Workplace{}, nil
	}

	s.LogDebug(ctx, "Workplaces listed successfully",
		slog.Int("count", len(workplaces)),
		slog.String("user_id", userID),
		slog.Bool("include_disabled", includeDisabled))
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
		IsActive:    true, // New workplaces are active by default
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

// DeactivateWorkplace marks a workplace as inactive
func (s *workplaceService) DeactivateWorkplace(ctx context.Context, workplaceID string, requestingUserID string) error {
	// Verify user has admin rights in this workplace
	if err := s.AuthorizeUserAction(ctx, requestingUserID, workplaceID, domain.RoleAdmin); err != nil {
		return err // AuthorizeUserAction already logs the error
	}

	// Check if the workplace exists and get its current status
	workplace, err := s.workplaceRepo.FindWorkplaceByID(ctx, workplaceID)
	if err != nil {
		s.LogError(ctx, err, "Failed to find workplace for deactivation",
			slog.String("workplace_id", workplaceID))
		return err
	}

	// Check if workplace is already inactive
	if !workplace.IsActive {
		s.LogInfo(ctx, "Workplace already inactive",
			slog.String("workplace_id", workplaceID))
		return nil // No-op, already in desired state
	}

	// Update the workplace status to inactive
	if err := s.workplaceRepo.UpdateWorkplaceStatus(ctx, workplace, false, requestingUserID); err != nil {
		s.LogError(ctx, err, "Failed to deactivate workplace",
			slog.String("workplace_id", workplaceID),
			slog.String("requesting_user_id", requestingUserID))
		return fmt.Errorf("failed to deactivate workplace: %w", err)
	}

	s.LogInfo(ctx, "Workplace deactivated successfully",
		slog.String("workplace_id", workplaceID),
		slog.String("deactivated_by", requestingUserID))
	return nil
}

// ActivateWorkplace marks a workplace as active
func (s *workplaceService) ActivateWorkplace(ctx context.Context, workplaceID string, requestingUserID string) error {
	// Verify user has admin rights in this workplace
	if err := s.AuthorizeUserAction(ctx, requestingUserID, workplaceID, domain.RoleAdmin); err != nil {
		return err // AuthorizeUserAction already logs the error
	}

	// Check if the workplace exists and get its current status
	workplace, err := s.workplaceRepo.FindWorkplaceByID(ctx, workplaceID)
	if err != nil {
		s.LogError(ctx, err, "Failed to find workplace for activation",
			slog.String("workplace_id", workplaceID))
		return err
	}

	// Check if workplace is already active
	if workplace.IsActive {
		s.LogInfo(ctx, "Workplace already active",
			slog.String("workplace_id", workplaceID))
		return nil // No-op, already in desired state
	}

	// Update the workplace status to active
	if err := s.workplaceRepo.UpdateWorkplaceStatus(ctx, workplace, true, requestingUserID); err != nil {
		s.LogError(ctx, err, "Failed to activate workplace",
			slog.String("workplace_id", workplaceID),
			slog.String("requesting_user_id", requestingUserID))
		return fmt.Errorf("failed to activate workplace: %w", err)
	}

	s.LogInfo(ctx, "Workplace activated successfully",
		slog.String("workplace_id", workplaceID),
		slog.String("activated_by", requestingUserID))
	return nil
}

// hasRequiredRole checks if the user's role meets or exceeds the required role
func hasRequiredRole(userRole, requiredRole domain.UserWorkplaceRole) bool {
	// First check if the user has been removed
	if userRole == domain.RoleRemoved {
		return false // Removed users have no access
	}

	// Simple role hierarchy check
	switch requiredRole {
	case domain.RoleReadOnly: // ReadOnly is the lowest access level
		return userRole == domain.RoleReadOnly || userRole == domain.RoleMember || userRole == domain.RoleAdmin
	case domain.RoleMember:
		return userRole == domain.RoleMember || userRole == domain.RoleAdmin
	case domain.RoleAdmin:
		return userRole == domain.RoleAdmin
	default:
		return false
	}
}

// NewWorkplaceServiceLegacy creates a workplace service with legacy signature
// Provided for backward compatibility
func NewWorkplaceServiceLegacy(wr portsrepo.WorkplaceRepositoryFacade, cr portsrepo.CurrencyRepositoryFacade) portssvc.WorkplaceSvcFacade {
	return NewWorkplaceService(wr, cr)
}

// ListWorkplaceUsers retrieves all users and their roles for a specific workplace
func (s *workplaceService) ListWorkplaceUsers(ctx context.Context, workplaceID string, requestingUserID string) ([]domain.UserWorkplace, error) {
	// First, check if the requesting user is authorized to view the workplace's members
	// (ReadOnly role is sufficient for viewing workplace users)
	err := s.AuthorizeUserAction(ctx, requestingUserID, workplaceID, domain.RoleReadOnly)
	if err != nil {
		s.LogError(ctx, err, "User not authorized to list workplace users",
			slog.String("user_id", requestingUserID),
			slog.String("workplace_id", workplaceID))
		return nil, err
	}

	// Retrieve the list of users from the repository
	userWorkplaces, err := s.workplaceRepo.ListUsersByWorkplaceID(ctx, workplaceID)
	if err != nil {
		s.LogError(ctx, err, "Failed to list users for workplace",
			slog.String("workplace_id", workplaceID))
		return nil, err
	}

	s.LogDebug(ctx, "Listed users for workplace successfully",
		slog.String("workplace_id", workplaceID),
		slog.Int("user_count", len(userWorkplaces)))

	return userWorkplaces, nil
}

// RemoveUserFromWorkplace marks a user as removed in a workplace without deleting the record
// Only workplace admins can remove users from a workplace
func (s *workplaceService) RemoveUserFromWorkplace(ctx context.Context, requestingUserID, targetUserID, workplaceID string) error {
	// Simply call UpdateUserWorkplaceRole with the REMOVED role
	return s.UpdateUserWorkplaceRole(ctx, requestingUserID, targetUserID, workplaceID, domain.RoleRemoved)
}

// UpdateUserWorkplaceRole updates a user's role in a workplace
// Only admin users can update user roles
func (s *workplaceService) UpdateUserWorkplaceRole(ctx context.Context, requestingUserID, targetUserID, workplaceID string, newRole domain.UserWorkplaceRole) error {
	// Verify requesting user has admin rights
	if err := s.AuthorizeUserAction(ctx, requestingUserID, workplaceID, domain.RoleAdmin); err != nil {
		s.LogError(ctx, err, "User not authorized to update roles in workplace",
			slog.String("requesting_user_id", requestingUserID),
			slog.String("workplace_id", workplaceID))
		return err
	}

	// Check if the target user exists in the workplace
	currentMembership, err := s.workplaceRepo.FindUserWorkplaceRole(ctx, targetUserID, workplaceID)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			s.LogDebug(ctx, "Target user not found in workplace",
				slog.String("target_user_id", targetUserID),
				slog.String("workplace_id", workplaceID))
			return apperrors.ErrNotFound
		}
		s.LogError(ctx, err, "Failed to check target user workplace role",
			slog.String("target_user_id", targetUserID),
			slog.String("workplace_id", workplaceID))
		return err
	}

	// If user is already removed and the new role is also REMOVED, return early
	if currentMembership.Role == domain.RoleRemoved && newRole == domain.RoleRemoved {
		s.LogDebug(ctx, "User is already marked as removed",
			slog.String("target_user_id", targetUserID),
			slog.String("workplace_id", workplaceID))
		return nil
	}

	// If attempting to downgrade an admin, ensure there's at least one other admin
	if currentMembership.Role == domain.RoleAdmin && newRole != domain.RoleAdmin {
		// Check if there are other admins
		admins, err := s.countAdminsInWorkplace(ctx, workplaceID)
		if err != nil {
			s.LogError(ctx, err, "Failed to count admins in workplace",
				slog.String("workplace_id", workplaceID))
			return err
		}

		if admins <= 1 {
			s.LogDebug(ctx, "Cannot demote the last admin in workplace",
				slog.String("target_user_id", targetUserID),
				slog.String("workplace_id", workplaceID))
			return fmt.Errorf("%w: cannot demote the last admin in workplace", apperrors.ErrValidation)
		}
	}

	// If role hasn't changed, return early
	if currentMembership.Role == newRole {
		s.LogDebug(ctx, "User already has the requested role",
			slog.String("target_user_id", targetUserID),
			slog.String("workplace_id", workplaceID),
			slog.String("role", string(newRole)))
		return nil
	}

	// Update the user's role
	if err := s.workplaceRepo.UpdateUserWorkplaceRole(ctx, targetUserID, workplaceID, newRole); err != nil {
		s.LogError(ctx, err, "Failed to update user's role in workplace",
			slog.String("target_user_id", targetUserID),
			slog.String("workplace_id", workplaceID))
		return err
	}

	// Log appropriate message based on the new role
	if newRole == domain.RoleRemoved {
		s.LogInfo(ctx, "User marked as removed from workplace",
			slog.String("target_user_id", targetUserID),
			slog.String("workplace_id", workplaceID),
			slog.String("updated_by", requestingUserID))
	} else if currentMembership.Role == domain.RoleRemoved {
		s.LogInfo(ctx, "User reinstated to workplace with new role",
			slog.String("target_user_id", targetUserID),
			slog.String("workplace_id", workplaceID),
			slog.String("new_role", string(newRole)),
			slog.String("updated_by", requestingUserID))
	} else {
		s.LogInfo(ctx, "User role updated successfully",
			slog.String("target_user_id", targetUserID),
			slog.String("workplace_id", workplaceID),
			slog.String("new_role", string(newRole)),
			slog.String("updated_by", requestingUserID))
	}

	return nil
}

// countAdminsInWorkplace counts the number of admin users in a workplace
func (s *workplaceService) countAdminsInWorkplace(ctx context.Context, workplaceID string) (int, error) {
	// Get all users in the workplace (including those marked as REMOVED, as we need to check all users)
	users, err := s.workplaceRepo.ListUsersByWorkplaceID(ctx, workplaceID, true)
	if err != nil {
		return 0, err
	}

	// Count admin users (don't count REMOVED users as admins even if they were admins)
	count := 0
	for _, user := range users {
		if user.Role == domain.RoleAdmin {
			count++
		}
	}

	return count, nil
}

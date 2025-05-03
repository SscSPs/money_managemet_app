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
	"github.com/SscSPs/money_managemet_app/internal/middleware"
	"github.com/google/uuid"
)

// WorkplaceService handles business logic related to workplaces and memberships.
type WorkplaceService struct {
	workplaceRepo portsrepo.WorkplaceRepository
	currencyRepo  portsrepo.CurrencyRepository
	// userRepo portsrepo.UserRepository // Might be needed for user validation
}

// NewWorkplaceService creates a new WorkplaceService.
func NewWorkplaceService(wr portsrepo.WorkplaceRepository, cr portsrepo.CurrencyRepository) portssvc.WorkplaceService {
	return &WorkplaceService{
		workplaceRepo: wr,
		currencyRepo:  cr,
	}
}

// Ensure WorkplaceService implements the portssvc.WorkplaceService interface
var _ portssvc.WorkplaceService = (*WorkplaceService)(nil)

// CreateWorkplace creates a new workplace and makes the creator the initial admin.
func (s *WorkplaceService) CreateWorkplace(ctx context.Context, name, description, defaultCurrencyCode, creatorUserID string) (*domain.Workplace, error) {
	logger := middleware.GetLoggerFromCtx(ctx)

	// Validate that the currency exists
	if defaultCurrencyCode != "" {
		_, err := s.currencyRepo.FindCurrencyByCode(ctx, defaultCurrencyCode)
		if err != nil {
			if errors.Is(err, apperrors.ErrNotFound) {
				logger.Warn("Invalid default currency code provided", slog.String("currency_code", defaultCurrencyCode))
				return nil, fmt.Errorf("%w: currency code %s not found", apperrors.ErrValidation, defaultCurrencyCode)
			}
			logger.Error("Failed to check currency code existence", slog.String("error", err.Error()), slog.String("currency_code", defaultCurrencyCode))
			return nil, fmt.Errorf("failed to validate currency code: %w", err)
		}
	}

	now := time.Now()
	newWorkplaceID := uuid.NewString()

	workplace := domain.Workplace{
		WorkplaceID: newWorkplaceID,
		Name:        name,
		Description: description,
		AuditFields: domain.AuditFields{
			CreatedAt:     now,
			CreatedBy:     creatorUserID,
			LastUpdatedAt: now,
			LastUpdatedBy: creatorUserID,
		},
	}

	// Only set default currency if provided (otherwise will be NULL in DB)
	if defaultCurrencyCode != "" {
		workplace.DefaultCurrencyCode = &defaultCurrencyCode
	}

	// Save the workplace
	if err := s.workplaceRepo.SaveWorkplace(ctx, workplace); err != nil {
		logger.Error("Failed to save workplace in repository", slog.String("error", err.Error()), slog.String("workplace_name", name))
		return nil, fmt.Errorf("failed to create workplace: %w", err)
	}

	// Add the creator as the initial admin
	membership := domain.UserWorkplace{
		UserID:      creatorUserID,
		WorkplaceID: newWorkplaceID,
		Role:        domain.RoleAdmin, // Creator is Admin
		JoinedAt:    now,
	}
	if err := s.workplaceRepo.AddUserToWorkplace(ctx, membership); err != nil {
		// Log the error, but maybe don't fail the whole operation?
		// Or perhaps implement transactional behavior in the repo.
		logger.Error("Failed to add creator as admin to new workplace", slog.String("error", err.Error()), slog.String("workplace_id", newWorkplaceID), slog.String("user_id", creatorUserID))
		// Decide on error handling: Return partial success? Return error?
		// For now, return the workplace but log the membership error.
	}

	logger.Info("Workplace created successfully", slog.String("workplace_id", newWorkplaceID), slog.String("creator_user_id", creatorUserID))
	return &workplace, nil
}

// AddUserToWorkplace adds a user to a workplace with a specific role.
func (s *WorkplaceService) AddUserToWorkplace(ctx context.Context, addingUserID, targetUserID, workplaceID string, role domain.UserWorkplaceRole) error {
	logger := middleware.GetLoggerFromCtx(ctx)

	// 1. Authorization: Check if addingUserID has permission (e.g., is ADMIN) in workplaceID
	if err := s.AuthorizeUserAction(ctx, addingUserID, workplaceID, domain.RoleAdmin); err != nil {
		return err // Return auth error (NotFound or Forbidden)
	}

	// TODO: Validate targetUserID exists using UserRepository if added
	// TODO: Validate workplaceID exists? (Repo AddUserToWorkplace might handle FK violation)

	// 2. Add the membership
	now := time.Now()
	membership := domain.UserWorkplace{
		UserID:      targetUserID,
		WorkplaceID: workplaceID,
		Role:        role,
		JoinedAt:    now,
	}

	if err := s.workplaceRepo.AddUserToWorkplace(ctx, membership); err != nil {
		logger.Error("Failed to add user to workplace in repository", slog.String("error", err.Error()), slog.String("target_user_id", targetUserID), slog.String("workplace_id", workplaceID))
		return fmt.Errorf("failed to add user %s to workplace %s: %w", targetUserID, workplaceID, err)
	}

	logger.Info("User added to workplace successfully", slog.String("target_user_id", targetUserID), slog.String("workplace_id", workplaceID), slog.String("role", string(role)), slog.String("added_by_user_id", addingUserID))
	return nil
}

// ListUserWorkplaces retrieves the list of workplaces a given user belongs to.
func (s *WorkplaceService) ListUserWorkplaces(ctx context.Context, userID string) ([]domain.Workplace, error) {
	logger := middleware.GetLoggerFromCtx(ctx)

	workplaces, err := s.workplaceRepo.ListWorkplacesByUserID(ctx, userID)
	if err != nil {
		logger.Error("Failed to list workplaces for user from repository", slog.String("error", err.Error()), slog.String("user_id", userID))
		return nil, fmt.Errorf("failed to list workplaces for user %s: %w", userID, err)
	}

	if workplaces == nil {
		return []domain.Workplace{}, nil // Return empty slice, not nil
	}

	logger.Debug("Workplaces listed successfully for user", slog.String("user_id", userID), slog.Int("count", len(workplaces)))
	return workplaces, nil
}

// FindWorkplaceByID retrieves a workplace by its ID.
// Implements portssvc.WorkplaceService interface.
func (s *WorkplaceService) FindWorkplaceByID(ctx context.Context, workplaceID string) (*domain.Workplace, error) {
	logger := middleware.GetLoggerFromCtx(ctx)
	// No specific authorization check here, assuming Get needs are handled by caller or subsequent checks.
	workplace, err := s.workplaceRepo.FindWorkplaceByID(ctx, workplaceID)
	if err != nil {
		if !errors.Is(err, apperrors.ErrNotFound) {
			logger.Error("Failed to find workplace by ID in repository", slog.String("error", err.Error()), slog.String("workplace_id", workplaceID))
		}
		return nil, err // Propagate error (including NotFound)
	}
	logger.Debug("Workplace found by ID", slog.String("workplace_id", workplaceID))
	return workplace, nil
}

// AuthorizeUserAction checks if a user has the required role (or higher) within a specific workplace.
// Returns apperrors.ErrNotFound if user/workplace doesn't exist or user not member.
// Returns apperrors.ErrForbidden if user is member but lacks the required role.
// Returns nil if authorized.
func (s *WorkplaceService) AuthorizeUserAction(ctx context.Context, userID, workplaceID string, requiredRole domain.UserWorkplaceRole) error {
	logger := middleware.GetLoggerFromCtx(ctx)

	membership, err := s.workplaceRepo.FindUserWorkplaceRole(ctx, userID, workplaceID)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			logger.Warn("Authorization failed: User or workplace not found, or user not a member", slog.String("user_id", userID), slog.String("workplace_id", workplaceID))
			// Return NotFound to avoid revealing workplace existence if user shouldn't know
			return apperrors.ErrNotFound // Or return a more specific AuthError
		}
		// Log unexpected repo error
		logger.Error("Failed to check user workplace role in repository", slog.String("error", err.Error()), slog.String("user_id", userID), slog.String("workplace_id", workplaceID))
		return fmt.Errorf("failed to check authorization: %w", err)
	}

	// Basic role check (ADMIN has all permissions)
	if membership.Role == domain.RoleAdmin {
		return nil // Admin is always authorized
	}

	// Check if the user's role meets the required role
	if membership.Role == requiredRole {
		return nil // User has the exact required role
	}

	// TODO: Implement more granular role hierarchy if needed (e.g., if EDITOR > MEMBER)

	logger.Warn("Authorization failed: User lacks required role", slog.String("user_id", userID), slog.String("workplace_id", workplaceID), slog.String("user_role", string(membership.Role)), slog.String("required_role", string(requiredRole)))
	return apperrors.ErrForbidden
}

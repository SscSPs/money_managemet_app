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
	"github.com/SscSPs/money_managemet_app/internal/dto"
	"github.com/SscSPs/money_managemet_app/internal/middleware"
	"github.com/google/uuid"
)

// userService provides business logic for user operations.
type userService struct {
	userRepo portsrepo.UserRepositoryFacade
}

// NewUserService creates a new UserService.
func NewUserService(repo portsrepo.UserRepositoryFacade) portssvc.UserSvcFacade {
	return &userService{
		userRepo: repo,
	}
}

func (s *userService) CreateUser(ctx context.Context, req dto.CreateUserRequest) (*domain.User, error) {
	logger := middleware.GetLoggerFromCtx(ctx)
	now := time.Now()
	newUserID := uuid.NewString()
	// Determine creatorUserID - for self-registration via /auth/register, it might be the user ID itself or a system ID.
	// For creation via /users endpoint, it should come from the authenticated admin user context.
	// For now, using a placeholder. This needs proper handling based on the call site.
	creatorUserID := "PLACEHOLDER_CREATOR_ID" // TODO: Determine creator properly

	user := domain.User{
		UserID: newUserID,
		Name:   req.Name,
		AuditFields: domain.AuditFields{
			CreatedAt:     now,
			CreatedBy:     creatorUserID,
			LastUpdatedAt: now,
			LastUpdatedBy: creatorUserID,
		},
	}

	err := s.userRepo.SaveUser(ctx, user)
	if err != nil {
		logger.Error("Failed to save user in repository", slog.String("error", err.Error()), slog.String("user_name", req.Name))
		return nil, fmt.Errorf("failed to create user in service: %w", err)
	}

	logger.Info("User created successfully in service", slog.String("user_id", user.UserID))
	return &user, nil
}

func (s *userService) GetUserByID(ctx context.Context, userID string) (*domain.User, error) {
	logger := middleware.GetLoggerFromCtx(ctx)
	user, err := s.userRepo.FindUserByID(ctx, userID)
	if err != nil {
		if !errors.Is(err, apperrors.ErrNotFound) {
			logger.Error("Failed to find user by ID in repository", slog.String("error", err.Error()), slog.String("user_id", userID))
		}
		return nil, err
	}
	logger.Debug("User retrieved successfully by ID from service", slog.String("user_id", user.UserID))
	return user, nil
}

// ListUsers retrieves a paginated list of non-deleted users.
// Implements portssvc.UserService interface.
func (s *userService) ListUsers(ctx context.Context, limit, offset int) ([]domain.User, error) {
	logger := middleware.GetLoggerFromCtx(ctx)
	// Apply defaults if needed (or rely on repo layer defaults)
	if limit <= 0 {
		limit = 20 // Default limit
	}
	if offset < 0 {
		offset = 0 // Default offset
	}

	users, err := s.userRepo.FindUsers(ctx, limit, offset)
	if err != nil {
		logger.Error("Failed to find users in repository", slog.String("error", err.Error()), slog.Int("limit", limit), slog.Int("offset", offset))
		return nil, fmt.Errorf("failed to list users: %w", err)
	}

	logger.Debug("Users listed successfully from service", slog.Int("count", len(users)), slog.Int("limit", limit), slog.Int("offset", offset))
	// Return the domain slice directly as required by the interface
	if users == nil {
		return []domain.User{}, nil // Ensure non-nil slice is returned
	}
	return users, nil
}

func (s *userService) UpdateUser(ctx context.Context, userID string, req dto.UpdateUserRequest, updaterUserID string) (*domain.User, error) {
	logger := middleware.GetLoggerFromCtx(ctx)
	existingUser, err := s.userRepo.FindUserByID(ctx, userID)
	if err != nil {
		if !errors.Is(err, apperrors.ErrNotFound) {
			logger.Error("Failed to find user by ID for update", slog.String("error", err.Error()), slog.String("user_id", userID))
		}
		return nil, err
	}

	updateOccurred := false
	if req.Name != nil && *req.Name != existingUser.Name {
		existingUser.Name = *req.Name
		updateOccurred = true
	}

	if !updateOccurred {
		logger.Debug("No update needed for user", slog.String("user_id", userID))
		return existingUser, nil
	}

	existingUser.LastUpdatedAt = time.Now()
	existingUser.LastUpdatedBy = updaterUserID

	err = s.userRepo.UpdateUser(ctx, *existingUser)
	if err != nil {
		logger.Error("Failed to update user in repository", slog.String("error", err.Error()), slog.String("user_id", userID))
		if errors.Is(err, apperrors.ErrNotFound) {
			logger.Warn("Update failed because user was not found (possibly deleted concurrently)", slog.String("user_id", userID))
			return nil, apperrors.ErrNotFound
		}
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	logger.Info("User updated successfully in service", slog.String("user_id", userID))
	return existingUser, nil
}

func (s *userService) DeleteUser(ctx context.Context, userID string, deleterUserID string) error {
	logger := middleware.GetLoggerFromCtx(ctx)
	now := time.Now()
	err := s.userRepo.MarkUserDeleted(ctx, userID, now, deleterUserID)
	if err != nil {
		if !errors.Is(err, apperrors.ErrNotFound) {
			logger.Error("Failed to mark user deleted in repository", slog.String("error", err.Error()), slog.String("user_id", userID))
		}
		return err
	}
	logger.Info("User marked as deleted successfully in service", slog.String("user_id", userID))
	return nil
}

// AuthenticateUser checks user credentials.
// TODO: Implement actual authentication logic (e.g., check password hash)
func (s *userService) AuthenticateUser(ctx context.Context, email, password string) (*domain.User, error) {
	logger := middleware.GetLoggerFromCtx(ctx)
	logger.Warn("AuthenticateUser not implemented", slog.String("email", email))
	// Placeholder: Find user by email (assuming email is unique identifier for login)
	// user, err := s.userRepo.FindUserByEmail(ctx, email) // Hypothetical repo method
	// if err != nil {
	// 	 return nil, err // Propagate NotFound or other errors
	// }
	// // Check password hash here
	// return user, nil

	// --- TEMPORARY: Return error indicating not implemented --- \
	return nil, fmt.Errorf("authentication not implemented")
	// --- /TEMPORARY ---
}

// TODO: Add other user management methods (Update, Delete/Deactivate, List) later

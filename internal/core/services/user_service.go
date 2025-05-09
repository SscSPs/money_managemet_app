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
	"github.com/SscSPs/money_managemet_app/internal/utils"
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

func (s *userService) GetUserByUsername(ctx context.Context, username string) (*domain.User, error) {
	return s.userRepo.FindUserByUsername(ctx, username)
}

func (s *userService) CreateUser(ctx context.Context, req dto.CreateUserRequest) (*domain.User, error) {
	logger := middleware.GetLoggerFromCtx(ctx)
	now := time.Now()
	newUserID := uuid.NewString()
	// Determine creatorUserID - for self-registration via /auth/register, it might be the user ID itself or a system ID.
	// For creation via /users endpoint, it should come from the authenticated admin user context.
	// For now, using a placeholder. This needs proper handling based on the call site.
	creatorUserID := "PLACEHOLDER_CREATOR_ID" // TODO: Determine creator properly

	// Hash password
	hash, err := utils.HashPassword(req.Password)
	if err != nil {
		logger.Error("Failed to hash password", slog.String("error", err.Error()))
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	user := domain.User{
		UserID:       newUserID,
		Name:         req.Name,
		Username:     req.Username,
		PasswordHash: hash,
		AuditFields: domain.AuditFields{
			CreatedAt:     now,
			CreatedBy:     creatorUserID,
			LastUpdatedAt: now,
			LastUpdatedBy: creatorUserID,
		},
	}

	err = s.userRepo.SaveUser(ctx, user)
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
func (s *userService) AuthenticateUser(ctx context.Context, username, password string) (*domain.User, error) {
	logger := middleware.GetLoggerFromCtx(ctx)
	user, err := s.userRepo.FindUserByUsername(ctx, username)
	if err != nil {
		logger.Warn("User not found for authentication", slog.String("username", username))
		return nil, apperrors.ErrNotFound
	}
	if !utils.CheckPasswordHash(password, user.PasswordHash) {
		logger.Warn("Invalid password for user", slog.String("username", username))
		return nil, apperrors.ErrUnauthorized
	}
	return user, nil
}

// UpdateRefreshToken stores the hashed refresh token and its expiry for a user.
func (s *userService) UpdateRefreshToken(ctx context.Context, userID string, refreshTokenHash string, refreshTokenExpiryTime time.Time) error {
	logger := middleware.GetLoggerFromCtx(ctx)
	err := s.userRepo.UpdateRefreshToken(ctx, userID, refreshTokenHash, refreshTokenExpiryTime)
	if err != nil {
		logger.Error("Failed to update refresh token in repository", slog.String("error", err.Error()), slog.String("user_id", userID))
		// Check if it's a not found error from the repo and return it directly if so
		if errors.Is(err, apperrors.ErrNotFound) {
			return apperrors.ErrNotFound // Or a more specific error like "user not found for refresh token update"
		}
		return fmt.Errorf("failed to update refresh token for user %s: %w", userID, err)
	}
	logger.Info("Refresh token updated successfully for user", slog.String("user_id", userID))
	return nil
}

// ClearRefreshToken clears the refresh token for a user.
func (s *userService) ClearRefreshToken(ctx context.Context, userID string) error {
	logger := middleware.GetLoggerFromCtx(ctx)
	// Input validation (optional, but good practice)
	if userID == "" {
		return fmt.Errorf("%w: userID cannot be empty", apperrors.ErrBadRequest)
	}

	// Call repository to clear the token
	err := s.userRepo.ClearRefreshToken(ctx, userID)
	if err != nil {
		// Log the error for observability
		logger.ErrorContext(ctx, "Failed to clear refresh token in repository", slog.String("user_id", userID), slog.String("error", err.Error()))
		// It's possible the user doesn't exist or the token was already cleared.
		// Depending on how strict we want to be, we might return the error or handle it.
		// For logout, if the user isn't found, it's not critical, so we can absorb some errors.
		// However, if the DB operation itself fails, we should return an error.
		if !errors.Is(err, apperrors.ErrNotFound) { // Assuming ClearRefreshToken in repo might return ErrNotFound
			return fmt.Errorf("%w: failed to clear refresh token: %v", apperrors.ErrInternal, err)
		}
		// If it's ErrNotFound, we can consider logout successful from the service perspective.
	}

	logger.InfoContext(ctx, "Refresh token cleared successfully", slog.String("user_id", userID))
	return nil
}

// TODO: Add other user management methods (Update, Delete/Deactivate, List) later

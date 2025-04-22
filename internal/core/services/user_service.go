package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/SscSPs/money_managemet_app/internal/apperrors"
	"github.com/SscSPs/money_managemet_app/internal/core/ports"
	"github.com/SscSPs/money_managemet_app/internal/dto"
	"github.com/SscSPs/money_managemet_app/internal/middleware"
	"github.com/SscSPs/money_managemet_app/internal/models"
	"github.com/google/uuid"
)

type UserService struct {
	userRepo ports.UserRepository
}

func NewUserService(userRepo ports.UserRepository) *UserService {
	return &UserService{userRepo: userRepo}
}

func (s *UserService) CreateUser(ctx context.Context, req dto.CreateUserRequest, creatorUserID string) (*models.User, error) {
	logger := middleware.GetLoggerFromCtx(ctx)
	now := time.Now()
	newUserID := uuid.NewString()

	user := models.User{
		UserID: newUserID,
		Name:   req.Name,
		AuditFields: models.AuditFields{
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

func (s *UserService) GetUserByID(ctx context.Context, userID string) (*models.User, error) {
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
func (s *UserService) ListUsers(ctx context.Context, req dto.ListUsersParams) (*dto.ListUsersResponse, error) {
	logger := middleware.GetLoggerFromCtx(ctx)
	limit := 10
	if req.Limit > 0 {
		limit = req.Limit
	}
	offset := 0
	if req.Offset > 0 {
		offset = req.Offset
	}

	users, err := s.userRepo.FindUsers(ctx, limit, offset)
	if err != nil {
		logger.Error("Failed to find users in repository", slog.String("error", err.Error()), slog.Int("limit", limit), slog.Int("offset", offset))
		return nil, fmt.Errorf("failed to list users: %w", err)
	}

	logger.Debug("Users listed successfully from service", slog.Int("count", len(users)), slog.Int("limit", limit), slog.Int("offset", offset))
	response := dto.ToListUserResponse(users)
	return &response, nil
}

func (s *UserService) UpdateUser(ctx context.Context, userID string, req dto.UpdateUserRequest, updaterUserID string) (*models.User, error) {
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

func (s *UserService) DeleteUser(ctx context.Context, userID string, deleterUserID string) error {
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

// TODO: Add other user management methods (Update, Delete/Deactivate, List) later

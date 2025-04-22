package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/SscSPs/money_managemet_app/internal/apperrors"
	"github.com/SscSPs/money_managemet_app/internal/core/ports"
	"github.com/SscSPs/money_managemet_app/internal/dto"
	"github.com/SscSPs/money_managemet_app/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type UserService struct {
	userRepo ports.UserRepository
}

func NewUserService(userRepo ports.UserRepository) *UserService {
	return &UserService{userRepo: userRepo}
}

func (s *UserService) CreateUser(ctx context.Context, req dto.CreateUserRequest, creatorUserID string) (*models.User, error) {
	now := time.Now()
	newUserID := uuid.NewString() // Generate a new UUID for the user

	user := models.User{
		UserID: newUserID,
		Name:   req.Name,
		AuditFields: models.AuditFields{
			CreatedAt:     now,
			CreatedBy:     creatorUserID, // The user performing the creation
			LastUpdatedAt: now,
			LastUpdatedBy: creatorUserID, // Initially same as creator
		},
	}

	err := s.userRepo.SaveUser(ctx, user)
	if err != nil {
		// TODO: Add structured logging
		return nil, fmt.Errorf("failed to create user in service: %w", err)
	}

	return &user, nil
}

func (s *UserService) GetUserByID(ctx context.Context, userID string) (*models.User, error) {
	user, err := s.userRepo.FindUserByID(ctx, userID)
	if err != nil {
		// TODO: Add structured logging
		// TODO: Consider if the repo should return a specific error for not found too
		// If the repo itself returns a specific ErrNotFound, we might just return that directly.
		// For now, assume repo returns (nil, nil) or (nil, otherError)
		if user == nil && err == nil { // Explicitly check if repo returned nil, nil (meaning not found by convention)
			return nil, apperrors.ErrNotFound
		}
		// Handle other potential errors from the repository
		return nil, fmt.Errorf("failed to get user by ID from repository: %w", err)
	}
	// Original check for nil user when no error occurred.
	// This handles the case where the repo explicitly returns (nil, nil).
	if user == nil {
		return nil, apperrors.ErrNotFound // Return the specific not found error
	}
	return user, nil
}

// ListUsers retrieves a paginated list of non-deleted users.
func (s *UserService) ListUsers(ctx context.Context, limit int, offset int) ([]models.User, error) {
	users, err := s.userRepo.FindUsers(ctx, limit, offset)
	if err != nil {
		// TODO: Add structured logging
		return nil, fmt.Errorf("failed to list users from repository: %w", err)
	}
	return users, nil
}

// UpdateUser updates an existing user's details.
// It only allows updating certain fields (e.g., Name).
func (s *UserService) UpdateUser(ctx context.Context, userID string, req dto.UpdateUserRequest, updaterUserID string) (*models.User, error) {
	// 1. Fetch the existing user to ensure it exists and is not deleted
	//    We fetch it anyway to return the updated object.
	existingUser, err := s.userRepo.FindUserByID(ctx, userID)
	if err != nil {
		// Check if the find error is pgx.ErrNoRows, map to apperrors.ErrNotFound
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.ErrNotFound
		}
		// TODO: Add structured logging
		return nil, fmt.Errorf("failed to find user for update: %w", err)
	}
	if existingUser == nil { // Should be caught by the error check above, but belt-and-suspenders
		return nil, apperrors.ErrNotFound
	}
	// Check if user is already deleted (FindUserByID should ideally filter this, but double check)
	if existingUser.DeletedAt != nil {
		return nil, apperrors.ErrNotFound // Treat deleted users as not found for updates
	}

	// 2. Update fields (only Name in this case)
	now := time.Now()
	updateNeeded := false
	if req.Name != nil && *req.Name != existingUser.Name {
		existingUser.Name = *req.Name
		updateNeeded = true
	}

	// If no fields were actually changed, we can optionally return early.
	if !updateNeeded {
		return existingUser, nil // Return the unchanged user
	}

	// 3. Set audit fields
	existingUser.LastUpdatedAt = now
	existingUser.LastUpdatedBy = updaterUserID

	// 4. Save updated user
	err = s.userRepo.UpdateUser(ctx, *existingUser)
	if err != nil {
		// Check if the update error is pgx.ErrNoRows (e.g., race condition where user was deleted)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.ErrNotFound
		}
		// TODO: Add structured logging
		return nil, fmt.Errorf("failed to update user in repository: %w", err)
	}

	return existingUser, nil
}

// DeleteUser performs a soft delete on a user.
func (s *UserService) DeleteUser(ctx context.Context, userID string, deleterUserID string) error {
	// Optionally, check if user exists and is not deleted first using FindUserByID.
	// This prevents an unnecessary update call but adds a read query.
	// For simplicity here, we rely on the MarkUserDeleted repo method to handle non-existent/already-deleted cases.

	now := time.Now()
	err := s.userRepo.MarkUserDeleted(ctx, userID, now, deleterUserID)
	if err != nil {
		// Check if the error is pgx.ErrNoRows, map to apperrors.ErrNotFound
		if errors.Is(err, pgx.ErrNoRows) {
			return apperrors.ErrNotFound
		}
		// TODO: Add structured logging
		return fmt.Errorf("failed to mark user deleted in repository: %w", err)
	}

	return nil
}

// TODO: Add other user management methods (Update, Delete/Deactivate, List) later

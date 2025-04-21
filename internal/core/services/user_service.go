package services

import (
	"context"
	"fmt"
	"time"

	"github.com/SscSPs/money_managemet_app/internal/core/ports"
	"github.com/SscSPs/money_managemet_app/internal/dto"
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
		return nil, fmt.Errorf("failed to get user by ID in service: %w", err)
	}
	if user == nil {
		// Service layer could return a specific "not found" error type here
		return nil, nil // Or return a custom error
	}
	return user, nil
}

// TODO: Add other user management methods (Update, Delete/Deactivate, List) later

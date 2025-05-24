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

var _ portssvc.UserSvcFacade = (*userService)(nil)

type userService struct {
	userRepo portsrepo.UserRepositoryWithTx
}

// NewUserService creates a new user service.
func NewUserService(repo portsrepo.UserRepositoryWithTx) *userService {
	return &userService{
		userRepo: repo,
	}
}

func (s *userService) GetUserByUsername(ctx context.Context, username string) (*domain.User, error) {
	logger := middleware.GetLoggerFromCtx(ctx)
	logger.Debug("Attempting to get user by username", "username", username)
	foundUser, err := s.userRepo.FindUserByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			logger.Debug("User not found by username", "username", username)
		} else {
			logger.Error("Error finding user by username from repository", "username", username, "error", err)
		}
		return nil, err
	}
	logger.Debug("Successfully found user by username", "username", username, "user_id", foundUser.UserID)
	return foundUser, nil
}

// CreateUser creates a new user.
// It hashes the password before saving it.
// If the username is not provided, it's generated from the email.
// It checks if the username or email already exists.
func (s *userService) CreateUser(ctx context.Context, req dto.CreateUserRequest) (*domain.User, error) {
	logger := middleware.GetLoggerFromCtx(ctx)
	now := time.Now()
	newUserID := uuid.NewString()

	// Validate password (DTO binding should ensure it's present)
	if req.Password == "" {
		logger.Warn("CreateUser called with empty password despite DTO validation", "username", req.Username)
		return nil, fmt.Errorf("%w: password is required", apperrors.ErrValidation)
	}

	// Username generation logic (e.g., from email) is not applicable here
	// as CreateUserRequest does not contain Email.
	// If Username is empty, DTO validation `binding:"required"` for Username should catch it.
	if req.Username == "" {
		logger.Warn("CreateUser called with empty username despite DTO validation", "username_attempt", req.Username)
		return nil, fmt.Errorf("%w: username is required", apperrors.ErrValidation)
	}

	// Check if username already exists
	logger.Debug("Checking if username exists", "username", req.Username)
	existingUserByUsername, err := s.userRepo.FindUserByUsername(ctx, req.Username)
	if err != nil && !errors.Is(err, apperrors.ErrNotFound) {
		logger.Error("Error checking if username exists from repository", "username", req.Username, "error", err)
		return nil, fmt.Errorf("error checking username: %w", err)
	}
	if existingUserByUsername != nil {
		logger.Warn("Username already exists", "username", req.Username)
		return nil, fmt.Errorf("username '%s' already exists: %w", req.Username, apperrors.ErrDuplicate)
	}

	// Hash password
	hash, err := utils.HashPassword(req.Password)
	if err != nil {
		logger.Error("Failed to hash password for new user", "username", req.Username, "error", err)
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	user := &domain.User{
		UserID:       newUserID,
		Name:         req.Name,
		Username:     req.Username,
		PasswordHash: &hash,
		AuthProvider: domain.ProviderLocal,
		IsVerified:   false,
		AuditFields: domain.AuditFields{
			CreatedAt:     now,
			LastUpdatedAt: now,
			CreatedBy:     newUserID,
			LastUpdatedBy: newUserID,
		},
	}

	// Save the new user to the repository
	logger.Info("Attempting to save new user", "username", user.Username, "user_id", user.UserID)
	err = s.userRepo.SaveUser(ctx, user)
	if err != nil {
		logger.Error("Failed to save new user to repository", "username", user.Username, "error", err)
		return nil, fmt.Errorf("failed to save user: %w", err)
	}

	logger.Info("User created successfully", "username", user.Username, "user_id", user.UserID)
	user.PasswordHash = nil // Clear password hash before returning
	return user, nil
}

func (s *userService) GetUserByID(ctx context.Context, userID string) (*domain.User, error) {
	logger := middleware.GetLoggerFromCtx(ctx)
	logger.Debug("Attempting to get user by ID", "user_id", userID)
	user, err := s.userRepo.FindUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			logger.Debug("User not found by ID", "user_id", userID)
		} else {
			logger.Error("Error finding user by ID from repository", "user_id", userID, "error", err)
		}
		return nil, err
	}
	logger.Debug("Successfully found user by ID", "user_id", userID)
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

	logger.Debug("Attempting to list users", "limit", limit, "offset", offset)
	users, err := s.userRepo.FindUsers(ctx, limit, offset)
	if err != nil {
		logger.Error("Error listing users from repository", "limit", limit, "offset", offset, "error", err)
		return nil, err
	}
	logger.Debug("Successfully listed users", "count", len(users), "limit", limit, "offset", offset)
	// Return the domain slice directly as required by the interface
	if users == nil {
		return []domain.User{}, nil // Ensure non-nil slice is returned
	}
	return users, nil
}

func (s *userService) UpdateUser(ctx context.Context, userID string, req dto.UpdateUserRequest, updaterUserID string) (*domain.User, error) {
	logger := middleware.GetLoggerFromCtx(ctx)
	logger.Info("Attempting to update user", "user_id_to_update", userID, "updater_user_id", updaterUserID, "request_name", req.Name)
	existingUser, err := s.userRepo.FindUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			logger.Warn("User to update not found", "user_id", userID)
		} else {
			logger.Error("Failed to find user for update from repository", "user_id", userID, "error", err)
		}
		return nil, fmt.Errorf("user with ID %s not found for update: %w", userID, err) // Keep specific error for handler
	}

	updateOccurred := false
	if req.Name != nil && existingUser.Name != *req.Name {
		existingUser.Name = *req.Name
		updateOccurred = true
		logger.Debug("User name updated", "user_id", userID, "new_name", req.Name)
	}

	// Note: Username and Email updates are typically handled by more specific methods
	// like UpdateUserProviderDetails or dedicated email/username change flows due to uniqueness constraints and verification.
	// This UpdateUser is intentionally kept simple for fields like 'Name'.

	if !updateOccurred {
		logger.Info("No changes detected for user update", "user_id", userID)
		return existingUser, nil
	}

	existingUser.LastUpdatedAt = time.Now()
	existingUser.LastUpdatedBy = updaterUserID

	err = s.userRepo.UpdateUser(ctx, existingUser)
	if err != nil {
		logger.Error("Failed to update user in repository", "user_id", existingUser.UserID, "error", err)
		return nil, fmt.Errorf("error updating user: %w", err)
	}
	logger.Info("User updated successfully", "user_id", existingUser.UserID)

	return existingUser, nil
}

func (s *userService) DeleteUser(ctx context.Context, userID string, deleterUserID string) error {
	logger := middleware.GetLoggerFromCtx(ctx)
	logger.Info("Attempting to delete user", "user_id_to_delete", userID, "deleter_user_id", deleterUserID)

	// First, check if the user exists to provide a clearer error if not.
	existingUser, err := s.userRepo.FindUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			logger.Warn("User to delete not found", "user_id", userID)
		} else {
			logger.Error("Error finding user to delete from repository", "user_id", userID, "error", err)
		}
		return fmt.Errorf("error finding user to delete: %w", err)
	}

	// Proceed with marking the user as deleted
	// The MarkUserDeleted method in the repo should handle setting DeletedAt, etc.
	// Assuming the repository method handles the actual deletion marking.
	err = s.userRepo.MarkUserDeleted(ctx, existingUser, deleterUserID)
	if err != nil {
		logger.Error("Failed to delete user from repository", "user_id", userID, "error", err)
		return fmt.Errorf("error deleting user: %w", err)
	}

	logger.Info("User deleted successfully", "user_id", userID)
	return nil
}

// AuthenticateUser checks user credentials.
func (s *userService) AuthenticateUser(ctx context.Context, username, password string) (*domain.User, error) {
	logger := middleware.GetLoggerFromCtx(ctx)
	logger.Info("Attempting to authenticate user", "username", username)

	user, err := s.userRepo.FindUserByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			logger.Warn("Authentication failed: user not found", "username", username)
		} else {
			logger.Error("Error finding user by username for authentication from repository", "username", username, "error", err)
		}
		return nil, fmt.Errorf("authentication error: %w", err) // Keep original error for context
	}

	if user.AuthProvider != domain.ProviderLocal {
		logger.Warn("Authentication attempt for non-local provider user", "username", username, "user_id", user.UserID, "auth_provider", user.AuthProvider)
		return nil, apperrors.ErrUnauthorized // Cannot use password auth for OAuth users
	}

	if user.PasswordHash == nil || *user.PasswordHash == "" {
		logger.Warn("Authentication failed: user has no password hash set", "username", username, "user_id", user.UserID)
		return nil, apperrors.ErrUnauthorized
	}

	if !utils.CheckPasswordHash(password, *user.PasswordHash) {
		logger.Warn("Authentication failed: invalid password", "username", username, "user_id", user.UserID)
		return nil, apperrors.ErrUnauthorized
	}

	logger.Info("User authenticated successfully", "username", username, "user_id", user.UserID)
	return user, nil
}

// UpdateRefreshToken stores the hashed refresh token and its expiry for a user.
func (s *userService) UpdateRefreshToken(ctx context.Context, userID string, refreshTokenHash string, refreshTokenExpiry time.Time) error {
	logger := middleware.GetLoggerFromCtx(ctx)
	logger.Info("Attempting to update refresh token", "user_id", userID)

	existingUser, err := s.userRepo.FindUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			logger.Warn("User not found for refresh token update", "user_id", userID)
		} else {
			logger.Error("Error finding user for refresh token update from repository", "user_id", userID, "error", err)
		}
		return err // Return original error (ErrNotFound or other)
	}

	err = s.userRepo.UpdateRefreshToken(ctx, existingUser, refreshTokenHash, refreshTokenExpiry)
	if err != nil {
		logger.Error("Failed to update user with refresh token in repository", "user_id", userID, "error", err)
		return fmt.Errorf("%w: failed to update user with refresh token: %v", apperrors.ErrInternal, err)
	}
	logger.Info("Refresh token updated successfully", "user_id", userID)
	return nil
}

func (s *userService) ClearRefreshToken(ctx context.Context, userID string) error {
	logger := middleware.GetLoggerFromCtx(ctx)
	logger.Info("Attempting to clear refresh token", "user_id", userID)

	if userID == "" {
		logger.Error("UserID cannot be empty for ClearRefreshToken")
		return fmt.Errorf("%w: userID cannot be empty", apperrors.ErrBadRequest)
	}

	existingUser, err := s.userRepo.FindUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			logger.Warn("User not found for refresh token update", "user_id", userID)
		} else {
			logger.Error("Error finding user for refresh token update from repository", "user_id", userID, "error", err)
		}
		return err // Return original error (ErrNotFound or other)
	}

	err = s.userRepo.ClearRefreshToken(ctx, existingUser)
	if err != nil {
		logger.Error("Failed to clear refresh token in repository", "user_id", userID, "error", err)
		// Depending on repo implementation, an error here might mean user not found or DB error.
		return fmt.Errorf("error clearing refresh token: %w", err)
	}

	logger.Info("Refresh token cleared successfully", "user_id", userID)
	return nil
}

func (s *userService) FindUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	logger := middleware.GetLoggerFromCtx(ctx)
	logger.Debug("Attempting to find user by email", "email", email)
	user, err := s.userRepo.FindUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			logger.Debug("User not found by email", "email", email)
		} else {
			logger.Error("Error finding user by email from repository", "email", email, "error", err)
		}
		return nil, err
	}
	logger.Debug("Successfully found user by email", "email", email, "user_id", user.UserID)
	return user, nil
}

// FindUserByProvider is a more specific version if domain.AuthProviderType is used directly.
func (s *userService) FindUserByProvider(ctx context.Context, provider domain.AuthProviderType, providerUserID string) (*domain.User, error) {
	logger := middleware.GetLoggerFromCtx(ctx)
	logger.Debug("Attempting to find user by provider details", "provider", provider, "provider_user_id", providerUserID)
	user, err := s.userRepo.FindUserByProviderDetails(ctx, string(provider), providerUserID)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			logger.Debug("User not found by provider details", "provider", provider, "provider_user_id", providerUserID)
		} else {
			logger.Error("Error finding user by provider details from repository", "provider", provider, "provider_user_id", providerUserID, "error", err)
		}
		return nil, err
	}
	logger.Debug("Successfully found user by provider details", "provider", provider, "provider_user_id", providerUserID, "user_id", user.UserID)
	return user, nil
}

func (s *userService) CreateOAuthUser(ctx context.Context, name, email, authProviderStr, providerUserID string, emailVerified bool) (*domain.User, error) {
	logger := middleware.GetLoggerFromCtx(ctx)
	logger.Info("Attempting to create OAuth user", "email", email, "auth_provider", authProviderStr, "provider_user_id", providerUserID)

	providerType := domain.AuthProviderType(authProviderStr)
	if providerType != domain.ProviderGoogle { // Add other supported providers here
		logger.Error("Unsupported OAuth provider for CreateOAuthUser", "auth_provider_attempted", authProviderStr)
		return nil, fmt.Errorf("%w: unsupported OAuth provider: %s", apperrors.ErrBadRequest, authProviderStr)
	}

	// Check if user already exists with this provider and providerUserID
	existingUser, err := s.userRepo.FindUserByProviderDetails(ctx, string(providerType), providerUserID)
	if err == nil && existingUser != nil {
		logger.Info("OAuth user already exists, returning existing user", "user_id", existingUser.UserID, "auth_provider", authProviderStr, "provider_user_id", providerUserID)
		return existingUser, nil // User already exists, return them
	} else if !errors.Is(err, apperrors.ErrNotFound) {
		// Handle unexpected errors from FindUserByProviderDetails
		logger.Error("Error finding user by provider details during OAuth creation", "auth_provider", authProviderStr, "provider_user_id", providerUserID, "error", err)
		return nil, fmt.Errorf("error finding user by provider details: %w", err)
	}

	// At this point, user does not exist with this specific providerID. Create them.
	now := time.Now()
	newUserID := uuid.NewString()

	user := &domain.User{
		UserID:         newUserID,
		Username:       email,
		Email:          email,
		Name:           name,
		AuthProvider:   providerType,
		ProviderUserID: providerUserID,
		IsVerified:     emailVerified,
		AuditFields: domain.AuditFields{
			CreatedAt:     now,
			LastUpdatedAt: now,
			CreatedBy:     newUserID,
			LastUpdatedBy: newUserID,
		},
	}

	// Before creating, check if the email is already in use by a *different* account.
	if email != "" {
		userByEmail, emailErr := s.FindUserByEmail(ctx, email)
		if emailErr == nil && userByEmail != nil {
			// Email exists. If it's for a different provider or a local account, it's a conflict.
			if userByEmail.AuthProvider != providerType || userByEmail.ProviderUserID != providerUserID {
				logger.Warn("Email already associated with a different account during OAuth creation", "email", email, "existing_auth_provider", userByEmail.AuthProvider, "existing_provider_user_id", userByEmail.ProviderUserID)
				return nil, fmt.Errorf("%w: email '%s' is already associated with another account", apperrors.ErrConflict, email)
			}
			// If it's the same provider and ID, existingUser check above should have caught it.
		} else if !errors.Is(emailErr, apperrors.ErrNotFound) {
			logger.Error("Error checking email for OAuth user creation from repository", "email", email, "error", emailErr)
			return nil, fmt.Errorf("error checking email for OAuth user: %w", emailErr)
		}
	}

	err = s.userRepo.SaveUser(ctx, user)
	if err != nil {
		logger.Error("Error saving new OAuth user to repository", "email", email, "auth_provider", authProviderStr, "provider_user_id", providerUserID, "error", err)
		return nil, fmt.Errorf("error saving OAuth user: %w", err)
	}

	logger.Info("New OAuth user created successfully", "user_id", user.UserID, "email", email, "auth_provider", authProviderStr)
	user.PasswordHash = nil // Ensure password hash is not exposed
	return user, nil
}

func (s *userService) UpdateUserProviderDetails(ctx context.Context, userID string, details domain.UpdateUserProviderDetails) (*domain.User, error) {
	logger := middleware.GetLoggerFromCtx(ctx)
	logger.Info("Attempting to update user provider details", "user_id", userID, "details_auth_provider", details.AuthProvider, "details_provider_user_id", details.ProviderUserID)

	user, err := s.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	updated, err := s.updateEmail(ctx, details, user, logger)
	if err != nil {
		return nil, err
	}

	updated = updated || s.updateAuthProvider(details, user, logger)
	updated = updated || s.updateProviderUserID(details, user, logger)
	updated = updated || s.updateName(details, user, logger)
	updated = updated || s.updateVerified(details, user, logger)
	updated = updated || s.updateProfilePic(details, user, logger)

	if updated {
		user.AuditFields.LastUpdatedAt = time.Now()
		user.AuditFields.LastUpdatedBy = userID // Or a system ID, or specific updater if available
		logger.Info("Applying updates to user provider details in repository", "user_id", userID)
		err = s.userRepo.UpdateUser(ctx, user)
		if err != nil {
			logger.Error("Error updating user provider details in repository", "user_id", userID, "error", err)
			return nil, fmt.Errorf("error updating user: %w", err)
		}
		logger.Info("User provider details updated successfully", "user_id", userID)
	} else {
		logger.Info("No changes detected for UpdateUserProviderDetails", "user_id", userID)
	}

	return user, nil
}

func (s *userService) updateAuthProvider(details domain.UpdateUserProviderDetails, user *domain.User, logger *slog.Logger) bool {
	if details.AuthProvider == "" || user.AuthProvider == details.AuthProvider {
		return false
	}
	logger.Debug("Updating AuthProvider", "user_id", user.UserID, "old_auth_provider", user.AuthProvider, "new_auth_provider", details.AuthProvider)
	user.AuthProvider = details.AuthProvider
	return true

}

func (s *userService) updateProviderUserID(details domain.UpdateUserProviderDetails, user *domain.User, logger *slog.Logger) bool {
	if details.ProviderUserID == "" || user.ProviderUserID == details.ProviderUserID {
		return false
	}
	logger.Debug("Updating ProviderUserID", "user_id", user.UserID, "old_provider_user_id", user.ProviderUserID, "new_provider_user_id", details.ProviderUserID)
	user.ProviderUserID = details.ProviderUserID
	return true

}

func (s *userService) updateName(details domain.UpdateUserProviderDetails, user *domain.User, logger *slog.Logger) bool {
	if details.Name == nil || user.Name == *details.Name {
		return false
	}
	logger.Debug("Updating Name", "user_id", user.UserID, "old_name", user.Name, "new_name", *details.Name)
	user.Name = *details.Name
	return true

}

func (s *userService) updateVerified(details domain.UpdateUserProviderDetails, user *domain.User, logger *slog.Logger) bool {
	if details.IsVerified == nil || user.IsVerified == *details.IsVerified {
		return false
	}
	logger.Debug("Updating IsVerified status", "user_id", user.UserID, "old_is_verified", user.IsVerified, "new_is_verified", *details.IsVerified)
	user.IsVerified = *details.IsVerified
	return true

}

func (s *userService) updateProfilePic(details domain.UpdateUserProviderDetails, user *domain.User, logger *slog.Logger) bool {
	if details.ProfilePicURL == nil || user.ProfilePicURL == *details.ProfilePicURL {
		return false
	}
	logger.Debug("Updating ProfilePicURL", "user_id", user.UserID)
	user.ProfilePicURL = *details.ProfilePicURL
	return true

}

func (s *userService) updateEmail(ctx context.Context, details domain.UpdateUserProviderDetails, user *domain.User, logger *slog.Logger) (bool, error) {
	if details.Email == nil || user.Email == *details.Email {
		return false, nil
	}
	newEmail := *details.Email
	logger.Debug("Attempting to update Email", "user_id", user.UserID, "old_email", user.Email, "new_email", newEmail)
	// Check if the new email is already in use by another user
	existingUserWithNewEmail, emailErr := s.userRepo.FindUserByEmail(ctx, newEmail)
	if emailErr == nil && existingUserWithNewEmail != nil && existingUserWithNewEmail.UserID != user.UserID {
		logger.Warn("Email update conflict: email already in use by another account", "user_id", user.UserID, "new_email", newEmail, "conflicting_user_id", existingUserWithNewEmail.UserID)
		return false, fmt.Errorf("%w: email '%s' is already in use by another account", apperrors.ErrConflict, newEmail)
	} else if !errors.Is(emailErr, apperrors.ErrNotFound) {
		// Handle unexpected errors from FindUserByEmail
		logger.Error("Error checking email for conflicts during UpdateUserProviderDetails from repository", "user_id", user.UserID, "new_email", newEmail, "error", emailErr)
		return false, fmt.Errorf("error checking email for conflicts: %w", emailErr)
	}
	user.Email = newEmail
	logger.Debug("Email updated", "user_id", user.UserID, "new_email", newEmail)
	return true, nil
}

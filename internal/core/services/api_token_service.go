package services

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/SscSPs/money_managemet_app/internal/core/domain"
	"github.com/SscSPs/money_managemet_app/internal/core/ports/repositories"
	portssvc "github.com/SscSPs/money_managemet_app/internal/core/ports/services"
	"golang.org/x/crypto/bcrypt"
)

// apiTokenService implements the APITokenSvc interface
type apiTokenService struct {
	tokenRepo repositories.APITokenRepository
	userSvc   portssvc.UserSvcFacade
}

// NewAPITokenService creates a new instance of apiTokenService
func NewAPITokenService(tokenRepo repositories.APITokenRepository, userSvc portssvc.UserSvcFacade) portssvc.APITokenSvc {
	return &apiTokenService{
		tokenRepo: tokenRepo,
		userSvc:   userSvc,
	}
}

// CreateToken generates a new API token for the user
func (s *apiTokenService) CreateToken(ctx context.Context, userID, name string, expiresIn *time.Duration) (string, *domain.APIToken, error) {
	// Validate input
	if userID == "" {
		return "", nil, errors.New("user ID is required")
	}
	if name == "" {
		return "", nil, errors.New("token name is required")
	}

	// Generate a random token
	token, err := generateSecureToken(32) // 32 bytes = 256 bits
	if err != nil {
		return "", nil, fmt.Errorf("failed to generate token: %w", err)
	}

	// Hash the token for storage
	tokenHash, err := bcrypt.GenerateFromPassword([]byte(token), bcrypt.DefaultCost)
	if err != nil {
		return "", nil, fmt.Errorf("failed to hash token: %w", err)
	}

	// Calculate expiration time
	var expiresAt *time.Time
	if expiresIn != nil {
		expiry := time.Now().Add(*expiresIn)
		expiresAt = &expiry
	}

	// Create token record
	apiToken := &domain.APIToken{
		UserID:    userID,
		Name:      name,
		TokenHash: string(tokenHash),
		ExpiresAt: expiresAt,
	}

	// Save to database
	if err := s.tokenRepo.Create(ctx, apiToken); err != nil {
		return "", nil, fmt.Errorf("failed to save token: %w", err)
	}

	// Return the plaintext token (only time it's available) and the token details
	return token, apiToken, nil
}

// ListTokens returns all API tokens for a user
func (s *apiTokenService) ListTokens(ctx context.Context, userID string) ([]domain.APIToken, error) {
	if userID == "" {
		return nil, errors.New("user ID is required")
	}

	tokens, err := s.tokenRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list tokens: %w", err)
	}

	return tokens, nil
}

// RevokeToken deletes a specific API token for a user
func (s *apiTokenService) RevokeToken(ctx context.Context, userID, tokenID string) error {
	if userID == "" || tokenID == "" {
		return errors.New("user ID and token ID are required")
	}

	// Verify the token belongs to the user
	token, err := s.tokenRepo.FindByID(ctx, tokenID)
	if err != nil {
		return fmt.Errorf("failed to find token: %w", err)
	}

	if token.UserID != userID {
		return errors.New("token not found")
	}

	// Delete the token
	if err := s.tokenRepo.Delete(ctx, tokenID); err != nil {
		return fmt.Errorf("failed to revoke token: %w", err)
	}

	return nil
}

// RevokeAllTokens deletes all API tokens for a user
func (s *apiTokenService) RevokeAllTokens(ctx context.Context, userID string) error {
	if userID == "" {
		return errors.New("user ID is required")
	}

	if err := s.tokenRepo.DeleteByUserID(ctx, userID); err != nil {
		return fmt.Errorf("failed to revoke all tokens: %w", err)
	}

	return nil
}

// ValidateToken checks if a token is valid and returns the associated user
func (s *apiTokenService) ValidateToken(ctx context.Context, tokenString string) (*domain.User, error) {
	if tokenString == "" {
		return nil, errors.New("token is required")
	}

	// Find the token by its hash (we need to hash the input to find it)
	token, err := s.tokenRepo.FindByToken(ctx, tokenString)
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	// Check if token is expired
	if token.IsExpired() {
		// Auto-revoke expired tokens
		_ = s.tokenRepo.Delete(ctx, token.ID)
		return nil, errors.New("token has expired")
	}

	// Update last used timestamp
	token.UpdateLastUsed()
	if err := s.tokenRepo.Update(ctx, token); err != nil {
		// Log the error but don't fail the validation
		// TODO: Add proper logging
	}

	// Get the associated user
	user, err := s.userSvc.GetUserByID(ctx, token.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

// generateSecureToken generates a secure random token
func generateSecureToken(length int) (string, error) {
	b := make([]byte, length)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	// Use URL-safe base64 encoding without padding
	return "mma_" + base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(b), nil
}

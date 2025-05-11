package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/SscSPs/money_managemet_app/internal/apperrors"
	"github.com/SscSPs/money_managemet_app/internal/core/domain"
	portssvc "github.com/SscSPs/money_managemet_app/internal/core/ports/services"
	"github.com/SscSPs/money_managemet_app/internal/platform/config"
	"github.com/SscSPs/money_managemet_app/internal/utils"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/idtoken" // Added for ID token validation
	// We might also need logger, etc. later
)

// tokenService implements the TokenSvcFacade for handling JWT and refresh tokens.
// It requires access to application configuration (for secrets and expiry times)
// and the user service (potentially, though maybe not directly for token generation itself).
type tokenService struct {
	cfg         *config.Config
	userService portssvc.UserSvcFacade
}

// NewTokenService creates a new instance of tokenService.
func NewTokenService(cfg *config.Config, userService portssvc.UserSvcFacade) portssvc.TokenSvcFacade {
	return &tokenService{
		cfg:         cfg,
		userService: userService,
	}
}

// GenerateAccessToken creates a new JWT access token for the given user.
func (s *tokenService) GenerateAccessToken(ctx context.Context, user *domain.User) (string, time.Time, error) {
	// Calculate expiry time first
	expiryTime := time.Now().Add(s.cfg.JWTExpiryDuration)

	accessToken, err := utils.GenerateJWT(user.UserID, s.cfg.JWTSecret, s.cfg.JWTExpiryDuration, s.cfg.JWTIssuer)
	if err != nil {
		// Consider logging the error here using a logger if available, e.g., from context
		// logger := middleware.GetLoggerFromCtx(ctx) // Example, if logger is passed or available
		// logger.ErrorContext(ctx, "Failed to generate access token", slog.String("error", err.Error()), slog.String("user_id", user.UserID))
		return "", time.Time{}, err // Propagate the error
	}
	return accessToken, expiryTime, nil
}

// GenerateRefreshToken creates a new refresh token for the given user.
func (s *tokenService) GenerateRefreshToken(ctx context.Context, user *domain.User) (string, time.Time, error) {
	// Generate a secure random string for the refresh token.
	// A common length is 32 bytes, which results in a 64-character hex string.
	rawRefreshToken, err := utils.GenerateSecureRandomString(32)
	if err != nil {
		// Consider logging the error
		return "", time.Time{}, fmt.Errorf("failed to generate secure random string for refresh token: %w", err)
	}

	expiryTime := time.Now().Add(s.cfg.RefreshTokenExpiryDuration)

	return rawRefreshToken, expiryTime, nil
}

// ValidateAndParseRefreshToken validates a refresh token string and returns the associated user.
// This will involve hashing the input token and comparing it with stored hashed tokens.
func (s *tokenService) ValidateAndParseRefreshToken(ctx context.Context, userID string, refreshTokenString string) (*domain.User, error) {
	// 1. Fetch the user by userID to get their stored refresh token hash and expiry.
	user, err := s.userService.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			// Consider logging: logger.WarnContext(ctx, "User not found for refresh token validation", slog.String("userID", userID))
			return nil, apperrors.ErrUnauthorized // Or a more specific error like apperrors.ErrInvalidRefreshToken
		}
		// Consider logging: logger.ErrorContext(ctx, "Failed to get user for refresh token validation", slog.String("userID", userID), slog.String("error", err.Error()))
		return nil, fmt.Errorf("failed to retrieve user for refresh token validation: %w", err)
	}

	// 2. Check if the user has a refresh token set and if it's expired.
	if user.RefreshTokenHash == "" || user.RefreshTokenExpiryTime == nil {
		// Consider logging: logger.WarnContext(ctx, "User has no refresh token set or expiry time", slog.String("userID", userID))
		return nil, apperrors.ErrUnauthorized // No refresh token to validate against
	}
	if time.Now().After(*user.RefreshTokenExpiryTime) {
		// Consider logging: logger.InfoContext(ctx, "Stored refresh token has expired", slog.String("userID", userID))
		return nil, apperrors.ErrRefreshTokenExpired
	}

	// 3. Compare the provided refreshTokenString (after hashing) with the stored hash.
	if !utils.CompareRefreshTokenHash(refreshTokenString, user.RefreshTokenHash) {
		// Consider logging: logger.WarnContext(ctx, "Refresh token mismatch", slog.String("userID", userID))
		return nil, apperrors.ErrUnauthorized // Token mismatch
	}

	// 4. If all checks pass, the refresh token is valid.
	return user, nil
}

// --- GoogleOAuthHandlerSvcFacade Implementation ---

// googleOAuthHandlerService implements the GoogleOAuthHandlerSvcFacade.
type googleOAuthHandlerService struct {
	cfg *config.Config
	// oauth2Config is configured at initialization time
	oauth2Config *oauth2.Config
}

// NewGoogleOAuthHandlerService creates a new instance of googleOAuthHandlerService.
func NewGoogleOAuthHandlerService(cfg *config.Config) portssvc.GoogleOAuthHandlerSvcFacade {
	return &googleOAuthHandlerService{
		cfg: cfg,
		oauth2Config: &oauth2.Config{
			ClientID:     cfg.GoogleClientID,
			ClientSecret: cfg.GoogleClientSecret,
			RedirectURL:  cfg.GoogleRedirectURL,
			Scopes:       []string{"https://www.googleapis.com/auth/userinfo.email", "https://www.googleapis.com/auth/userinfo.profile"},
			Endpoint:     google.Endpoint, // from "golang.org/x/oauth2/google"
		},
	}
}

// GenerateStateString creates a secure random string to be used as a CSRF token for OAuth flow.
func (s *googleOAuthHandlerService) GenerateStateString(ctx context.Context) (string, error) {
	// Generate a secure random string for the state. 16 bytes -> 32 char hex string
	state, err := utils.GenerateSecureRandomString(16)
	if err != nil {
		return "", fmt.Errorf("failed to generate state string for OAuth: %w", err)
	}
	return state, nil
}

// GetGoogleLoginURL returns the URL to redirect the user to for Google login.
func (s *googleOAuthHandlerService) GetGoogleLoginURL(ctx context.Context, state string) string {
	return s.oauth2Config.AuthCodeURL(state)
}

// ExchangeCodeForToken exchanges an OAuth authorization code for a token.
func (s *googleOAuthHandlerService) ExchangeCodeForToken(ctx context.Context, code string) (*oauth2.Token, error) {
	token, err := s.oauth2Config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange oauth code for token: %w", err)
	}
	return token, nil
}

// GetUserInfo uses the access token to get user information from Google.
func (s *googleOAuthHandlerService) GetUserInfo(ctx context.Context, token *oauth2.Token) (*domain.GoogleUserInfo, error) {
	// The oauth2.Token contains the AccessToken needed to make API calls.
	// We need to make an HTTP request to a Google API endpoint (e.g., userinfo endpoint).
	// The specific endpoint: https://www.googleapis.com/oauth2/v2/userinfo

	client := s.oauth2Config.Client(ctx, token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		return nil, fmt.Errorf("failed to get user info from google: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// bodyBytes, _ := io.ReadAll(resp.Body) // For debugging
		// log.Printf("Google API error response: %s", string(bodyBytes))
		return nil, fmt.Errorf("google api returned non-200 status for userinfo: %s", resp.Status)
	}

	var userInfo domain.GoogleUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, fmt.Errorf("failed to decode user info from google: %w", err)
	}

	return &userInfo, nil
}

// ValidateGoogleIDToken validates an ID token received from Google and returns the payload if valid.
func (s *googleOAuthHandlerService) ValidateGoogleIDToken(ctx context.Context, idTokenString string) (*idtoken.Payload, error) {
	if s.cfg.GoogleClientID == "" {
		// This should ideally be caught at startup, but as a safeguard:
		return nil, errors.New("google client ID is not configured in the application")
	}

	payload, err := idtoken.Validate(ctx, idTokenString, s.cfg.GoogleClientID)
	if err != nil {
		// It's good practice to wrap external library errors for context.
		// The error from idtoken.Validate can be quite descriptive, e.g., "idtoken: token expired".
		// You might want to inspect it further if specific handling is needed for different validation errors.
		return nil, fmt.Errorf("google ID token validation failed: %w", err)
	}

	// The payload contains verified user information like Email, Subject (sub), Name, etc.
	return payload, nil
}

package services

import (
	"context"
	"time"

	"github.com/SscSPs/money_managemet_app/internal/core/domain"
	"golang.org/x/oauth2"
	"google.golang.org/api/idtoken"
)

// TokenSvcFacade defines the interface for token management services.
type TokenSvcFacade interface {
	// Placeholder - Add actual methods needed by AuthHandler
	GenerateAccessToken(ctx context.Context, user *domain.User) (string, time.Time, error)
	GenerateRefreshToken(ctx context.Context, user *domain.User) (string, time.Time, error)
	// ValidateAndParseRefreshToken validates a refresh token string against a user's stored token details.
	// It returns the user if the token is valid and not expired.
	ValidateAndParseRefreshToken(ctx context.Context, userID string, refreshTokenString string) (*domain.User, error)
}

// GoogleOAuthHandlerSvcFacade defines the interface for Google OAuth operations.
type GoogleOAuthHandlerSvcFacade interface {
	// GenerateStateString creates a secure random string to be used as a CSRF token for OAuth flow.
	GenerateStateString(ctx context.Context) (string, error)
	// GetGoogleLoginURL returns the URL to redirect the user to for Google login.
	GetGoogleLoginURL(ctx context.Context, state string) string
	// ExchangeCodeForToken exchanges an OAuth authorization code for a token.
	ExchangeCodeForToken(ctx context.Context, code string) (*oauth2.Token, error)
	// GetUserInfo uses the access token to get user information from Google.
	GetUserInfo(ctx context.Context, token *oauth2.Token) (*domain.GoogleUserInfo, error)
	// ValidateGoogleIDToken validates an ID token string from Google and returns its payload.
	ValidateGoogleIDToken(ctx context.Context, idTokenString string) (*idtoken.Payload, error)
}

package handlers

import (
	"errors"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/SscSPs/money_managemet_app/internal/apperrors"
	"github.com/SscSPs/money_managemet_app/internal/core/domain"
	portssvc "github.com/SscSPs/money_managemet_app/internal/core/ports/services"
	"github.com/SscSPs/money_managemet_app/internal/middleware"

	"github.com/gin-gonic/gin"
)

// GoogleOAuthHandler handles Google OAuth related requests.
// It depends on the Google OAuth service, user service, and token service.
// It also requires access to application configuration (cfg).
// For the new flow, cfg is mainly used by the GoogleOAuthService for ClientID/Secret/RedirectURI.
type GoogleOAuthHandler struct {
	logger             *slog.Logger
	googleOAuthService portssvc.GoogleOAuthHandlerSvcFacade
	userService        portssvc.UserSvcFacade
	tokenService       portssvc.TokenSvcFacade
	// cfg *config.Config // cfg might not be directly needed here if services are fully configured
}

// NewGoogleOAuthHandler creates a new instance of GoogleOAuthHandler.
func NewGoogleOAuthHandler(
	logger *slog.Logger,
	googleOAuthService portssvc.GoogleOAuthHandlerSvcFacade,
	userService portssvc.UserSvcFacade,
	tokenService portssvc.TokenSvcFacade,
	// cfg *config.Config,
) *GoogleOAuthHandler {
	return &GoogleOAuthHandler{
		logger:             logger,
		googleOAuthService: googleOAuthService,
		userService:        userService,
		tokenService:       tokenService,
		// cfg: cfg,
	}
}

// ExchangeCodeRequest defines the expected JSON body for the /google/exchange-code endpoint.
// Note: Field names must be capitalized to be exported and thus visible to the JSON marshaller/unmarshaller.
// Use `json:"code"` tags to map to lowercase JSON field names if needed by the client.
type ExchangeCodeRequest struct {
	Code string `json:"code" binding:"required"`
}

// ExchangeCodeResponse defines the successful response for the /google/exchange-code endpoint.
type ExchangeCodeResponse struct {
	Token string `json:"token"`
}

// ExchangeCodeGoogle handles the POST request from the frontend containing the authorization code from Google.
// It exchanges the code for Google tokens, validates the ID token, creates or retrieves the user,
// generates an application-specific JWT, and returns it.
// @Summary Exchange authorization code for access token
// @Description Exchange authorization code for access token
// @Tags oauth
// @Accept  json
// @Produce  json
// @Param   code body ExchangeCodeRequest true "Authorization code"
// @Success 200 {object} ExchangeCodeResponse
// @Failure 400 {object} map[string]string "Invalid authorization code"
// @Failure 500 {object} map[string]string "Failed to exchange authorization code for access token"
// @Security BearerAuth
// @Router /google/exchange-code [post]
func (h *GoogleOAuthHandler) ExchangeCodeGoogle(c *gin.Context) {
	ctx := c.Request.Context()
	logger := middleware.GetLoggerFromCtx(c) // Use logger from context via middleware

	var req ExchangeCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.ErrorContext(ctx, "Failed to bind JSON for exchange code request", slog.String("error", err.Error()))
		// Respond with a structured error
		appErr := apperrors.NewBadRequestError("Invalid request payload: " + err.Error())
		c.JSON(appErr.Code, appErr)
		return
	}

	if req.Code == "" {
		logger.WarnContext(ctx, "Authorization code missing in exchange code request")
		appErr := apperrors.NewBadRequestError("Authorization code is required.")
		c.JSON(appErr.Code, appErr)
		return
	}

	logger.InfoContext(ctx, "Received authorization code, attempting to exchange for token with Google")

	// 1. Exchange authorization code for Google tokens
	// The googleOAuthService.ExchangeCodeForToken now uses the frontend's redirect URI from config.
	oauth2Token, err := h.googleOAuthService.ExchangeCodeForToken(ctx, req.Code)
	if err != nil {
		logger.ErrorContext(ctx, "Failed to exchange authorization code with Google", slog.String("error", err.Error()))
		appErr := apperrors.NewGatewayTimeoutError("Failed to communicate with Google OAuth service.")
		// Check if the error is due to an invalid code from Google (which is a client-side issue in this flow)
		if strings.Contains(strings.ToLower(err.Error()), "invalid_grant") || strings.Contains(strings.ToLower(err.Error()), "bad request") {
			appErr = apperrors.NewBadRequestError("Invalid or expired authorization code provided by Google.")
		}
		c.JSON(appErr.Code, appErr)
		return
	}

	// Extract ID token from Google's response
	idTokenString, ok := oauth2Token.Extra("id_token").(string)
	if !ok || idTokenString == "" {
		logger.ErrorContext(ctx, "ID token not found in Google's token response")
		appErr := apperrors.NewInternalServerError("Failed to retrieve ID token from Google.")
		c.JSON(appErr.Code, appErr)
		return
	}
	logger.InfoContext(ctx, "Successfully exchanged code for Google tokens, received ID token.")

	// 2. Validate Google's ID Token
	googleIDTokenPayload, err := h.googleOAuthService.ValidateGoogleIDToken(ctx, idTokenString)
	if err != nil {
		logger.ErrorContext(ctx, "Google ID token validation failed", slog.String("error", err.Error()))
		appErr := apperrors.NewUnauthorizedError("Invalid Google ID token: " + err.Error())
		c.JSON(appErr.Code, appErr)
		return
	}
	logger.InfoContext(ctx, "Google ID token validated successfully",
		slog.String("google_user_id", googleIDTokenPayload.Subject),
		slog.String("email", googleIDTokenPayload.Claims["email"].(string)), // Be cautious with direct type assertion
	)

	// 3. Extract User Information from validated ID token payload
	// Ensure claims exist and handle type assertions carefully
	email, _ := googleIDTokenPayload.Claims["email"].(string)
	name, _ := googleIDTokenPayload.Claims["name"].(string)
	emailVerified, _ := googleIDTokenPayload.Claims["email_verified"].(bool)
	providerUserID := googleIDTokenPayload.Subject // Google's unique ID for the user

	if email == "" || providerUserID == "" {
		logger.ErrorContext(ctx, "Essential claims (email or sub) missing from Google ID token payload",
			slog.Any("claims", googleIDTokenPayload.Claims))
		appErr := apperrors.NewInternalServerError("Essential user information missing from Google token.")
		c.JSON(appErr.Code, appErr)
		return
	}

	// 4. User Lookup/Creation in Your Database
	// This logic is similar to what was in the old CallbackGoogle
	finalUser, err := h.userService.CreateOAuthUser(
		ctx,
		name,                          // Name from Google token
		email,                         // Email from Google token
		string(domain.ProviderGoogle), // AuthProvider
		providerUserID,                // ProviderUserID (Google's 'sub' claim)
		emailVerified,                 // EmailVerified status from Google
	)

	if err != nil {
		logger.ErrorContext(ctx, "Failed to create or get OAuth user from service", slog.String("error", err.Error()), slog.String("google_user_id", providerUserID))
		var appErr *apperrors.AppError
		if errors.As(err, &appErr) {
			c.JSON(appErr.Code, appErr) // Use the code and message from the AppError
		} else {
			// Fallback for unexpected errors
			defaultErr := apperrors.NewInternalServerError("Failed to process user authentication: " + err.Error())
			c.JSON(defaultErr.Code, defaultErr)
		}
		return
	}
	logger.InfoContext(ctx, "User processed successfully via Google OAuth", slog.String("user_id", finalUser.UserID), slog.String("email", finalUser.Email))

	// 5. Generate Your Application's JWT (Access Token)
	accessToken, _, err := h.tokenService.GenerateAccessToken(ctx, finalUser)
	if err != nil {
		logger.ErrorContext(ctx, "Failed to generate application access token", slog.String("error", err.Error()), slog.String("user_id", finalUser.UserID))
		appErr := apperrors.NewInternalServerError("Failed to generate access token.")
		c.JSON(appErr.Code, appErr)
		return
	}

	// TODO: Consider if refresh token generation and setting as HTTPOnly cookie is still needed here
	// or if the frontend will manage token refresh differently with just the access token.
	// For now, returning only the access token as per frontend team's spec.

	// 6. Return Your JWT to the Frontend
	// The frontend team requested the format: { "token": "YOUR_JWT" }
	// And confirmed data should be wrapped: { "data": { "token": "YOUR_JWT" } }
	c.JSON(http.StatusOK, gin.H{
		"data": ExchangeCodeResponse{
			Token: accessToken,
		},
	})
	logger.InfoContext(ctx, "Successfully generated and returned application JWT to frontend.", slog.String("user_id", finalUser.UserID))
}

// registerGoogleOAuthRoutes registers the Google OAuth routes.
func registerGoogleOAuthRoutes(rg *gin.RouterGroup, services *portssvc.ServiceContainer) {
	h := NewGoogleOAuthHandler(slog.New(slog.NewJSONHandler(os.Stdout, nil)), services.GoogleOAuthHandler, services.User, services.TokenService)
	googleRoutes := rg.Group("/google")
	{
		googleRoutes.POST("/exchange-code", h.ExchangeCodeGoogle)
	}
}

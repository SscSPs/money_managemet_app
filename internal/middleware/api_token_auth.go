package middleware

import (
	"context"
	"log/slog"

	"github.com/SscSPs/money_managemet_app/internal/core/ports/services"
	"github.com/gin-gonic/gin"
)

// APITokenAuth is a middleware that authenticates requests using API tokens
func APITokenAuth(tokenSvc services.APITokenSvc) gin.HandlerFunc {
	return func(c *gin.Context) {
		logger := GetLoggerFromCtx(c.Request.Context())
		// Skip authentication for public routes
		if isPublicRoute(c.Request.URL.Path) {
			c.Next()
			return
		}

		// Get the Authorization header
		authHeader := c.GetHeader("x-api-key")
		if authHeader == "" {
			logger.Warn("API key not found")
			c.Next() // No api key provided, let it continue
			return
		}

		// Validate the token
		userID, err := tokenSvc.ValidateToken(c.Request.Context(), authHeader)
		if err != nil {
			logger.Warn("Invalid API token", "error", err, "token", authHeader)
			c.Next() // Token validation failed, let it continue
			return
		}

		// Token is valid, set user ID in context and skip JWT auth
		c.Set("userID", userID.UserID)
		c.Set("authMethod", "api_token")

		// Store the user ID in the context (using standard context)
		ctxWithUser := context.WithValue(c.Request.Context(), userIDKey, userID.UserID)

		// Add user ID to the logger
		enrichedLogger := logger.With(slog.String("user_id", userID.UserID))

		// Store the *enriched* logger back into the standard context
		ctxWithLoggerAndUser := context.WithValue(ctxWithUser, loggerCtxKey, enrichedLogger)

		// Update the request context
		c.Request = c.Request.WithContext(ctxWithLoggerAndUser)
		c.Next()
	}
}

// isPublicRoute checks if the given path is a public route that doesn't require authentication
func isPublicRoute(path string) bool {
	// Add public routes here
	publicRoutes := []string{
		"/api/v1/auth/login",
		"/api/v1/auth/register",
		"/api/v1/health",
		// Add other public routes as needed
	}

	for _, route := range publicRoutes {
		if path == route {
			return true
		}
	}

	return false
}

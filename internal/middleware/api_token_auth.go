package middleware

import (
	"github.com/SscSPs/money_managemet_app/internal/core/ports/services"
	"github.com/gin-gonic/gin"
)

// APITokenAuth is a middleware that authenticates requests using API tokens
func APITokenAuth(tokenSvc services.APITokenSvc) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip authentication for public routes
		if isPublicRoute(c.Request.URL.Path) {
			c.Next()
			return
		}

		// Get the Authorization header
		authHeader := c.GetHeader("x-api-key")
		if authHeader == "" {
			c.Next() // No api key provided, let it continue
			return
		}

		// Validate the token
		userID, err := tokenSvc.ValidateToken(c.Request.Context(), authHeader)
		if err != nil {
			c.Next() // Token validation failed, let it continue
			return
		}

		// Token is valid, set user ID in context and skip JWT auth
		c.Set("userID", userID)
		c.Set("authMethod", "api_token")
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

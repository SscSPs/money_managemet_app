package middleware

import (
	"net/http"
	"strings"

	"github.com/SscSPs/money_managemet_app/internal/utils"
	"github.com/gin-gonic/gin"
)

// pathsToSkip contains paths that should not be tracked by PostHog
var pathsToSkip = map[string]bool{
	"/health": true,
}

// PosthogMiddleware creates a Gin middleware handler that tracks API events with PostHog
func PosthogMiddleware(posthogClient *utils.PosthogClientWrapper) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip if PostHog is not initialized or path is in skip list
		if posthogClient == nil || !posthogClient.IsInitialized() || pathsToSkip[c.Request.URL.Path] {
			c.Next()
			return
		}

		// Process request first
		c.Next()

		// Skip if there was an error processing the request
		if len(c.Errors) > 0 || c.Writer.Status() >= http.StatusBadRequest {
			return
		}

		// Get user ID from context (set by auth middleware)
		userID, exists := GetUserIDFromContext(c)
		if !exists {
			// No user ID, can't track event
			return
		}

		// Create event name from route path (e.g., "/api/v1/workplaces" -> "api_v1_workplaces")
		eventName := strings.TrimPrefix(c.FullPath(), "/")
		eventName = strings.ReplaceAll(eventName, "/", "_")

		// Skip if event name is empty (e.g., for 404s)
		if eventName == "" {
			return
		}

		// Prepare event properties
		props := map[string]any{
			"method":      c.Request.Method,
			"path":        c.Request.URL.Path,
			"status_code": c.Writer.Status(),
		}

		// Add route parameters if any
		if len(c.Params) > 0 {
			params := make(map[string]string)
			for _, param := range c.Params {
				params[param.Key] = param.Value
			}
			props["params"] = params
		}

		// Send event to PostHog
		posthogClient.Enqueue(userID, eventName, props)
	}
}

// PosthogEvent is a helper to manually send custom events from handlers when needed
func PosthogEvent(c *gin.Context, posthogClient *utils.PosthogClientWrapper, eventName string, properties map[string]any) {
	if posthogClient == nil || !posthogClient.IsInitialized() {
		return
	}

	// Get user ID from context
	userID, exists := GetUserIDFromContext(c)
	if !exists {
		return
	}

	// Ensure properties is not nil
	if properties == nil {
		properties = make(map[string]any)
	}

	// Add request context
	properties["method"] = c.Request.Method
	properties["path"] = c.Request.URL.Path

	// Send custom event
	posthogClient.Enqueue(userID, eventName, properties)
}

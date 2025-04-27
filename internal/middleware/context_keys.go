package middleware

import "github.com/gin-gonic/gin"

// userIDKey is the key used to store the authenticated user's ID in the Gin context.
// Using a custom type prevents collisions.
const userIDKey = contextKey("userID")

// GetUserIDFromContext retrieves the authenticated user ID from the Gin context.
// It returns the user ID and a boolean indicating if it was found.
func GetUserIDFromContext(c *gin.Context) (string, bool) {
	userIDVal, exists := c.Get(string(userIDKey))
	if !exists {
		// check in the request context as well
		userIdVal := c.Request.Context().Value(userIDKey)
		if userIdVal != nil {
			return userIdVal.(string), true
		}
		return "", false
	}

	userID, ok := userIDVal.(string)
	if !ok {
		// This should not happen if the auth middleware sets it correctly
		// Consider logging an error here if it occurs.
		return "", false
	}

	return userID, true
}

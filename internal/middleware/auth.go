package middleware

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// AuthMiddleware creates a Gin middleware handler that validates JWT tokens.
func AuthMiddleware(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Retrieve logger from the standard context
		logger := GetLoggerFromCtx(c.Request.Context())
		logger.Info("AuthMiddleware", "method", c.Request.Method, "path", c.Request.URL.Path)
		// if auth is already done, skip this middleware
		if authMethod, exists := c.Get("authMethod"); exists {
			logger.Info("Auth already done", "authMethod", authMethod)
			c.Next()
			return
		}
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			logger.Warn("Authorization header missing")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			logger.Warn("Authorization header format invalid", "header", authHeader)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header format must be Bearer {token}"})
			return
		}

		tokenString := parts[1]

		// Parse and validate the token
		token, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{}, func(token *jwt.Token) (interface{}, error) {
			// Check the signing method
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("unexpected signing method")
			}
			return []byte(jwtSecret), nil
		})

		if err != nil {
			logger.Warn("Invalid token", "error", err)
			status := http.StatusUnauthorized
			msg := "Invalid token"
			if errors.Is(err, jwt.ErrTokenExpired) {
				msg = "Token has expired"
			} else if errors.Is(err, jwt.ErrTokenNotValidYet) {
				msg = "Token not valid yet"
			}
			c.AbortWithStatusJSON(status, gin.H{"error": msg})
			return
		}

		if claims, ok := token.Claims.(*jwt.RegisteredClaims); ok && token.Valid {
			userID := claims.Subject
			if userID == "" {
				logger.Error("User ID (subject) missing from valid token")
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
				return
			}

			// Store the user ID in the context (using standard context)
			ctxWithUser := context.WithValue(c.Request.Context(), userIDKey, userID)

			// Add user ID to the logger
			enrichedLogger := logger.With(slog.String("user_id", userID))

			// Store the *enriched* logger back into the standard context
			ctxWithLoggerAndUser := context.WithValue(ctxWithUser, loggerCtxKey, enrichedLogger)

			// Update the request context
			c.Request = c.Request.WithContext(ctxWithLoggerAndUser)

			// Remove setting logger in Gin context map (legacy)
			// c.Set(string(loggerKey), enrichedLogger)

			c.Next() // Proceed to the next handler
		} else {
			logger.Warn("Invalid token claims or token is not valid")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		}
	}
}

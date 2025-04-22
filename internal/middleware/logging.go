package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// loggerKey is the key used to store the logger in the Gin context.
// Using a custom type prevents collisions.
type contextKey string

const loggerKey = contextKey("logger")

// StructuredLoggingMiddleware creates a Gin middleware handler that injects
// a request-scoped logger into the context.
func StructuredLoggingMiddleware(baseLogger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		requestID := uuid.NewString()

		// Create a logger enriched with request-specific fields
		requestLogger := baseLogger.With(
			slog.String("request_id", requestID),
			slog.String("method", c.Request.Method),
			slog.String("path", c.Request.URL.Path),
		)

		// Add request ID to response header
		c.Header("X-Request-ID", requestID)

		// Store the logger in the context
		c.Set(string(loggerKey), requestLogger)

		// Process the request
		c.Next()

		// Log request completion details
		latency := time.Since(start)
		requestLogger.Info("Request completed",
			slog.Int("status", c.Writer.Status()),
			slog.Duration("latency", latency),
		)
	}
}

// GetLoggerFromContext retrieves the request-scoped logger from the Gin context.
// It returns the default logger if none is found (though this shouldn't happen
// if the middleware is applied correctly).
func GetLoggerFromContext(c *gin.Context) *slog.Logger {
	logger, exists := c.Get(string(loggerKey))
	if !exists {
		// Fallback, although ideally middleware ensures this doesn't happen
		return slog.Default()
	}

	slogLogger, ok := logger.(*slog.Logger)
	if !ok {
		// Should not happen if we set it correctly
		slog.Error("Logger in context is not of type *slog.Logger")
		return slog.Default()
	}

	return slogLogger
}

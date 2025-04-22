package middleware

import (
	"context"
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Use custom type for context key to avoid collisions
type contextKey string

const loggerCtxKey = contextKey("logger")

// StructuredLoggingMiddleware creates a Gin middleware handler that injects
// a request-scoped logger into the standard context.Context.
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

		// Create a new context with the logger
		ctx := context.WithValue(c.Request.Context(), loggerCtxKey, requestLogger)
		// Replace the request's context with the new one
		c.Request = c.Request.WithContext(ctx)

		// Process the request
		c.Next()

		// Log request completion details using the enriched logger from the final context
		finalLogger := GetLoggerFromCtx(c.Request.Context()) // Get logger potentially updated by later middleware (like auth)
		latency := time.Since(start)
		finalLogger.Info("Request completed",
			slog.Int("status", c.Writer.Status()),
			slog.Duration("latency", latency),
		)
	}
}

// GetLoggerFromGinContext retrieves the request-scoped logger from the Gin context.
// DEPRECATED: Prefer GetLoggerFromCtx if standard context.Context is available.
// This function remains for compatibility if direct Gin context access is needed
// and the logger was potentially set via c.Set by other means (though discouraged).
func GetLoggerFromGinContext(c *gin.Context) *slog.Logger {
	// First, try getting from the standard context attached to the request
	loggerFromStdCtx := GetLoggerFromCtx(c.Request.Context())
	if loggerFromStdCtx != slog.Default() {
		return loggerFromStdCtx
	}

	// Fallback to checking Gin's context map (legacy/compatibility)
	loggerVal, exists := c.Get(string(loggerCtxKey)) // Use the same key for consistency
	if exists {
		if slogLogger, ok := loggerVal.(*slog.Logger); ok {
			return slogLogger
		}
		slog.Error("Logger in Gin context is not of type *slog.Logger")
	}

	// Final fallback
	slog.Warn("No logger found in Gin context or standard context, returning default logger.")
	return slog.Default()
}

// GetLoggerFromCtx retrieves the request-scoped logger from the standard context.Context.
// It returns the default logger if none is found.
func GetLoggerFromCtx(ctx context.Context) *slog.Logger {
	loggerVal := ctx.Value(loggerCtxKey)
	if loggerVal == nil {
		// Logger not found in context, return default
		// This might happen if the context didn't pass through the middleware
		// or if the context was replaced somewhere downstream without carrying the value.
		slog.Debug("Logger not found in context, returning default logger.") // Debug level to avoid noise
		return slog.Default()
	}

	slogLogger, ok := loggerVal.(*slog.Logger)
	if !ok {
		// This indicates a programming error - something else was stored with the logger key.
		slog.Error("Value found for logger key in context is not of type *slog.Logger")
		return slog.Default()
	}

	return slogLogger
}

package middleware

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ulule/limiter/v3"
	limitergin "github.com/ulule/limiter/v3/drivers/middleware/gin"
)

// RateLimit creates a Gin middleware for rate limiting requests.
// It uses the provided limiter instance.
func RateLimit(limiterInstance *limiter.Limiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get the IP address for rate limiting
		ip := c.ClientIP()

		// Apply the rate limiting
		context, err := limiterInstance.Get(c.Request.Context(), ip)
		if err != nil {
			GetLoggerFromCtx(c.Request.Context()).Error("Failed to get rate limit context", slog.String("ip", ip), slog.String("error", err.Error()))
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Internal server error during rate limit check"})
			return
		}

		if context.Reached {
			GetLoggerFromCtx(c.Request.Context()).Warn("Rate limit exceeded", slog.String("ip", ip), slog.Int64("limit", context.Limit), slog.Int64("remaining_requests", context.Remaining))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "Too many requests. Please try again later."})
			return
		}

		c.Next()
	}
}

// GinMiddlewarize is a wrapper around limitergin.NewMiddleware
// It is kept for compatibility or specific use cases where the limitergin direct middleware is preferred.
// Generally, the RateLimit function above provides more control and logging.
func GinMiddlewarize(limiterInstance *limiter.Limiter) gin.HandlerFunc {
	return limitergin.NewMiddleware(limiterInstance)
}

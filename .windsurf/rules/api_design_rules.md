---
trigger: model_decision
description: when working on creating/updating/handling API design
---

## 🎨 API Design

### API Versioning

Use URL-based versioning for your API. This makes it clear to consumers which version of the API they are using.

```go
func RegisterRoutes(router *gin.Engine, ...) {
    api := router.Group("/api")
    {
        v1 := api.Group("/v1")
        {
            registerUserRoutes(v1, ...)
            registerAccountRoutes(v1, ...)
        }
        // v2 := api.Group("/v2")
        // {
        //     // ... register v2 routes ...
        // }
    }
}
```

### Request/Response Logging

Implement a middleware to log the details of incoming requests and outgoing responses. This is invaluable for debugging and monitoring.

```go
package middleware

import (
    "time"

    "github.com/gin-gonic/gin"
    "log/slog"
)

func RequestResponseLogger() gin.HandlerFunc {
    return func(c *gin.Context) {
        start := time.Now()

        c.Next()

        latency := time.Since(start)
        logger := GetLoggerFromCtx(c.Request.Context())

        logger.Info("request details",
            slog.String("method", c.Request.Method),
            slog.String("path", c.Request.URL.Path),
            slog.Int("status", c.Writer.Status()),
            slog.Duration("latency", latency),
            slog.String("ip", c.ClientIP()),
        )
    }
}
```

### API Design Rules:

- ✅ Use URL-based versioning for your API (e.g., `/api/v1/...`).
- ✅ Use a middleware to log the details of incoming requests and outgoing responses.
- ✅ Strive for a consistent and predictable API design.
- ✅ Use standard HTTP status codes to indicate the outcome of a request.

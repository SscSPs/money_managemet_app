---
trigger: model_decision
description: when working on creating/updating/handling code that have logsing
---


## 📝 Logging Standards

```go
import "log/slog"

// Get logger from context
logger := middleware.GetLoggerFromCtx(ctx)

// Log levels:
logger.Debug("Detailed info", "key", value)
logger.Info("Important event", "user_id", userID)
logger.Warn("Warning", slog.String("error", err.Error()))
logger.Error("Error occurred", slog.String("error", err.Error()))

// Structured logging:
logger = logger.With(slog.String("user_id", userID))
logger.Info("Processing request")

// Multiple attributes:
logger.Info("User created",
    slog.String("user_id", user.UserID),
    slog.String("username", user.Username),
    slog.Bool("is_verified", user.IsVerified))
```

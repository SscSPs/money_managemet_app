---
trigger: model_decision
description: when working on creating/updating/handling code with error handling
---


## 🔐 Error Handling

```go
import "github.com/SscSPs/money_managemet_app/internal/apperrors"

// Available sentinel errors:
apperrors.ErrNotFound
apperrors.ErrDuplicate
apperrors.ErrForbidden
apperrors.ErrUnauthorized
apperrors.ErrValidation
apperrors.ErrInternal
apperrors.ErrConflict
apperrors.ErrBadRequest

// Check with errors.Is:
if errors.Is(err, apperrors.ErrNotFound) {
    // handle not found
}

// Wrap errors:
return fmt.Errorf("%w: additional context", apperrors.ErrValidation)

// Create custom errors:
apperrors.NewAppError(http.StatusBadRequest, "Custom message", err)
apperrors.NewNotFoundError("Resource not found")
```

---
trigger: model_decision
description: when working on creating/updating/handling handlers
---

## 🎛️ Handler Pattern

```go
package handlers

type userHandler struct {
    userService portssvc.UserSvcFacade // Interface, not concrete
}

func newUserHandler(us portssvc.UserSvcFacade) *userHandler {
    return &userHandler{userService: us}
}

func registerUserRoutes(rg *gin.RouterGroup, userService portssvc.UserSvcFacade) {
    h := newUserHandler(userService)
    users := rg.Group("/users")
    {
        users.GET("/:id", h.getUser)
    }
}

func (h *userHandler) getUser(c *gin.Context) {
    logger := middleware.GetLoggerFromCtx(c.Request.Context())
    userID := c.Param("id")
    
    user, err := h.userService.GetUserByID(c.Request.Context(), userID)
    if err != nil {
        if errors.Is(err, apperrors.ErrNotFound) {
            c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
        } else {
            logger.Error("Failed to get user", slog.String("error", err.Error()))
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve user"})
        }
        return
    }
    
    c.JSON(http.StatusOK, dto.ToUserResponse(user))
}
```

**Handler Checklist:**
- ✅ Get logger: `middleware.GetLoggerFromCtx(c.Request.Context())`
- ✅ Pass context to service: `h.service.Method(c.Request.Context(), ...)`
- ✅ Use `errors.Is()` for error comparison
- ✅ Return `gin.H{"error": "message"}` for errors
- ✅ Convert domain to DTO before responding
- ❌ Never import repository packages

**HTTP Status Codes:**
- 200 OK, 201 Created, 204 No Content
- 400 Bad Request, 401 Unauthorized, 403 Forbidden
- 404 Not Found, 409 Conflict, 422 Validation Failed
- 500 Internal Server Error
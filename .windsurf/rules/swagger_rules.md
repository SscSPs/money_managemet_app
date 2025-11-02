---
trigger: model_decision
description: when working on creating/updating/handling handlers that needs swagger documentations
---


## 📚 Swagger Documentation

```go
// @Summary Get a user by ID
// @Description Retrieves details for a specific user
// @Tags users
// @Produce json
// @Param id path string true "User ID"
// @Success 200 {object} dto.UserResponse
// @Failure 404 {object} map[string]string "Not found"
// @Failure 500 {object} map[string]string "Internal error"
// @Security BearerAuth
// @Router /users/{id} [get]
func (h *userHandler) getUser(c *gin.Context) { ... }
```

**Swagger Checklist:**
- ✅ Add annotations above ALL handler methods
- ✅ Use correct tags (`@Tags users`)
- ✅ Document all params with types and descriptions
- ✅ Document all responses with status codes
- ✅ Add `@Security BearerAuth` for protected routes
- ✅ Regenerate after changes: `make swagger`

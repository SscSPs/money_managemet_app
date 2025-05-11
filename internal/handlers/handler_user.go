package handlers

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/SscSPs/money_managemet_app/internal/apperrors"
	portssvc "github.com/SscSPs/money_managemet_app/internal/core/ports/services"
	"github.com/SscSPs/money_managemet_app/internal/dto"
	"github.com/SscSPs/money_managemet_app/internal/middleware"

	"github.com/gin-gonic/gin"
)

// userHandler handles HTTP requests related to users.
type userHandler struct {
	userService portssvc.UserSvcFacade // Updated to use UserSvcFacade
}

// newUserHandler creates a new userHandler.
func newUserHandler(us portssvc.UserSvcFacade) *userHandler { // Updated interface
	return &userHandler{
		userService: us,
	}
}

// registerUserRoutes registers all user-related routes.
func registerUserRoutes(rg *gin.RouterGroup, userService portssvc.UserSvcFacade) { // Updated interface
	h := newUserHandler(userService)

	users := rg.Group("/users")
	{
		users.GET("", h.listUsers)         // Admin only
		users.GET("/:id", h.getUser)       // Own or admin
		users.PUT("/:id", h.updateUser)    // Own or admin
		users.DELETE("/:id", h.deleteUser) // Admin only
		users.POST("", h.createUser)       // Admin only
	}
}

// createUser godoc
// @Summary Create a new user
// @Description Creates a new user (typically an admin action)
// @Tags users
// @Accept  json
// @Produce  json
// @Param   user body dto.CreateUserRequest true "User details"
// @Success 201 {object} dto.UserResponse
// @Failure 400 {object} map[string]string "Invalid input"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 500 {object} map[string]string "Failed to create user"
// @Security BearerAuth
// @Router /users [post]
func (h *userHandler) createUser(c *gin.Context) {
	logger := middleware.GetLoggerFromCtx(c.Request.Context())
	var req dto.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error("Failed to bind JSON for create user request", slog.String("error", err.Error()))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format: " + err.Error()})
		return
	}

	// Get creator UserID from context
	creatorUserID, ok := middleware.GetUserIDFromContext(c)
	if !ok {
		logger.Error("Creator user ID not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// TODO: Add authorization check - is the creator an admin?

	logger = logger.With(slog.String("creator_user_id", creatorUserID))
	logger.Info("Received request to create user", slog.String("user_name", req.Name))

	createdUser, err := h.userService.CreateUser(c.Request.Context(), req)
	if err != nil {
		// TODO: Handle specific errors like duplicate username/email if implemented
		logger.Error("Failed to create user in service", slog.String("error", err.Error()))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	logger.Info("User created successfully", slog.String("new_user_id", createdUser.UserID))
	c.JSON(http.StatusCreated, dto.ToUserResponse(createdUser))
}

// getUser godoc
// @Summary Get a user by ID
// @Description Retrieves details for a specific user by their ID
// @Tags users
// @Produce  json
// @Param   id path string true "User ID"
// @Success 200 {object} dto.UserResponse
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden (trying to access another user's details)"
// @Failure 404 {object} map[string]string "User not found"
// @Failure 500 {object} map[string]string "Failed to retrieve user"
// @Security BearerAuth
// @Router /users/{id} [get]
func (h *userHandler) getUser(c *gin.Context) {
	logger := middleware.GetLoggerFromCtx(c.Request.Context())
	userID := c.Param("id")

	// Get logged-in UserID from context for authorization
	loggedInUserID, ok := middleware.GetUserIDFromContext(c)
	if !ok {
		logger.Error("Logged-in user ID not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Authorization Check: Allow users to get their own details, maybe admins get others?
	if loggedInUserID != userID {
		// TODO: Implement admin role check if admins should be allowed
		logger.Warn("User forbidden to access another user's details", slog.String("accessor_id", loggedInUserID), slog.String("target_id", userID))
		c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden"})
		return
	}

	logger = logger.With(slog.String("target_user_id", userID))
	logger.Info("Received request to get user")

	user, err := h.userService.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			logger.Warn("User not found")
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		} else {
			logger.Error("Failed to get user from service", slog.String("error", err.Error()))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve user"})
		}
		return
	}

	logger.Info("User retrieved successfully")
	c.JSON(http.StatusOK, dto.ToUserResponse(user))
}

// listUsers godoc
// @Summary List users
// @Description Retrieves a list of users (potentially admin only)
// @Tags users
// @Produce  json
// @Param   limit query int false "Limit number of results" default(20)
// @Param   offset query int false "Offset for pagination" default(0)
// @Success 200 {object} dto.ListUsersResponse
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden"
// @Failure 500 {object} map[string]string "Failed to list users"
// @Security BearerAuth
// @Router /users [get]
func (h *userHandler) listUsers(c *gin.Context) {
	logger := middleware.GetLoggerFromCtx(c.Request.Context())

	// TODO: Authorization check - Is the logged-in user an admin?
	// loggedInUserID, _ := middleware.GetUserIDFromContext(c)
	// isAdmin := checkAdminRole(loggedInUserID) // Hypothetical check
	// if !isAdmin {
	// 	 c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden"})
	// 	 return
	// }

	var params dto.ListUsersParams
	if err := c.ShouldBindQuery(&params); err != nil {
		logger.Warn("Failed to bind query params for ListUsers", slog.String("error", err.Error()))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid query parameters: " + err.Error()})
		return
	}

	logger.Info("Received request to list users", slog.Int("limit", params.Limit), slog.Int("offset", params.Offset))

	users, err := h.userService.ListUsers(c.Request.Context(), params.Limit, params.Offset)
	if err != nil {
		logger.Error("Failed to list users from service", slog.String("error", err.Error()))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list users"})
		return
	}

	logger.Info("Users listed successfully", slog.Int("count", len(users)))
	// Convert []domain.User to dto.ListUsersResponse
	userResponses := make([]dto.UserResponse, len(users))
	for i := range users {
		userResponses[i] = dto.ToUserResponse(&users[i]) // Use address of user in slice
	}
	resp := dto.ListUsersResponse{Users: userResponses}
	c.JSON(http.StatusOK, resp)
}

// updateUser godoc
// @Summary Update a user
// @Description Updates a user's details (currently only name)
// @Tags users
// @Accept  json
// @Produce  json
// @Param   id path string true "User ID to update"
// @Param   user body dto.UpdateUserRequest true "User details to update"
// @Success 200 {object} dto.UserResponse
// @Failure 400 {object} map[string]string "Invalid input"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden"
// @Failure 404 {object} map[string]string "User not found"
// @Failure 500 {object} map[string]string "Failed to update user"
// @Security BearerAuth
// @Router /users/{id} [put]
func (h *userHandler) updateUser(c *gin.Context) {
	logger := middleware.GetLoggerFromCtx(c.Request.Context())
	userID := c.Param("id")
	var req dto.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn("Failed to bind JSON for UpdateUser", slog.String("error", err.Error()))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format: " + err.Error()})
		return
	}

	// Get logged-in UserID from context for audit and authorization
	loggedInUserID, ok := middleware.GetUserIDFromContext(c)
	if !ok {
		logger.Error("Logged-in user ID not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Authorization Check: Allow users to update their own details, maybe admins update others?
	if loggedInUserID != userID {
		// TODO: Implement admin role check if admins should be allowed
		logger.Warn("User forbidden to update another user's details", slog.String("updater_id", loggedInUserID), slog.String("target_id", userID))
		c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden"})
		return
	}

	logger = logger.With(slog.String("target_user_id", userID), slog.String("updater_user_id", loggedInUserID))
	logger.Info("Received request to update user")

	updatedUser, err := h.userService.UpdateUser(c.Request.Context(), userID, req, loggedInUserID)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			logger.Warn("User not found for update")
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		} else {
			// Handle other potential errors from service
			logger.Error("Failed to update user in service", slog.String("error", err.Error()))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user"})
		}
		return
	}

	logger.Info("User updated successfully")
	c.JSON(http.StatusOK, dto.ToUserResponse(updatedUser))
}

// deleteUser godoc
// @Summary Delete a user
// @Description Marks a user as deleted (soft delete)
// @Tags users
// @Produce  json
// @Param   id path string true "User ID to delete"
// @Success 204 "No Content"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden"
// @Failure 404 {object} map[string]string "User not found"
// @Failure 500 {object} map[string]string "Failed to delete user"
// @Security BearerAuth
// @Router /users/{id} [delete]
func (h *userHandler) deleteUser(c *gin.Context) {
	logger := middleware.GetLoggerFromCtx(c.Request.Context())
	userID := c.Param("id")

	// Get logged-in UserID from context for audit and authorization
	loggedInUserID, ok := middleware.GetUserIDFromContext(c)
	if !ok {
		logger.Error("Logged-in user ID not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Authorization Check: Typically only admins can delete users, or maybe users delete themselves?
	// For now, let's assume only admins (or requires specific logic)
	// if loggedInUserID != userID { // Or check if user is admin
	// 	 logger.Warn("User forbidden to delete user", slog.String("deleter_id", loggedInUserID), slog.String("target_id", userID))
	// 	 c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden"})
	// 	 return
	// }
	// For simplicity, proceeding without strict delete authorization for now.

	logger = logger.With(slog.String("target_user_id", userID), slog.String("deleter_user_id", loggedInUserID))
	logger.Info("Received request to delete user")

	err := h.userService.DeleteUser(c.Request.Context(), userID, loggedInUserID)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			logger.Warn("User not found for deletion")
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		} else {
			logger.Error("Failed to delete user in service", slog.String("error", err.Error()))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete user"})
		}
		return
	}

	logger.Info("User deleted successfully")
	c.Status(http.StatusNoContent)
}

// TODO: Add other user handlers (List, Update, Delete) later

package handlers

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/SscSPs/money_managemet_app/internal/adapters/database/pgsql"
	"github.com/SscSPs/money_managemet_app/internal/apperrors"
	"github.com/SscSPs/money_managemet_app/internal/core/services"
	"github.com/SscSPs/money_managemet_app/internal/dto"
	"github.com/SscSPs/money_managemet_app/internal/middleware"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type userHandler struct {
	userService *services.UserService
}

func newUserHandler(userService *services.UserService) *userHandler {
	return &userHandler{
		userService: userService,
	}
}

// createUser godoc
// @Summary Create a new user
// @Description Creates a new user account
// @Tags users
// @Accept  json
// @Produce  json
// @Param   user body dto.CreateUserRequest true "User details"
// @Success 201 {object} dto.UserResponse
// @Failure 400 {object} string "Invalid input"
// @Failure 500 {object} string "Internal server error"
// @Router /users [post]
func (h *userHandler) createUser(c *gin.Context) {
	logger := middleware.GetLoggerFromContext(c)

	var createReq dto.CreateUserRequest
	if err := c.ShouldBindJSON(&createReq); err != nil {
		logger.Error("Failed to bind JSON for CreateUser", slog.String("error", err.Error()))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	// Get the ID of the user *performing* the action from the context
	creatorUserID, ok := middleware.GetUserIDFromContext(c)
	if !ok {
		// This indicates an issue with auth middleware or unauthenticated request
		logger.Error("Creator user ID not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Log the creator performing the action
	logger = logger.With(slog.String("creator_user_id", creatorUserID))

	user, err := h.userService.CreateUser(c.Request.Context(), createReq, creatorUserID)
	if err != nil {
		logger.Error("Failed to create user in service", slog.String("error", err.Error()), slog.String("requested_name", createReq.Name))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	logger.Info("User created successfully", slog.String("user_id", user.UserID))
	c.JSON(http.StatusCreated, dto.ToUserResponse(user))
}

// getUser godoc
// @Summary Get a user by ID
// @Description Retrieves details for a specific user by their ID
// @Tags users
// @Accept  json
// @Produce  json
// @Param   userID path string true "User ID"
// @Success 200 {object} dto.UserResponse
// @Failure 404 {object} string "User not found"
// @Failure 500 {object} string "Internal server error"
// @Router /users/{userID} [get]
func (h *userHandler) getUser(c *gin.Context) {
	logger := middleware.GetLoggerFromContext(c) // Get logger from context
	userID := c.Param("userID")

	user, err := h.userService.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			logger.Warn("User not found", slog.String("user_id", userID))
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}

		logger.Error("Failed to get user from service", slog.String("error", err.Error()), slog.String("user_id", userID))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve user"})
		return
	}

	logger.Debug("User retrieved successfully", slog.String("user_id", user.UserID)) // Example debug log
	c.JSON(http.StatusOK, dto.ToUserResponse(user))
}

// listUsers godoc
// @Summary List users
// @Description Retrieves a paginated list of users
// @Tags users
// @Accept  json
// @Produce  json
// @Param   limit query int false "Limit number of results" default(20)
// @Param   offset query int false "Offset for pagination" default(0)
// @Success 200 {object} dto.ListUsersResponse
// @Failure 500 {object} string "Internal server error"
// @Router /users [get]
func (h *userHandler) listUsers(c *gin.Context) {
	logger := middleware.GetLoggerFromContext(c)

	// Bind query parameters for pagination
	var params dto.ListUsersParams
	if err := c.ShouldBindQuery(&params); err != nil {
		logger.Warn("Failed to bind query parameters for ListUsers", slog.String("error", err.Error()))
		// Use defaults if binding fails, perhaps?
		// Or return bad request? For now, let service use defaults.
		// Resetting to 0 to let service layer apply defaults cleanly
		params.Limit = 0
		params.Offset = 0
	}

	// Sanitize/Validate params (optional, service/repo can also handle defaults)
	if params.Limit <= 0 {
		params.Limit = 20 // Default limit
	}
	if params.Offset < 0 {
		params.Offset = 0 // Default offset
	}

	users, err := h.userService.ListUsers(c.Request.Context(), params.Limit, params.Offset)
	if err != nil {
		logger.Error("Failed to list users from service", slog.String("error", err.Error()))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve users"})
		return
	}

	logger.Debug("Users listed successfully", slog.Int("count", len(users)), slog.Int("limit", params.Limit), slog.Int("offset", params.Offset))
	c.JSON(http.StatusOK, dto.ToListUserResponse(users))
}

// updateUser godoc
// @Summary Update a user
// @Description Updates a user's details (currently only name)
// @Tags users
// @Accept  json
// @Produce  json
// @Param   userID path string true "User ID to update"
// @Param   user body dto.UpdateUserRequest true "User details to update"
// @Success 200 {object} dto.UserResponse
// @Failure 400 {object} string "Invalid input"
// @Failure 401 {object} string "Unauthorized"
// @Failure 404 {object} string "User not found"
// @Failure 500 {object} string "Internal server error"
// @Router /users/{userID} [put]
func (h *userHandler) updateUser(c *gin.Context) {
	logger := middleware.GetLoggerFromContext(c)
	userID := c.Param("userID")

	var updateReq dto.UpdateUserRequest
	if err := c.ShouldBindJSON(&updateReq); err != nil {
		logger.Error("Failed to bind JSON for UpdateUser", slog.String("error", err.Error()), slog.String("user_id", userID))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	// Get the ID of the user *performing* the action from the context
	updaterUserID, ok := middleware.GetUserIDFromContext(c)
	if !ok {
		logger.Error("Updater user ID not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	logger = logger.With(slog.String("updater_user_id", updaterUserID))

	// TODO: Authorization check - does updaterUserID have permission to update userID?
	// For now, allow any authenticated user to update any other user.

	user, err := h.userService.UpdateUser(c.Request.Context(), userID, updateReq, updaterUserID)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			logger.Warn("User not found for update", slog.String("user_id", userID))
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		} else {
			logger.Error("Failed to update user in service", slog.String("error", err.Error()), slog.String("user_id", userID))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user"})
		}
		return
	}

	logger.Info("User updated successfully", slog.String("user_id", userID))
	c.JSON(http.StatusOK, dto.ToUserResponse(user))
}

// deleteUser godoc
// @Summary Delete a user
// @Description Soft-deletes a user by their ID
// @Tags users
// @Accept  json
// @Produce  json
// @Param   userID path string true "User ID to delete"
// @Success 204 "No Content"
// @Failure 401 {object} string "Unauthorized"
// @Failure 404 {object} string "User not found"
// @Failure 500 {object} string "Internal server error"
// @Router /users/{userID} [delete]
func (h *userHandler) deleteUser(c *gin.Context) {
	logger := middleware.GetLoggerFromContext(c)
	userID := c.Param("userID")

	// Get the ID of the user *performing* the action from the context
	deleterUserID, ok := middleware.GetUserIDFromContext(c)
	if !ok {
		logger.Error("Deleter user ID not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	logger = logger.With(slog.String("deleter_user_id", deleterUserID))

	// TODO: Authorization check - does deleterUserID have permission to delete userID?
	// Cannot delete self? Requires admin role? etc.

	err := h.userService.DeleteUser(c.Request.Context(), userID, deleterUserID)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			logger.Warn("User not found for deletion", slog.String("user_id", userID))
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		} else {
			logger.Error("Failed to delete user in service", slog.String("error", err.Error()), slog.String("user_id", userID))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete user"})
		}
		return
	}

	logger.Info("User deleted successfully", slog.String("user_id", userID))
	c.Status(http.StatusNoContent)
}

// registerUserRoutes registers user CRUD routes
func registerUserRoutes(group *gin.RouterGroup, dbPool *pgxpool.Pool) {
	// Instantiate dependencies
	userRepo := pgsql.NewUserRepository(dbPool)
	userService := services.NewUserService(userRepo)
	userHandler := newUserHandler(userService)

	// Define routes
	users := group.Group("/users")
	{
		users.POST("/", userHandler.createUser)          // Create
		users.GET("/", userHandler.listUsers)            // List (Read all)
		users.GET("/:userID", userHandler.getUser)       // Read one
		users.PUT("/:userID", userHandler.updateUser)    // Update
		users.DELETE("/:userID", userHandler.deleteUser) // Delete
	}
}

// TODO: Add other user handlers (List, Update, Delete) later

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

// workplaceHandler handles HTTP requests related to workplaces.
type workplaceHandler struct {
	workplaceService portssvc.WorkplaceSvcFacade
}

// newWorkplaceHandler creates a new workplaceHandler.
func newWorkplaceHandler(ws portssvc.WorkplaceSvcFacade) *workplaceHandler {
	return &workplaceHandler{
		workplaceService: ws,
	}
}

// registerWorkplaceRoutes registers routes related to workplaces and their members.
// It now also registers JOURNAL and ACCOUNT routes nested under a specific workplace.
func registerWorkplaceRoutes(rg *gin.RouterGroup, workplaceService portssvc.WorkplaceSvcFacade, journalService portssvc.JournalSvcFacade, accountService portssvc.AccountSvcFacade, reportingService portssvc.ReportingService) {
	h := newWorkplaceHandler(workplaceService)

	// Routes for managing workplaces themselves (e.g., creating, listing user's workplaces)
	workplacesTopLevel := rg.Group("/workplaces")
	{
		workplacesTopLevel.POST("", h.createWorkplace)
		workplacesTopLevel.GET("", h.listUserWorkplaces) // List workplaces the calling user belongs to
	}

	// Routes specific to a single workplace (identified by workplace_id)
	workplaceSpecific := rg.Group("/workplaces/:workplace_id")
	{
		// Status management endpoints
		workplaceSpecific.POST("/deactivate", h.deactivateWorkplace)
		workplaceSpecific.POST("/activate", h.activateWorkplace)

		// Manage users within a workplace
		workplaceUsers := workplaceSpecific.Group("/users")
		{
			workplaceUsers.GET("", h.listWorkplaceUsers) // New endpoint to list users in the workplace
			workplaceUsers.POST("", h.addUserToWorkplace)
			workplaceUsers.DELETE("/:user_id", h.removeUserFromWorkplace) // Remove a user from workplace
			workplaceUsers.PUT("/:user_id", h.updateUserWorkplaceRole)    // Change user's role in workplace
		}

		// -- NESTED JOURNAL ROUTES --
		// Register journal routes relative to this specific workplace group
		registerJournalRoutes(workplaceSpecific, journalService) // Pass the group and service

		// -- NESTED ACCOUNT ROUTES --
		// Register account routes relative to this specific workplace group
		RegisterAccountRoutes(workplaceSpecific, accountService, journalService) // Use exported name (no package needed)

		// -- NESTED REPORTING ROUTES --
		// Register reporting routes relative to this specific workplace group
		registerReportingRoutes(workplaceSpecific, reportingService)
	}
}

// createWorkplace godoc
// @Summary Create a new workplace
// @Description Creates a new workplace and assigns the creator as admin.
// @Tags workplaces
// @Accept  json
// @Produce  json
// @Param   workplace body dto.CreateWorkplaceRequest true "Workplace details"
// @Success 201 {object} dto.WorkplaceResponse
// @Failure 400 {object} map[string]string "Invalid input"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 500 {object} map[string]string "Failed to create workplace"
// @Security BearerAuth
// @Router /workplaces [post]
func (h *workplaceHandler) createWorkplace(c *gin.Context) {
	logger := middleware.GetLoggerFromCtx(c.Request.Context())
	var req dto.CreateWorkplaceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn("Failed to bind JSON for CreateWorkplace", slog.String("error", err.Error()))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format: " + err.Error()})
		return
	}

	creatorUserID, ok := middleware.GetUserIDFromContext(c)
	if !ok {
		logger.Error("Creator user ID not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	logger = logger.With(slog.String("creator_user_id", creatorUserID))
	logger.Info("Received request to create workplace", slog.String("workplace_name", req.Name))

	newWorkplace, err := h.workplaceService.CreateWorkplace(c.Request.Context(), req.Name, req.Description, req.DefaultCurrencyCode, creatorUserID)
	if err != nil {
		if errors.Is(err, apperrors.ErrValidation) {
			logger.Warn("Validation error creating workplace", slog.String("error", err.Error()))
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		logger.Error("Failed to create workplace in service", slog.String("error", err.Error()))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create workplace"})
		return
	}

	logger.Info("Workplace created successfully", slog.String("workplace_id", newWorkplace.WorkplaceID))
	c.JSON(http.StatusCreated, dto.ToWorkplaceResponse(newWorkplace))
}

// listUserWorkplaces godoc
// @Summary List workplaces for current user
// @Description Retrieves a list of workplaces the authenticated user belongs to.
// @Tags workplaces
// @Produce  json
// @Param includeDisabled query bool false "Include disabled workplaces (default: false)"
// @Success 200 {object} dto.ListWorkplacesResponse
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 500 {object} map[string]string "Failed to list workplaces"
// @Security BearerAuth
// @Router /workplaces [get]
func (h *workplaceHandler) listUserWorkplaces(c *gin.Context) {
	logger := middleware.GetLoggerFromCtx(c.Request.Context())

	userID, ok := middleware.GetUserIDFromContext(c)
	if !ok {
		logger.Error("User ID not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Parse the includeDisabled query parameter
	var params dto.ListUserWorkplacesParams
	if err := c.ShouldBindQuery(&params); err != nil {
		logger.Warn("Invalid query parameters", slog.String("error", err.Error()))
		// Continue with defaults rather than rejecting the request
	}

	logger = logger.With(slog.String("user_id", userID))
	logger.Info("Received request to list user's workplaces", slog.Bool("include_disabled", params.IncludeDisabled))

	workplaces, err := h.workplaceService.ListUserWorkplaces(c.Request.Context(), userID, params.IncludeDisabled)
	if err != nil {
		logger.Error("Failed to list workplaces from service", slog.String("error", err.Error()))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list workplaces"})
		return
	}

	logger.Info("Workplaces listed successfully", slog.Int("count", len(workplaces)))
	c.JSON(http.StatusOK, dto.ToListWorkplacesResponse(workplaces))
}

// addUserToWorkplace godoc
// @Summary Add a user to a workplace
// @Description Adds a specified user to a workplace with a given role (requires admin permission).
// @Tags workplaces
// @Accept  json
// @Produce  json
// @Param   workplace_id path string true "Workplace ID"
// @Param   user_details body dto.AddUserToWorkplaceRequest true "User ID and Role"
// @Success 204 "No Content"
// @Failure 400 {object} map[string]string "Invalid input"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden (caller is not admin)"
// @Failure 404 {object} map[string]string "Workplace or User not found"
// @Failure 500 {object} map[string]string "Failed to add user"
// @Security BearerAuth
// @Router /workplaces/{workplace_id}/users [post]
func (h *workplaceHandler) addUserToWorkplace(c *gin.Context) {
	logger := middleware.GetLoggerFromCtx(c.Request.Context())
	workplaceID := c.Param("workplace_id")

	var req dto.AddUserToWorkplaceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn("Failed to bind JSON for AddUserToWorkplace", slog.String("error", err.Error()))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format: " + err.Error()})
		return
	}

	addingUserID, ok := middleware.GetUserIDFromContext(c)
	if !ok {
		logger.Error("Adding user ID not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	logger = logger.With(slog.String("adding_user_id", addingUserID), slog.String("workplace_id", workplaceID), slog.String("target_user_id", req.UserID))
	logger.Info("Received request to add user to workplace", slog.String("role", string(req.Role)))

	err := h.workplaceService.AddUserToWorkplace(c.Request.Context(), addingUserID, req.UserID, workplaceID, req.Role)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			logger.Warn("Add user failed: Workplace/User not found or Adding User not member/admin")
			c.JSON(http.StatusNotFound, gin.H{"error": "Resource not found or insufficient permissions"}) // Combine NotFound/Forbidden from service perspective
		} else if errors.Is(err, apperrors.ErrForbidden) {
			logger.Warn("Add user failed: Forbidden")
			c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden"})
		} else {
			logger.Error("Failed to add user to workplace in service", slog.String("error", err.Error()))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add user to workplace"})
		}
		return
	}

	logger.Info("User added to workplace successfully")
	c.Status(http.StatusNoContent)
}

// deactivateWorkplace godoc
// @Summary Deactivate a workplace
// @Description Marks a workplace as inactive (requires admin permission).
// @Tags workplaces
// @Accept  json
// @Produce  json
// @Param   workplace_id path string true "Workplace ID"
// @Param   request body dto.DeactivateWorkplaceRequest false "Deactivation details (optional)"
// @Success 204 "No Content"
// @Failure 400 {object} map[string]string "Invalid input"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden (caller is not admin)"
// @Failure 404 {object} map[string]string "Workplace not found"
// @Failure 500 {object} map[string]string "Failed to deactivate workplace"
// @Security BearerAuth
// @Router /workplaces/{workplace_id}/deactivate [post]
func (h *workplaceHandler) deactivateWorkplace(c *gin.Context) {
	logger := middleware.GetLoggerFromCtx(c.Request.Context())
	workplaceID := c.Param("workplace_id")

	userID, ok := middleware.GetUserIDFromContext(c)
	if !ok {
		logger.Error("User ID not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Optional: Parse request body if needed in the future
	var req dto.DeactivateWorkplaceRequest
	if err := c.ShouldBindJSON(&req); err != nil && err.Error() != "EOF" {
		logger.Warn("Failed to bind JSON for DeactivateWorkplace", slog.String("error", err.Error()))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format: " + err.Error()})
		return
	}

	logger = logger.With(slog.String("user_id", userID), slog.String("workplace_id", workplaceID))
	logger.Info("Received request to deactivate workplace")

	err := h.workplaceService.DeactivateWorkplace(c.Request.Context(), workplaceID, userID)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			logger.Warn("Deactivate workplace failed: Workplace not found or User not member")
			c.JSON(http.StatusNotFound, gin.H{"error": "Workplace not found or user not a member"})
		} else if errors.Is(err, apperrors.ErrForbidden) {
			logger.Warn("Deactivate workplace failed: User is not an admin")
			c.JSON(http.StatusForbidden, gin.H{"error": "Only workplace admins can deactivate workplaces"})
		} else {
			logger.Error("Failed to deactivate workplace", slog.String("error", err.Error()))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to deactivate workplace"})
		}
		return
	}

	logger.Info("Workplace deactivated successfully")
	c.Status(http.StatusNoContent)
}

// activateWorkplace godoc
// @Summary Activate a workplace
// @Description Marks a workplace as active (requires admin permission).
// @Tags workplaces
// @Accept  json
// @Produce  json
// @Param   workplace_id path string true "Workplace ID"
// @Success 204 "No Content"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden (caller is not admin)"
// @Failure 404 {object} map[string]string "Workplace not found"
// @Failure 500 {object} map[string]string "Failed to activate workplace"
// @Security BearerAuth
// @Router /workplaces/{workplace_id}/activate [post]
func (h *workplaceHandler) activateWorkplace(c *gin.Context) {
	logger := middleware.GetLoggerFromCtx(c.Request.Context())
	workplaceID := c.Param("workplace_id")

	userID, ok := middleware.GetUserIDFromContext(c)
	if !ok {
		logger.Error("User ID not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	logger = logger.With(slog.String("user_id", userID), slog.String("workplace_id", workplaceID))
	logger.Info("Received request to activate workplace")

	err := h.workplaceService.ActivateWorkplace(c.Request.Context(), workplaceID, userID)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			logger.Warn("Activate workplace failed: Workplace not found or User not member")
			c.JSON(http.StatusNotFound, gin.H{"error": "Workplace not found or user not a member"})
		} else if errors.Is(err, apperrors.ErrForbidden) {
			logger.Warn("Activate workplace failed: User is not an admin")
			c.JSON(http.StatusForbidden, gin.H{"error": "Only workplace admins can activate workplaces"})
		} else {
			logger.Error("Failed to activate workplace", slog.String("error", err.Error()))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to activate workplace"})
		}
		return
	}

	logger.Info("Workplace activated successfully")
	c.Status(http.StatusNoContent)
}

// listWorkplaceUsers godoc
// @Summary List users in a workplace
// @Description Retrieves a list of users and their roles in the specified workplace.
// @Tags workplaces
// @Produce json
// @Param workplace_id path string true "Workplace ID"
// @Success 200 {object} dto.ListWorkplaceUsersResponse
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden (caller is not a member)"
// @Failure 404 {object} map[string]string "Workplace not found"
// @Failure 500 {object} map[string]string "Failed to list workplace users"
// @Security BearerAuth
// @Router /workplaces/{workplace_id}/users [get]
func (h *workplaceHandler) listWorkplaceUsers(c *gin.Context) {
	logger := middleware.GetLoggerFromCtx(c.Request.Context())
	workplaceID := c.Param("workplace_id")

	userID, ok := middleware.GetUserIDFromContext(c)
	if !ok {
		logger.Error("User ID not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	logger = logger.With(slog.String("user_id", userID), slog.String("workplace_id", workplaceID))
	logger.Info("Received request to list workplace users")

	users, err := h.workplaceService.ListWorkplaceUsers(c.Request.Context(), workplaceID, userID)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			logger.Warn("List workplace users failed: Workplace not found")
			c.JSON(http.StatusNotFound, gin.H{"error": "Workplace not found"})
		} else if errors.Is(err, apperrors.ErrForbidden) {
			logger.Warn("List workplace users failed: User is not a member")
			c.JSON(http.StatusForbidden, gin.H{"error": "You must be a member of this workplace to view its users"})
		} else {
			logger.Error("Failed to list workplace users", slog.String("error", err.Error()))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list workplace users"})
		}
		return
	}

	logger.Info("Workplace users listed successfully", slog.Int("count", len(users)))
	c.JSON(http.StatusOK, dto.ToListWorkplaceUsersResponse(users))
}

// removeUserFromWorkplace godoc
// @Summary Remove a user from a workplace
// @Description Removes a user from a workplace (requires admin permission)
// @Tags workplaces
// @Produce json
// @Param workplace_id path string true "Workplace ID"
// @Param user_id path string true "User ID to remove"
// @Success 204 "No Content"
// @Failure 400 {object} map[string]string "Invalid input"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden (caller is not admin)"
// @Failure 404 {object} map[string]string "Workplace or User not found"
// @Failure 422 {object} map[string]string "Cannot remove the last admin"
// @Failure 500 {object} map[string]string "Failed to remove user"
// @Security BearerAuth
// @Router /workplaces/{workplace_id}/users/{user_id} [delete]
func (h *workplaceHandler) removeUserFromWorkplace(c *gin.Context) {
	logger := middleware.GetLoggerFromCtx(c.Request.Context())
	workplaceID := c.Param("workplace_id")
	targetUserID := c.Param("user_id")

	requestingUserID, ok := middleware.GetUserIDFromContext(c)
	if !ok {
		logger.Error("Requesting user ID not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	logger = logger.With(
		slog.String("requesting_user_id", requestingUserID),
		slog.String("workplace_id", workplaceID),
		slog.String("target_user_id", targetUserID),
	)
	logger.Info("Received request to remove user from workplace")

	// Call service method to remove the user
	err := h.workplaceService.RemoveUserFromWorkplace(c.Request.Context(), requestingUserID, targetUserID, workplaceID)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			logger.Warn("Remove user failed: Workplace or User not found")
			c.JSON(http.StatusNotFound, gin.H{"error": "Workplace or User not found"})
		} else if errors.Is(err, apperrors.ErrForbidden) {
			logger.Warn("Remove user failed: Forbidden")
			c.JSON(http.StatusForbidden, gin.H{"error": "You must be an admin to remove users"})
		} else if errors.Is(err, apperrors.ErrValidation) {
			logger.Warn("Cannot remove the last admin from workplace", slog.String("error", err.Error()))
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "Cannot remove the last admin from the workplace"})
		} else {
			logger.Error("Failed to remove user from workplace", slog.String("error", err.Error()))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove user from workplace"})
		}
		return
	}

	logger.Info("User removed from workplace successfully")
	c.Status(http.StatusNoContent)
}

// updateUserWorkplaceRole godoc
// @Summary Update a user's role in a workplace
// @Description Updates a user's role in a workplace (requires admin permission)
// @Tags workplaces
// @Accept json
// @Produce json
// @Param workplace_id path string true "Workplace ID"
// @Param user_id path string true "User ID to update"
// @Param role body dto.UpdateUserRoleRequest true "New role for the user"
// @Success 204 "No Content"
// @Failure 400 {object} map[string]string "Invalid input"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden (caller is not admin)"
// @Failure 404 {object} map[string]string "Workplace or User not found"
// @Failure 422 {object} map[string]string "Cannot demote the last admin"
// @Failure 500 {object} map[string]string "Failed to update role"
// @Security BearerAuth
// @Router /workplaces/{workplace_id}/users/{user_id} [put]
func (h *workplaceHandler) updateUserWorkplaceRole(c *gin.Context) {
	logger := middleware.GetLoggerFromCtx(c.Request.Context())
	workplaceID := c.Param("workplace_id")
	targetUserID := c.Param("user_id")

	// Parse request body
	var req dto.UpdateUserRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn("Failed to bind JSON for UpdateUserRole", slog.String("error", err.Error()))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format: " + err.Error()})
		return
	}

	requestingUserID, ok := middleware.GetUserIDFromContext(c)
	if !ok {
		logger.Error("Requesting user ID not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	logger = logger.With(
		slog.String("requesting_user_id", requestingUserID),
		slog.String("workplace_id", workplaceID),
		slog.String("target_user_id", targetUserID),
		slog.String("new_role", string(req.Role)),
	)
	logger.Info("Received request to update user role in workplace")

	// Call service method to update the user's role
	err := h.workplaceService.UpdateUserWorkplaceRole(c.Request.Context(), requestingUserID, targetUserID, workplaceID, req.Role)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			logger.Warn("Update role failed: Workplace or User not found")
			c.JSON(http.StatusNotFound, gin.H{"error": "Workplace or User not found"})
		} else if errors.Is(err, apperrors.ErrForbidden) {
			logger.Warn("Update role failed: Forbidden")
			c.JSON(http.StatusForbidden, gin.H{"error": "You must be an admin to update user roles"})
		} else if errors.Is(err, apperrors.ErrValidation) {
			logger.Warn("Cannot demote the last admin in workplace", slog.String("error", err.Error()))
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "Cannot demote the last admin in the workplace"})
		} else {
			logger.Error("Failed to update user role in workplace", slog.String("error", err.Error()))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user role"})
		}
		return
	}

	logger.Info("User role updated successfully")
	c.Status(http.StatusNoContent)
}

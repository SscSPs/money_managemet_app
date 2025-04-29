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
	workplaceService portssvc.WorkplaceService
}

// newWorkplaceHandler creates a new workplaceHandler.
func newWorkplaceHandler(ws portssvc.WorkplaceService) *workplaceHandler {
	return &workplaceHandler{
		workplaceService: ws,
	}
}

// registerWorkplaceRoutes registers routes related to workplaces and their members.
// It now also registers JOURNAL and ACCOUNT routes nested under a specific workplace.
func registerWorkplaceRoutes(rg *gin.RouterGroup, workplaceService portssvc.WorkplaceService, journalService portssvc.JournalService, accountService portssvc.AccountService) {
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
		// TODO: Add GET /workplaces/:workplace_id to get details?
		// TODO: Add PUT /workplaces/:workplace_id to update?
		// TODO: Add DELETE /workplaces/:workplace_id?

		// Manage users within a workplace
		workplaceUsers := workplaceSpecific.Group("/users")
		{
			workplaceUsers.POST("", h.addUserToWorkplace)
			// TODO: Add GET /users to list users in the workplace?
			// TODO: Add DELETE /users/:user_id to remove a user?
			// TODO: Add PUT /users/:user_id to change role?
		}

		// -- NESTED JOURNAL ROUTES --
		// Register journal routes relative to this specific workplace group
		registerJournalRoutes(workplaceSpecific, journalService) // Pass the group and service

		// -- NESTED ACCOUNT ROUTES --
		// Register account routes relative to this specific workplace group
		RegisterAccountRoutes(workplaceSpecific, accountService, journalService) // Use exported name (no package needed)
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

	newWorkplace, err := h.workplaceService.CreateWorkplace(c.Request.Context(), req.Name, req.Description, creatorUserID)
	if err != nil {
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

	logger = logger.With(slog.String("user_id", userID))
	logger.Info("Received request to list user's workplaces")

	workplaces, err := h.workplaceService.ListUserWorkplaces(c.Request.Context(), userID)
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

package handlers

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/SscSPs/money_managemet_app/internal/apperrors"                    // Import if needed for DTO conversion
	portssvc "github.com/SscSPs/money_managemet_app/internal/core/ports/services" // Use ports services

	// "github.com/SscSPs/money_managemet_app/internal/core/services" // Remove concrete services
	"github.com/SscSPs/money_managemet_app/internal/dto"
	"github.com/SscSPs/money_managemet_app/internal/middleware"
	"github.com/gin-gonic/gin"
	// For balance calculation
)

// safeStringDeref safely dereferences a string pointer, returning "" if nil.
func safeStringDeref(s *string) string {
	if s != nil {
		return *s
	}
	return ""
}

// journalHandler handles HTTP requests related to journals.
type journalHandler struct {
	journalService portssvc.JournalSvcFacade // Updated to use JournalSvcFacade
}

// newJournalHandler creates a new journalHandler.
func newJournalHandler(js portssvc.JournalSvcFacade) *journalHandler { // Updated interface
	return &journalHandler{
		journalService: js,
	}
}

// registerJournalRoutes registers all routes related to journals.
func registerJournalRoutes(rg *gin.RouterGroup, journalService portssvc.JournalSvcFacade) { // Updated interface
	h := newJournalHandler(journalService)

	journals := rg.Group("/journals")
	{
		journals.POST("", h.createJournal)
		journals.GET("/:id", h.getJournal)
		journals.GET("", h.listJournals)
		journals.PUT("/:id", h.updateJournal)
		journals.DELETE("/:id", h.deleteJournal)
		journals.POST("/:id/reverse", h.reverseJournal)
	}
}

// createJournal godoc
// @Summary Create a new journal in workplace
// @Description Creates a new journal entry within the specified workplace.
// @Tags journals
// @Accept  json
// @Produce  json
// @Param   workplace_id path string true "Workplace ID"
// @Param   journal body dto.CreateJournalRequest true "Journal details"
// @Success 201 {object} dto.JournalResponse
// @Failure 400 {object} map[string]string "Invalid input or missing Workplace ID"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden (User cannot create in this workplace)"
// @Failure 500 {object} map[string]string "Failed to create journal"
// @Security BearerAuth
// @Router /workplaces/{workplace_id}/journals [post]
func (h *journalHandler) createJournal(c *gin.Context) {
	logger := middleware.GetLoggerFromCtx(c.Request.Context())
	workplaceID := c.Param("workplace_id") // Get from path
	if workplaceID == "" {
		logger.Error("Workplace ID missing from path for createJournal")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Workplace ID required in path"})
		return
	}

	var req dto.CreateJournalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn("Failed to bind JSON for CreateJournal", slog.String("error", err.Error()))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format: " + err.Error()})
		return
	}

	creatorUserID, ok := middleware.GetUserIDFromContext(c)
	if !ok {
		logger.Error("Creator user ID not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	logger = logger.With(slog.String("creator_user_id", creatorUserID), slog.String("workplace_id", workplaceID))
	logger.Info("Received request to create journal", slog.Time("date", req.Date), slog.String("description", req.Description))

	newJournal, err := h.journalService.CreateJournal(c.Request.Context(), workplaceID, req, creatorUserID) // Pass workplaceID & creatorUserID
	if err != nil {
		if errors.Is(err, apperrors.ErrForbidden) {
			logger.Warn("User forbidden to create journal in workplace", slog.String("user_id", creatorUserID), slog.String("workplace_id", workplaceID))
			c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden"})
		} else if errors.Is(err, apperrors.ErrValidation) || errors.Is(err, apperrors.ErrNotFound) {
			logger.Warn("Validation/NotFound error creating journal", slog.String("error", err.Error()))
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		} else {
			logger.Error("Failed to create journal in service", slog.String("error", err.Error()))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create journal"})
		}
		return
	}

	logger.Info("Journal created successfully", slog.String("journal_id", newJournal.JournalID))
	c.JSON(http.StatusCreated, dto.ToJournalResponse(newJournal))
}

// getJournal godoc
// @Summary Get a journal by ID from workplace
// @Description Retrieves details for a specific journal entry by its ID within a workplace.
// @Tags journals
// @Produce  json
// @Param   workplace_id path string true "Workplace ID"
// @Param   id path string true "Journal ID"
// @Success 200 {object} dto.JournalResponse
// @Failure 400 {object} map[string]string "Missing Workplace or Journal ID"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden (User not part of workplace)"
// @Failure 404 {object} map[string]string "Journal not found in this workplace"
// @Failure 500 {object} map[string]string "Failed to retrieve journal"
// @Security BearerAuth
// @Router /workplaces/{workplace_id}/journals/{id} [get]
func (h *journalHandler) getJournal(c *gin.Context) {
	logger := middleware.GetLoggerFromCtx(c.Request.Context())
	workplaceID := c.Param("workplace_id") // Get from path
	journalID := c.Param("id")
	if workplaceID == "" || journalID == "" {
		logger.Error("Workplace ID or Journal ID missing from path for getJournal")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Workplace and Journal ID required in path"})
		return
	}

	loggedInUserID, ok := middleware.GetUserIDFromContext(c)
	if !ok {
		logger.Error("Logged-in user ID not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	logger = logger.With(slog.String("target_journal_id", journalID), slog.String("workplace_id", workplaceID), slog.String("requesting_user_id", loggedInUserID))
	logger.Info("Received request to get journal")

	journal, err := h.journalService.GetJournalByID(c.Request.Context(), workplaceID, journalID, loggedInUserID)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			logger.Warn("Journal not found or not in this workplace")
			c.JSON(http.StatusNotFound, gin.H{"error": "Journal not found"})
		} else if errors.Is(err, apperrors.ErrForbidden) {
			logger.Warn("User forbidden to access journal workplace", slog.String("user_id", loggedInUserID), slog.String("workplace_id", workplaceID))
			c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden"})
		} else {
			logger.Error("Failed to get journal from service", slog.String("error", err.Error()))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve journal"})
		}
		return
	}

	// Auth check is handled by the service layer (ensuring user is in workplace and journal belongs to it)
	logger.Info("Journal retrieved successfully")
	c.JSON(http.StatusOK, dto.ToJournalResponse(journal))
}

// listJournals godoc
// @Summary List journals for current user in workplace
// @Description Retrieves a list of journals for the specified workplace if the user is a member.
// @Tags journals
// @Produce  json
// @Param   workplace_id path string true "Workplace ID"
// @Param   limit query int false "Limit number of results" default(20)
// @Param   offset query int false "Offset for pagination" default(0)
// @Success 200 {object} dto.ListJournalsResponse
// @Failure 400 {object} map[string]string "Missing Workplace ID"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden (User not part of workplace)"
// @Failure 500 {object} map[string]string "Failed to list journals"
// @Security BearerAuth
// @Router /workplaces/{workplace_id}/journals [get]
func (h *journalHandler) listJournals(c *gin.Context) {
	logger := middleware.GetLoggerFromCtx(c.Request.Context())

	workplaceID := c.Param("workplace_id") // Get from path
	if workplaceID == "" {
		logger.Error("Workplace ID missing from request path for listJournals")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Workplace ID required in path"})
		return
	}

	loggedInUserID, ok := middleware.GetUserIDFromContext(c)
	if !ok {
		logger.Error("Logged-in user ID not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var params dto.ListJournalsParams
	if err := c.ShouldBindQuery(&params); err != nil {
		logger.Warn("Failed to bind query params for ListJournals", slog.String("error", err.Error()))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid query parameters: " + err.Error()})
		return
	}

	logger = logger.With(slog.String("user_id", loggedInUserID), slog.String("workplace_id", workplaceID))
	logger.Info("Received request to list journals", slog.Int("limit", params.Limit), slog.String("nextToken", safeStringDeref(params.NextToken)))

	resp, err := h.journalService.ListJournals(c.Request.Context(), workplaceID, loggedInUserID, params)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			logger.Warn("User forbidden from workplace for list journals")
			c.JSON(http.StatusNotFound, gin.H{"error": "Workplace not found or access denied"})
		} else if errors.Is(err, apperrors.ErrForbidden) {
			logger.Warn("User forbidden to list journals for workplace", slog.String("user_id", loggedInUserID), slog.String("workplace_id", workplaceID))
			c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden"})
		} else {
			logger.Error("Failed to list journals from service", slog.String("error", err.Error()))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list journals"})
		}
		return
	}

	logger.Info("Journals listed successfully", slog.Int("count", len(resp.Journals)))
	c.JSON(http.StatusOK, resp)
}

// updateJournal godoc
// @Summary Update a journal entry in workplace
// @Description Updates details (like description, date) for a specific journal entry within a workplace.
// @Tags journals
// @Accept  json
// @Produce  json
// @Param   workplace_id path string true "Workplace ID"
// @Param   id path string true "Journal ID"
// @Param   journal body dto.UpdateJournalRequest true "Journal details to update"
// @Success 200 {object} dto.JournalResponse
// @Failure 400 {object} map[string]string "Invalid input or missing IDs"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden (User cannot update)"
// @Failure 404 {object} map[string]string "Journal not found in this workplace"
// @Failure 500 {object} map[string]string "Failed to update journal"
// @Security BearerAuth
// @Router /workplaces/{workplace_id}/journals/{id} [put]
func (h *journalHandler) updateJournal(c *gin.Context) {
	logger := middleware.GetLoggerFromCtx(c.Request.Context())
	workplaceID := c.Param("workplace_id") // Get from path
	journalID := c.Param("id")
	if workplaceID == "" || journalID == "" {
		logger.Error("Workplace ID or Journal ID missing from path for updateJournal")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Workplace and Journal ID required in path"})
		return
	}

	var req dto.UpdateJournalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn("Failed to bind JSON for UpdateJournal", slog.String("error", err.Error()))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format: " + err.Error()})
		return
	}

	loggedInUserID, ok := middleware.GetUserIDFromContext(c)
	if !ok {
		logger.Error("Logged-in user ID not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	logger = logger.With(slog.String("target_journal_id", journalID), slog.String("workplace_id", workplaceID), slog.String("updater_user_id", loggedInUserID))
	logger.Info("Received request to update journal")

	updatedJournal, err := h.journalService.UpdateJournal(c.Request.Context(), workplaceID, journalID, req, loggedInUserID)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			logger.Warn("Journal not found for update (or in wrong workplace)")
			c.JSON(http.StatusNotFound, gin.H{"error": "Journal not found"})
		} else if errors.Is(err, apperrors.ErrForbidden) {
			logger.Warn("User forbidden to update journal", slog.String("user_id", loggedInUserID), slog.String("journal_id", journalID))
			c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden"})
		} else if errors.Is(err, apperrors.ErrValidation) {
			logger.Warn("Validation error updating journal", slog.String("error", err.Error()))
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		} else {
			logger.Error("Failed to update journal in service", slog.String("error", err.Error()))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update journal"})
		}
		return
	}

	logger.Info("Journal updated successfully")
	c.JSON(http.StatusOK, dto.ToJournalResponse(updatedJournal))
}

// deleteJournal godoc
// @Summary Deactivate a journal entry in workplace
// @Description Marks a journal entry as inactive within a specified workplace.
// @Tags journals
// @Produce  json
// @Param   workplace_id path string true "Workplace ID"
// @Param   id path string true "Journal ID to deactivate"
// @Success 204 "No Content"
// @Failure 400 {object} map[string]string "Missing Workplace or Journal ID"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden (User cannot deactivate)"
// @Failure 404 {object} map[string]string "Journal not found in this workplace"
// @Failure 409 {object} map[string]string "Conflict (e.g., already inactive)"
// @Failure 500 {object} map[string]string "Failed to deactivate journal"
// @Security BearerAuth
// @Router /workplaces/{workplace_id}/journals/{id} [delete]
func (h *journalHandler) deleteJournal(c *gin.Context) {
	logger := middleware.GetLoggerFromCtx(c.Request.Context())
	workplaceID := c.Param("workplace_id") // Get from path
	journalID := c.Param("id")
	if workplaceID == "" || journalID == "" {
		logger.Error("Workplace ID or Journal ID missing from path for deleteJournal")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Workplace and Journal ID required in path"})
		return
	}

	loggedInUserID, ok := middleware.GetUserIDFromContext(c)
	if !ok {
		logger.Error("Logged-in user ID not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	logger = logger.With(slog.String("target_journal_id", journalID), slog.String("workplace_id", workplaceID), slog.String("deleter_user_id", loggedInUserID))
	logger.Info("Received request to delete journal")

	err := h.journalService.DeactivateJournal(c.Request.Context(), workplaceID, journalID, loggedInUserID)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			logger.Warn("Journal not found for delete (or in wrong workplace)")
			c.JSON(http.StatusNotFound, gin.H{"error": "Journal not found"})
		} else if errors.Is(err, apperrors.ErrForbidden) {
			logger.Warn("User forbidden to deactivate journal", slog.String("user_id", loggedInUserID), slog.String("journal_id", journalID))
			c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden"})
		} else if errors.Is(err, apperrors.ErrValidation) {
			logger.Warn("Validation error deactivating journal (already inactive?)", slog.String("error", err.Error()))
			c.JSON(http.StatusConflict, gin.H{"error": "Journal already inactive or cannot be deactivated"})
		} else {
			logger.Error("Failed to deactivate journal in service", slog.String("error", err.Error()))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to deactivate journal"})
		}
		return
	}

	logger.Info("Journal deleted successfully")
	c.Status(http.StatusNoContent)
}

// reverseJournal godoc
// @Summary Reverse a journal entry in workplace
// @Description Reverses a specific journal entry by creating a new journal with opposite transaction types.
// @Tags journals
// @Produce  json
// @Param   workplace_id path string true "Workplace ID"
// @Param   id path string true "Journal ID to reverse"
// @Success 200 {object} dto.JournalResponse "The newly created reversing journal entry"
// @Failure 400 {object} map[string]string "Missing Workplace or Journal ID"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden (User cannot reverse)"
// @Failure 404 {object} map[string]string "Journal not found in this workplace"
// @Failure 409 {object} map[string]string "Conflict (e.g., journal already reversed or not posted)"
// @Failure 500 {object} map[string]string "Failed to reverse journal"
// @Security BearerAuth
// @Router /workplaces/{workplace_id}/journals/{id}/reverse [post]
func (h *journalHandler) reverseJournal(c *gin.Context) {
	logger := middleware.GetLoggerFromCtx(c.Request.Context())
	workplaceID := c.Param("workplace_id")
	journalID := c.Param("id")
	if workplaceID == "" || journalID == "" {
		logger.Error("Workplace ID or Journal ID missing from path for reverseJournal")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Workplace and Journal ID required in path"})
		return
	}

	loggedInUserID, ok := middleware.GetUserIDFromContext(c)
	if !ok {
		logger.Error("Logged-in user ID not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	logger = logger.With(slog.String("target_journal_id", journalID), slog.String("workplace_id", workplaceID), slog.String("reverser_user_id", loggedInUserID))
	logger.Info("Received request to reverse journal")

	reversingJournal, err := h.journalService.ReverseJournal(c.Request.Context(), workplaceID, journalID, loggedInUserID)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			logger.Warn("Journal not found for reversal (or in wrong workplace)")
			c.JSON(http.StatusNotFound, gin.H{"error": "Journal not found"})
		} else if errors.Is(err, apperrors.ErrForbidden) {
			logger.Warn("User forbidden to reverse journal", slog.String("user_id", loggedInUserID), slog.String("journal_id", journalID))
			c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden"})
		} else if errors.Is(err, apperrors.ErrConflict) {
			logger.Warn("Conflict reversing journal (e.g., already reversed)", slog.String("error", err.Error()))
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		} else {
			logger.Error("Failed to reverse journal in service", slog.String("error", err.Error()))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reverse journal"})
		}
		return
	}

	logger.Info("Journal reversed successfully", slog.String("reversing_journal_id", reversingJournal.JournalID))
	c.JSON(http.StatusOK, dto.ToJournalResponse(reversingJournal))
}

package handlers

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/SscSPs/money_managemet_app/internal/apperrors" // Import if needed for DTO conversion
	"github.com/SscSPs/money_managemet_app/internal/core/services"
	"github.com/SscSPs/money_managemet_app/internal/dto"
	"github.com/SscSPs/money_managemet_app/internal/middleware"
	"github.com/gin-gonic/gin"
	// For balance calculation
)

// journalHandler handles HTTP requests related to journals.
type journalHandler struct {
	journalService *services.JournalService
	// We might need AccountService too if we add balance endpoints here
}

// newJournalHandler creates a new journalHandler.
func newJournalHandler(js *services.JournalService) *journalHandler {
	return &journalHandler{
		journalService: js,
	}
}

// registerJournalRoutes registers routes related to journals.
func registerJournalRoutes(rg *gin.RouterGroup, journalService services.JournalService) {
	h := newJournalHandler(&journalService) // Inject service

	journals := rg.Group("/journals")
	{
		journals.POST("", h.persistJournal)
		journals.GET("/:id", h.getJournal)
		// Placeholder for balance endpoint - could be here or under accounts
		// rg.GET("/accounts/:id/balance", h.getAccountBalance) // Needs account service
	}
}

// persistJournal godoc
// @Summary Persist a journal entry with its transactions
// @Description Creates a new journal and associated transactions
// @Tags journals
// @Accept  json
// @Produce  json
// @Param   journal body dto.CreateJournalAndTxn true "Journal and Transactions"
// @Success 200 {object} map[string]string "Returns the ID of the created journal"
// @Failure 400 {object} map[string]string "Invalid request format or validation error"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 500 {object} map[string]string "Failed to persist journal"
// @Security BearerAuth
// @Router /journals [post]
func (h *journalHandler) persistJournal(c *gin.Context) {
	logger := middleware.GetLoggerFromCtx(c.Request.Context())
	var req dto.CreateJournalAndTxn
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn("Failed to bind JSON for PersistJournal", slog.String("error", err.Error()))
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
	logger.Info("Received request to persist journal")

	journal, err := h.journalService.PersistJournal(c.Request.Context(), req, creatorUserID)
	if err != nil {
		// Handle specific validation errors from service
		if errors.Is(err, services.ErrJournalMinEntries) ||
			errors.Is(err, services.ErrCurrencyMismatch) ||
			errors.Is(err, services.ErrAccountNotFound) ||
			errors.Is(err, services.ErrJournalUnbalanced) ||
			errors.Is(err, apperrors.ErrValidation) { // Catch-all validation
			logger.Warn("Validation error persisting journal", slog.String("error", err.Error()))
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		} else {
			logger.Error("Failed to persist journal in service", slog.String("error", err.Error()))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to persist journal"})
		}
		return
	}

	logger.Info("Journal persisted successfully", slog.String("journal_id", journal.JournalID))
	c.JSON(http.StatusOK, gin.H{"journalID": journal.JournalID})
}

// getJournal godoc
// @Summary Get a journal entry and its transactions
// @Description Retrieves a journal and its associated transactions by journal ID
// @Tags journals
// @Produce  json
// @Param   id path string true "Journal ID"
// @Success 200 {object} dto.GetJournalResponse "Journal and its transactions"
// @Failure 404 {object} map[string]string "Journal not found"
// @Failure 500 {object} map[string]string "Failed to retrieve journal"
// @Security BearerAuth
// @Router /journals/{id} [get]
func (h *journalHandler) getJournal(c *gin.Context) {
	logger := middleware.GetLoggerFromCtx(c.Request.Context())
	journalID := c.Param("id")

	logger = logger.With(slog.String("journal_id", journalID))
	logger.Info("Received request to get journal")

	journal, transactions, err := h.journalService.GetJournalWithTransactions(c.Request.Context(), journalID)
	if err != nil {
		// TODO: Distinguish between not found and other errors if service returns specific errors
		logger.Error("Failed to get journal with transactions from service", slog.String("error", err.Error()))
		if errors.Is(err, apperrors.ErrNotFound) { // Assuming service might return this for the journal itself
			c.JSON(http.StatusNotFound, gin.H{"error": "Journal not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve journal"})
		}
		return
	}

	logger.Info("Journal retrieved successfully")
	// Convert to response DTO using the new functions and structs
	resp := dto.GetJournalResponse{
		Journal:      dto.ToJournalResponse(journal),
		Transactions: dto.ToTransactionResponses(transactions),
	}
	c.JSON(http.StatusOK, resp)
}

/* // Placeholder: Balance endpoint likely belongs with Accounts handler
func (h *journalHandler) getAccountBalance(c *gin.Context) {
	logger := middleware.GetLoggerFromCtx(c.Request.Context())
	accountID := c.Param("id")

	balance, err := h.journalService.CalculateAccountBalance(c.Request.Context(), accountID)
	if err != nil {
		if errors.Is(err, services.ErrAccountNotFound) || errors.Is(err, apperrors.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Account not found or inactive"})
		} else if strings.Contains(err.Error(), "inactive") { // Simple check for inactive error message
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		} else {
			logger.Error("Failed to calculate account balance", slog.String("error", err.Error()), slog.String("account_id", accountID))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to calculate balance"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"accountID": accountID, "balance": balance})
}
*/

package handlers

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/SscSPs/money_managemet_app/internal/apperrors"
	"github.com/SscSPs/money_managemet_app/internal/core/services"
	"github.com/SscSPs/money_managemet_app/internal/dto"
	"github.com/SscSPs/money_managemet_app/internal/middleware"
	"github.com/SscSPs/money_managemet_app/internal/models"
	"github.com/SscSPs/money_managemet_app/internal/repositories/database/pgsql"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// journalHandler handles HTTP requests related to journals.
type journalHandler struct {
	journalService *services.JournalService
}

// newJournalHandler creates a new journalHandler.
func newJournalHandler(journalService *services.JournalService) *journalHandler {
	return &journalHandler{
		journalService: journalService,
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
// @Failure 400 {object} map[string]string "Invalid request format"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 500 {object} map[string]string "Failed to persist journal"
// @Router /journals/ [post]
func (h *journalHandler) persistJournal(c *gin.Context) {
	logger := middleware.GetLoggerFromCtx(c.Request.Context())

	createReq := dto.CreateJournalAndTxn{}
	if err := c.ShouldBindJSON(&createReq); err != nil {
		logger.Error("Failed to bind JSON for PersistJournal", slog.String("error", err.Error()))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	creatorUserID, ok := middleware.GetUserIDFromContext(c)
	if !ok {
		logger.Error("Creator user ID not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	logger = logger.With(slog.String("creator_user_id", creatorUserID))

	journal, err := h.journalService.PersistJournal(c.Request.Context(), createReq, creatorUserID)
	if err != nil {
		if errors.Is(err, apperrors.ErrValidation) {
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
// @Accept  json
// @Produce  json
// @Param   journalID path string true "Journal ID"
// @Success 200 {object} dto.CreateJournalAndTxn "Journal and its transactions"
// @Failure 404 {object} map[string]string "Journal not found"
// @Failure 500 {object} map[string]string "Failed to retrieve journal"
// @Router /journals/{journalID} [get]
func (h *journalHandler) getJournal(c *gin.Context) {
	logger := middleware.GetLoggerFromCtx(c.Request.Context())
	journalID := c.Param("journalID")

	journal, txns, err := h.journalService.GetJournalWithTransactions(c.Request.Context(), journalID)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			logger.Warn("Journal not found", slog.String("journal_id", journalID))
			c.JSON(http.StatusNotFound, gin.H{"error": "Journal not found"})
			return
		}
		logger.Error("Failed to get journal from service", slog.String("error", err.Error()), slog.String("journal_id", journalID))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve journal"})
		return
	}

	logger.Debug("Journal retrieved successfully", slog.String("journal_id", journalID))

	modelJournal := models.Journal{
		JournalID:    journal.JournalID,
		JournalDate:  journal.JournalDate,
		Description:  journal.Description,
		CurrencyCode: journal.CurrencyCode,
		Status:       models.JournalStatus(journal.Status),
		AuditFields: models.AuditFields{
			CreatedAt:     journal.CreatedAt,
			CreatedBy:     journal.CreatedBy,
			LastUpdatedAt: journal.LastUpdatedAt,
			LastUpdatedBy: journal.LastUpdatedBy,
		},
	}
	modelTxns := make([]models.Transaction, len(txns))
	for i, dTxn := range txns {
		modelTxns[i] = models.Transaction{
			TransactionID:   dTxn.TransactionID,
			JournalID:       dTxn.JournalID,
			AccountID:       dTxn.AccountID,
			Amount:          dTxn.Amount,
			TransactionType: models.TransactionType(dTxn.TransactionType),
			CurrencyCode:    dTxn.CurrencyCode,
			Notes:           dTxn.Notes,
			AuditFields: models.AuditFields{
				CreatedAt:     dTxn.CreatedAt,
				CreatedBy:     dTxn.CreatedBy,
				LastUpdatedAt: dTxn.LastUpdatedAt,
				LastUpdatedBy: dTxn.LastUpdatedBy,
			},
		}
	}

	c.JSON(http.StatusOK, dto.CreateJournalAndTxn{Journal: modelJournal, Transactions: modelTxns})
}

// registerJournalRoutes registers journal specific routes
func registerJournalRoutes(group *gin.RouterGroup, dbPool *pgxpool.Pool) {
	journalRepo := pgsql.NewPgxJournalRepository(dbPool)
	// Instantiate accountRepo as it's needed by JournalService
	accountRepo := pgsql.NewPgxAccountRepository(dbPool)
	journalService := services.NewJournalService(accountRepo, journalRepo)

	journalHandler := newJournalHandler(journalService)

	journals := group.Group("/journals")
	{
		journals.POST("/", journalHandler.persistJournal)
		journals.GET("/:journalID", journalHandler.getJournal)
	}
}

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

type ledgerHandler struct {
	ledgerService *services.LedgerService
}

func newLedgerHandler(ledgerService *services.LedgerService) *ledgerHandler {
	return &ledgerHandler{
		ledgerService: ledgerService,
	}
}

// persistJournal godoc
// @Summary Persist a journal entry with its transactions
// @Description Creates a new journal and associated transactions
// @Tags ledger
// @Accept  json
// @Produce  json
// @Param   journal body dto.CreateJournalAndTxn true "Journal and Transactions"
// @Success 200 {object} map[string]string "Returns the ID of the created journal"
// @Failure 400 {object} map[string]string "Invalid request format"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 500 {object} map[string]string "Failed to persist journal"
// @Router /ledger/ [post]
func (h *ledgerHandler) persistJournal(c *gin.Context) {
	logger := middleware.GetLoggerFromContext(c)

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

	journal, err := h.ledgerService.PersistJournal(c.Request.Context(), createReq.Journal, createReq.Transactions, creatorUserID)
	if err != nil {
		logger.Error("Failed to persist journal in service", slog.String("error", err.Error()))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to persist journal"})
		return
	}

	logger.Info("Journal persisted successfully", slog.String("journal_id", journal.JournalID))
	c.JSON(http.StatusOK, gin.H{"journalID": journal.JournalID})
}

// getJournal godoc
// @Summary Get a journal entry and its transactions
// @Description Retrieves a journal and its associated transactions by journal ID
// @Tags ledger
// @Accept  json
// @Produce  json
// @Param   journalID path string true "Journal ID"
// @Success 200 {object} dto.CreateJournalAndTxn "Journal and its transactions"
// @Failure 404 {object} map[string]string "Journal not found"
// @Failure 500 {object} map[string]string "Failed to retrieve journal"
// @Router /ledger/{journalID} [get]
func (h *ledgerHandler) getJournal(c *gin.Context) {
	logger := middleware.GetLoggerFromContext(c)
	journalID := c.Param("journalID")

	journal, txns, err := h.ledgerService.GetJournalWithTransactions(c.Request.Context(), journalID)
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
	c.JSON(http.StatusOK, dto.CreateJournalAndTxn{Journal: *journal, Transactions: txns})
}

// registerLedgerRoutes registers ledger specific routes
func registerLedgerRoutes(group *gin.RouterGroup, dbPool *pgxpool.Pool) {
	journalRepo := pgsql.NewJournalRepository(dbPool)
	accountRepo := pgsql.NewAccountRepository(dbPool)
	ledgerService := services.NewLedgerService(accountRepo, journalRepo)

	ledgerHandler := newLedgerHandler(ledgerService)

	ledger := group.Group("/ledger")
	{
		ledger.POST("/", ledgerHandler.persistJournal)
		ledger.GET("/:journalID", ledgerHandler.getJournal)
	}
}

package handlers

import (
	"net/http"

	"github.com/SscSPs/money_managemet_app/internal/core/services"
	"github.com/SscSPs/money_managemet_app/internal/dto"
	"github.com/gin-gonic/gin"
)

type LedgerHandler struct {
	ledgerService *services.LedgerService
}

func NewLedgerHandler(ledgerService *services.LedgerService) *LedgerHandler {
	return &LedgerHandler{ledgerService: ledgerService}
}

// PersistJournal godoc
// @Summary Persist a journal entry with its transactions
// @Description Creates a new journal and associated transactions
// @Tags ledger
// @Accept  json
// @Produce  json
// @Param   journal body dto.CreateJournalAndTxn true "Journal and Transactions"
// @Success 200 {object} string
// @Failure 500 {object} string
// @Router /ledger/ [post]
func (h *LedgerHandler) PersistJournal(c *gin.Context) {
	createReq := dto.CreateJournalAndTxn{}
	c.ShouldBindJSON(&createReq)
	journal, err := h.ledgerService.PersistJournal(c.Request.Context(), createReq.Journal, createReq.Transactions, createReq.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"journalID": journal.JournalID})
}

// GetJournal godoc
// @Summary Get a journal entry and its transactions
// @Description Retrieves a journal and its associated transactions by journal ID
// @Tags ledger
// @Accept  json
// @Produce  json
// @Param   journalID path string true "Journal ID"
// @Success 200 {object} dto.CreateJournalAndTxn
// @Failure 500 {object} string
// @Router /ledger/{journalID} [get]
func (h *LedgerHandler) GetJournal(c *gin.Context) {
	journalID := c.Param("journalID")
	journal, txns, err := h.ledgerService.GetJournalWithTransactions(c.Request.Context(), journalID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, dto.CreateJournalAndTxn{Journal: *journal, Transactions: txns})
}

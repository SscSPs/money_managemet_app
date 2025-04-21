package handlers

import (
	"net/http"

	"github.com/SscSPs/money_managemet_app/internal/core/services"
	"github.com/SscSPs/money_managemet_app/internal/dto"
	"github.com/gin-gonic/gin"
	// TODO: Add logging import (e.g., "log/slog")
)

type AccountHandler struct {
	accountService *services.AccountService
	// TODO: Inject logger
}

func NewAccountHandler(accountService *services.AccountService) *AccountHandler {
	return &AccountHandler{accountService: accountService}
}

// CreateAccount godoc
// @Summary Create a new financial account
// @Description Creates a new account (Asset, Liability, Equity, Income, Expense)
// @Tags accounts
// @Accept  json
// @Produce  json
// @Param   account body dto.CreateAccountRequest true "Account details"
// @Success 201 {object} dto.AccountResponse
// @Failure 400 {object} string "Invalid input"
// @Failure 500 {object} string "Internal server error"
// @Router /accounts [post]
func (h *AccountHandler) CreateAccount(c *gin.Context) {
	var createReq dto.CreateAccountRequest
	if err := c.ShouldBindJSON(&createReq); err != nil {
		// TODO: Add structured logging
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input: " + err.Error()})
		return
	}

	userID := createReq.UserID

	account, err := h.accountService.CreateAccount(c.Request.Context(), createReq, userID)
	if err != nil {
		// TODO: Add structured logging
		// TODO: Differentiate between validation errors (4xx) and server errors (5xx) if service returns specific errors
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create account: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, dto.ToAccountResponse(account))
}

// GetAccount godoc
// @Summary Get an account by ID
// @Description Retrieves details for a specific account by its ID
// @Tags accounts
// @Accept  json
// @Produce  json
// @Param   accountID path string true "Account ID"
// @Success 200 {object} dto.AccountResponse
// @Failure 404 {object} string "Account not found"
// @Failure 500 {object} string "Internal server error"
// @Router /accounts/{accountID} [get]
func (h *AccountHandler) GetAccount(c *gin.Context) {
	accountID := c.Param("accountID")

	account, err := h.accountService.GetAccountByID(c.Request.Context(), accountID)
	if err != nil {
		// TODO: Add structured logging
		// TODO: Check if error is a specific "not found" error and return 404
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get account: " + err.Error()})
		return
	}

	if account == nil { // Should ideally be handled by error checking above
		c.JSON(http.StatusNotFound, gin.H{"error": "Account not found"})
		return
	}

	c.JSON(http.StatusOK, dto.ToAccountResponse(account))
}

// TODO: Add ListAccounts handler later
// TODO: Add UpdateAccount handler later
// TODO: Add DeleteAccount (or Deactivate) handler later

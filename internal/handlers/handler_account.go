package handlers

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/SscSPs/money_managemet_app/internal/apperrors"
	"github.com/SscSPs/money_managemet_app/internal/core/services"
	"github.com/SscSPs/money_managemet_app/internal/dto"
	"github.com/SscSPs/money_managemet_app/internal/middleware"
	"github.com/gin-gonic/gin"
	// TODO: Add logging import (e.g., "log/slog")
)

type AccountHandler struct {
	accountService *services.AccountService
	// logger         *slog.Logger // Removed
}

func NewAccountHandler(accountService *services.AccountService) *AccountHandler {
	return &AccountHandler{
		accountService: accountService,
	}
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
	logger := middleware.GetLoggerFromContext(c)

	var createReq dto.CreateAccountRequest
	if err := c.ShouldBindJSON(&createReq); err != nil {
		logger.Error("Failed to bind JSON for CreateAccount", slog.String("error", err.Error()))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	// Get the ID of the user *performing* the action from the context
	creatorUserID, ok := middleware.GetUserIDFromContext(c)
	if !ok {
		logger.Error("Creator user ID not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Log the creator performing the action
	logger = logger.With(slog.String("creator_user_id", creatorUserID))

	// Note: createReq.UserID likely refers to the owner of the account, which might
	// differ from the creatorUserID if an admin is creating an account for someone else.
	// The service layer needs to handle this distinction based on business logic.
	account, err := h.accountService.CreateAccount(c.Request.Context(), createReq, creatorUserID)
	if err != nil {
		// TODO: Differentiate between validation errors (4xx) and server errors (5xx) if service returns specific errors
		logger.Error("Failed to create account in service", slog.String("error", err.Error()), slog.String("requested_name", createReq.Name), slog.String("account_owner_id", createReq.UserID))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create account"})
		return
	}
	logger.Info("Account created successfully", slog.String("account_id", account.AccountID))
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
	logger := middleware.GetLoggerFromContext(c) // Get logger from context
	accountID := c.Param("accountID")

	account, err := h.accountService.GetAccountByID(c.Request.Context(), accountID)
	if err != nil {
		// Check for specific ErrNotFound
		if errors.Is(err, apperrors.ErrNotFound) {
			logger.Warn("Account not found", slog.String("account_id", accountID))
			c.JSON(http.StatusNotFound, gin.H{"error": "Account not found"})
			return
		}
		// Log other errors
		logger.Error("Failed to get account from service", slog.String("error", err.Error()), slog.String("account_id", accountID))
		// Use generic error message
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve account"})
		return
	}

	// No need for explicit nil check if service returns ErrNotFound

	logger.Debug("Account retrieved successfully", slog.String("account_id", account.AccountID))
	c.JSON(http.StatusOK, dto.ToAccountResponse(account))
}

// TODO: Add ListAccounts handler later
// TODO: Add UpdateAccount handler later
// TODO: Add DeleteAccount (or Deactivate) handler later

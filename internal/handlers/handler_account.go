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

type accountHandler struct {
	accountService *services.AccountService
}

func newAccountHandler(accountService *services.AccountService) *accountHandler {
	return &accountHandler{
		accountService: accountService,
	}
}

// createAccount godoc
// @Summary Create a new financial account
// @Description Creates a new account (Asset, Liability, Equity, Income, Expense)
// @Tags accounts
// @Accept  json
// @Produce  json
// @Param   account body dto.CreateAccountRequest true "Account details"
// @Success 201 {object} dto.AccountResponse "The created account"
// @Failure 400 {object} map[string]string "Invalid request format"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /accounts [post]
func (h *accountHandler) createAccount(c *gin.Context) {
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

	// Call the service to create the account
	account, err := h.accountService.CreateAccount(c.Request.Context(), createReq, creatorUserID)
	if err != nil {
		// TODO: Differentiate between validation errors (4xx) and server errors (5xx) if service returns specific errors
		logger.Error("Failed to create account in service", slog.String("error", err.Error()), slog.String("requested_name", createReq.Name), slog.String("account_owner_id", createReq.UserID))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create account"})
		return
	}
	logger.Info("Account created successfully", slog.String("account_id", account.AccountID))
	c.JSON(http.StatusCreated, dto.ToAccountResponse(account)) // Use DTO for response
}

// getAccount godoc
// @Summary Get an account by ID
// @Description Retrieves details for a specific account by its ID
// @Tags accounts
// @Accept  json
// @Produce  json
// @Param   accountID path string true "Account ID"
// @Success 200 {object} dto.AccountResponse "The requested account"
// @Failure 404 {object} map[string]string "Account not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /accounts/{accountID} [get]
func (h *accountHandler) getAccount(c *gin.Context) {
	logger := middleware.GetLoggerFromContext(c)
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

	logger.Debug("Account retrieved successfully", slog.String("account_id", account.AccountID))
	c.JSON(http.StatusOK, dto.ToAccountResponse(account)) // Use DTO for response
}

// registerAccountRoutes registers account specific routes
func registerAccountRoutes(group *gin.RouterGroup, dbPool *pgxpool.Pool) {
	// Instantiate dependencies
	accountRepo := pgsql.NewAccountRepository(dbPool)
	accountService := services.NewAccountService(accountRepo)
	accountHandler := newAccountHandler(accountService)

	// Define routes
	accounts := group.Group("/accounts")
	{
		accounts.POST("/", accountHandler.createAccount)
		accounts.GET("/:accountID", accountHandler.getAccount)
	}
}

// TODO: Add ListAccounts handler later
// TODO: Add UpdateAccount handler later
// TODO: Add DeleteAccount (or Deactivate) handler later

package handlers

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/SscSPs/money_managemet_app/internal/apperrors"
	"github.com/SscSPs/money_managemet_app/internal/core/services"
	"github.com/SscSPs/money_managemet_app/internal/dto"
	"github.com/SscSPs/money_managemet_app/internal/middleware"
	"github.com/SscSPs/money_managemet_app/internal/repositories/database/pgsql"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AccountHandler struct {
	accountService *services.AccountService
	journalService *services.JournalService
}

func NewAccountHandler(accountService *services.AccountService, journalService *services.JournalService) *AccountHandler {
	return &AccountHandler{
		accountService: accountService,
		journalService: journalService,
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
func (h *AccountHandler) createAccount(c *gin.Context) {
	logger := middleware.GetLoggerFromCtx(c.Request.Context())

	var createReq dto.CreateAccountRequest
	if err := c.ShouldBindJSON(&createReq); err != nil {
		logger.Error("Failed to bind JSON for CreateAccount", slog.String("error", err.Error()))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	creatorUserID, ok := middleware.GetUserIDFromContext(c)
	if !ok {
		logger.Error("Creator user ID not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	account, err := h.accountService.CreateAccount(c.Request.Context(), createReq, creatorUserID)
	if err != nil {
		// Check for specific errors from the service
		if errors.Is(err, apperrors.ErrValidation) {
			logger.Warn("Validation error creating account", slog.String("error", err.Error()))
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		} else {
			// Log unexpected errors
			logger.Error("Failed to create account in service", slog.String("error", err.Error()), slog.String("requested_name", createReq.Name), slog.String("account_owner_id", createReq.UserID))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create account"})
		}
		return
	}
	logger.Info("Account created successfully", slog.String("account_id", account.AccountID))
	c.JSON(http.StatusCreated, dto.ToAccountResponse(account))
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
func (h *AccountHandler) getAccount(c *gin.Context) {
	logger := middleware.GetLoggerFromCtx(c.Request.Context())
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

// getAccountBalance godoc
// @Summary Get account balance
// @Description Retrieves the current calculated balance for a specific account
// @Tags accounts
// @Produce json
// @Param accountID path string true "Account ID"
// @Success 200 {object} dto.AccountBalanceResponse
// @Failure 404 {object} map[string]string "Account not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /accounts/{accountID}/balance [get]
func (h *AccountHandler) getAccountBalance(c *gin.Context) {
	logger := middleware.GetLoggerFromCtx(c.Request.Context())
	accountID := c.Param("accountID")

	balance, err := h.journalService.CalculateAccountBalance(c.Request.Context(), accountID)
	if err != nil {
		// Check for specific ErrNotFound (assuming service returns this)
		if errors.Is(err, apperrors.ErrNotFound) {
			logger.Warn("Account not found when calculating balance", slog.String("account_id", accountID))
			c.JSON(http.StatusNotFound, gin.H{"error": "Account not found"})
		} else {
			// Log other errors from calculation or repository
			logger.Error("Failed to calculate account balance", slog.String("error", err.Error()), slog.String("account_id", accountID))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to calculate account balance"})
		}
		return
	}

	logger.Info("Account balance retrieved successfully")
	// Return the balance
	resp := dto.AccountBalanceResponse{
		AccountID: accountID,
		Balance:   balance,
	}
	c.JSON(http.StatusOK, resp)
}

// listAccounts godoc
// @Summary List accounts
// @Description Retrieves a paginated list of active financial accounts
// @Tags accounts
// @Accept  json
// @Produce  json
// @Param   limit query int false "Limit number of results" default(20)
// @Param   offset query int false "Offset for pagination" default(0)
// @Success 200 {object} dto.ListAccountsResponse
// @Failure 400 {object} map[string]string "Invalid query parameters"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /accounts [get]
func (h *AccountHandler) listAccounts(c *gin.Context) {
	logger := middleware.GetLoggerFromCtx(c.Request.Context())

	var params dto.ListAccountsParams
	if err := c.ShouldBindQuery(&params); err != nil {
		logger.Warn("Failed to bind query parameters for ListAccounts", slog.String("error", err.Error()))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid query parameters: " + err.Error()})
		return
	}

	accounts, err := h.accountService.ListAccounts(c.Request.Context(), params.Limit, params.Offset)
	if err != nil {
		logger.Error("Failed to list accounts from service", slog.String("error", err.Error()))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve accounts"})
		return
	}

	resp := dto.ListAccountsResponse{
		Accounts: dto.ToListAccountResponse(accounts), // Use existing DTO converter
	}

	logger.Debug("Accounts listed successfully")
	c.JSON(http.StatusOK, resp)
}

// deactivateAccount godoc
// @Summary Deactivate an account
// @Description Marks a financial account as inactive
// @Tags accounts
// @Accept  json
// @Produce  json
// @Param   accountID path string true "Account ID to deactivate"
// @Success 204 "No Content"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 404 {object} map[string]string "Account not found"
// @Failure 400 {object} map[string]string "Account already inactive or other validation error"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /accounts/{accountID} [delete]
func (h *AccountHandler) deactivateAccount(c *gin.Context) {
	logger := middleware.GetLoggerFromCtx(c.Request.Context())
	accountID := c.Param("accountID")

	userID, ok := middleware.GetUserIDFromContext(c)
	if !ok {
		logger.Error("User ID not found in context for DeactivateAccount")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// TODO: Add authorization check if only specific users can deactivate specific accounts

	err := h.accountService.DeactivateAccount(c.Request.Context(), accountID, userID)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			logger.Warn("Account not found for deactivation", slog.String("account_id", accountID))
			c.JSON(http.StatusNotFound, gin.H{"error": "Account not found"})
		} else if errors.Is(err, apperrors.ErrValidation) {
			// This likely means the account was already inactive
			logger.Warn("Validation error deactivating account (likely already inactive)", slog.String("account_id", accountID), slog.String("error", err.Error()))
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		} else {
			logger.Error("Failed to deactivate account in service", slog.String("error", err.Error()), slog.String("account_id", accountID))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to deactivate account"})
		}
		return
	}

	logger.Info("Account deactivated successfully", slog.String("account_id", accountID))
	c.Status(http.StatusNoContent)
}

// registerAccountRoutes registers account specific routes
func registerAccountRoutes(group *gin.RouterGroup, dbPool *pgxpool.Pool) {
	// Instantiate dependencies
	accountRepo := pgsql.NewPgxAccountRepository(dbPool)
	journalRepo := pgsql.NewPgxJournalRepository(dbPool)
	accountService := services.NewAccountService(accountRepo)
	journalService := services.NewJournalService(accountRepo, journalRepo)
	accountHandler := NewAccountHandler(accountService, journalService)

	// Define routes
	accounts := group.Group("/accounts")
	{
		accounts.POST("/", accountHandler.createAccount)
		accounts.GET("/", accountHandler.listAccounts)
		accounts.GET("/:accountID", accountHandler.getAccount)
		accounts.GET("/:accountID/balance", accountHandler.getAccountBalance)
		accounts.DELETE("/:accountID", accountHandler.deactivateAccount)
	}
}

// TODO: Add UpdateAccount handler later
// TODO: Add DeleteAccount (or Deactivate) handler later

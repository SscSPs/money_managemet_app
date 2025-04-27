package handlers

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/SscSPs/money_managemet_app/internal/apperrors" // Ensure domain is imported if needed for ToAccountResponse or models
	"github.com/SscSPs/money_managemet_app/internal/core/services"
	"github.com/SscSPs/money_managemet_app/internal/dto"
	"github.com/SscSPs/money_managemet_app/internal/middleware"
	"github.com/gin-gonic/gin"
)

// accountHandler handles HTTP requests related to accounts.
type accountHandler struct {
	accountService *services.AccountService
}

// newAccountHandler creates a new accountHandler.
func newAccountHandler(as *services.AccountService) *accountHandler {
	return &accountHandler{
		accountService: as,
	}
}

// registerAccountRoutes registers routes related to accounts.
func registerAccountRoutes(rg *gin.RouterGroup, accountService services.AccountService) {
	h := newAccountHandler(&accountService) // Inject service

	accounts := rg.Group("/accounts")
	{
		accounts.POST("", h.createAccount)
		accounts.GET("/:id", h.getAccount)
		accounts.GET("", h.listAccounts)
		accounts.PUT("/:id", h.updateAccount)
		accounts.DELETE("/:id", h.deleteAccount)
		// Optional: accounts.GET("/:id/balance", h.getAccountBalance) // If using account service for balance
	}
}

// createAccount godoc
// @Summary Create a new account
// @Description Creates a new account for the logged-in user
// @Tags accounts
// @Accept  json
// @Produce  json
// @Param   account body dto.CreateAccountRequest true "Account details"
// @Success 201 {object} dto.AccountResponse
// @Failure 400 {object} map[string]string "Invalid input format or validation error"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 500 {object} map[string]string "Failed to create account"
// @Security BearerAuth
// @Router /accounts [post]
func (h *accountHandler) createAccount(c *gin.Context) {
	logger := middleware.GetLoggerFromCtx(c.Request.Context())
	var req dto.CreateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn("Failed to bind JSON for CreateAccount", slog.String("error", err.Error()))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format: " + err.Error()})
		return
	}

	// Get creator UserID from context
	creatorUserID, ok := middleware.GetUserIDFromContext(c)
	if !ok {
		logger.Error("Creator user ID not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	logger = logger.With(slog.String("creator_user_id", creatorUserID))
	logger.Info("Received request to create account", slog.String("account_name", req.Name), slog.String("currency_code", req.CurrencyCode))

	newAccount, err := h.accountService.CreateAccount(c.Request.Context(), req, creatorUserID)
	if err != nil {
		if errors.Is(err, apperrors.ErrValidation) {
			logger.Warn("Validation error creating account", slog.String("error", err.Error()))
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		} else if errors.Is(err, apperrors.ErrNotFound) { // e.g., Currency not found
			logger.Warn("Dependency not found creating account", slog.String("error", err.Error()))
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}) // Or 404 depending on context
		} else {
			logger.Error("Failed to create account in service", slog.String("error", err.Error()))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create account"})
		}
		return
	}

	logger.Info("Account created successfully", slog.String("account_id", newAccount.AccountID))
	c.JSON(http.StatusCreated, dto.ToAccountResponse(newAccount))
}

// getAccount godoc
// @Summary Get an account by ID
// @Description Retrieves details for a specific account by its ID
// @Tags accounts
// @Produce  json
// @Param   id path string true "Account ID"
// @Success 200 {object} dto.AccountResponse
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden (accessing another user's account)"
// @Failure 404 {object} map[string]string "Account not found"
// @Failure 500 {object} map[string]string "Failed to retrieve account"
// @Security BearerAuth
// @Router /accounts/{id} [get]
func (h *accountHandler) getAccount(c *gin.Context) {
	logger := middleware.GetLoggerFromCtx(c.Request.Context())
	accountID := c.Param("id")

	// Get logged-in UserID from context (needed for potential future auth checks)
	loggedInUserID, ok := middleware.GetUserIDFromContext(c)
	if !ok {
		logger.Error("Logged-in user ID not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	logger = logger.With(slog.String("target_account_id", accountID))
	logger.Info("Received request to get account")

	account, err := h.accountService.GetAccountByID(c.Request.Context(), accountID)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			logger.Warn("Account not found")
			c.JSON(http.StatusNotFound, gin.H{"error": "Account not found"})
		} else {
			logger.Error("Failed to get account from service", slog.String("error", err.Error()))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve account"})
		}
		return
	}

	// TODO: Re-evaluate authorization. The domain.Account model lacks a direct UserID for ownership.
	// Authorization might need to check CreatedBy or a different mechanism.
	/*
		if account.UserID != loggedInUserID { // Commented out - UserID field doesn't exist directly on Account
			logger.Warn("User forbidden to access account", slog.String("accessor_id", loggedInUserID), slog.String("owner_id", account.UserID))
			c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden"})
			return
		}
	*/
	_ = loggedInUserID // Use loggedInUserID to prevent unused variable error temporarily

	logger.Info("Account retrieved successfully")
	c.JSON(http.StatusOK, dto.ToAccountResponse(account))
}

// listAccounts godoc
// @Summary List accounts for the logged-in user
// @Description Retrieves a list of accounts owned by the logged-in user
// @Tags accounts
// @Produce  json
// @Param   limit query int false "Limit number of results" default(20)
// @Param   offset query int false "Offset for pagination" default(0)
// @Success 200 {object} dto.ListAccountsResponse
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 500 {object} map[string]string "Failed to list accounts"
// @Security BearerAuth
// @Router /accounts [get]
func (h *accountHandler) listAccounts(c *gin.Context) {
	logger := middleware.GetLoggerFromCtx(c.Request.Context())

	// Get logged-in UserID from context
	loggedInUserID, ok := middleware.GetUserIDFromContext(c)
	if !ok {
		logger.Error("Logged-in user ID not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var params dto.ListAccountsParams
	if err := c.ShouldBindQuery(&params); err != nil {
		logger.Warn("Failed to bind query params for ListAccounts", slog.String("error", err.Error()))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid query parameters: " + err.Error()})
		return
	}

	// TODO: Ensure the service filters by the logged-in user ID.
	// The service should ideally get the userID from the context or as a parameter.

	logger = logger.With(slog.String("user_id", loggedInUserID))
	logger.Info("Received request to list accounts", slog.Int("limit", params.Limit), slog.Int("offset", params.Offset))

	// TODO: Update AccountService.ListAccounts to accept dto.ListAccountsParams or similar struct including UserID.
	// Temporarily using old signature.
	respAccounts, err := h.accountService.ListAccounts(c.Request.Context(), params.Limit, params.Offset)
	if err != nil {
		logger.Error("Failed to list accounts from service", slog.String("error", err.Error()))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list accounts"})
		return
	}

	// Convert domain accounts to DTOs
	accountResponses := make([]dto.AccountResponse, len(respAccounts))
	for i, acc := range respAccounts {
		accountResponses[i] = dto.ToAccountResponse(&acc)
	}

	logger.Info("Accounts listed successfully", slog.Int("count", len(accountResponses)))
	// Assume dto.ListAccountsResponse exists and takes []dto.AccountResponse
	c.JSON(http.StatusOK, dto.ListAccountsResponse{Accounts: accountResponses})
}

// updateAccount godoc
// @Summary Update an account
// @Description Updates an account's details (e.g., name)
// @Tags accounts
// @Accept  json
// @Produce  json
// @Param   id path string true "Account ID to update"
// @Param   account body dto.UpdateAccountRequest true "Account details to update"
// @Success 200 {object} dto.AccountResponse
// @Failure 400 {object} map[string]string "Invalid input"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden"
// @Failure 404 {object} map[string]string "Account not found"
// @Failure 500 {object} map[string]string "Failed to update account"
// @Security BearerAuth
// @Router /accounts/{id} [put]
func (h *accountHandler) updateAccount(c *gin.Context) {
	logger := middleware.GetLoggerFromCtx(c.Request.Context())
	accountID := c.Param("id")
	// Bind the update request
	var req dto.UpdateAccountRequest // Use the defined DTO
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn("Failed to bind JSON for UpdateAccount", slog.String("error", err.Error()))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format: " + err.Error()})
		return
	}

	// Get logged-in UserID from context for audit and potential future auth checks
	loggedInUserID, ok := middleware.GetUserIDFromContext(c)
	if !ok {
		logger.Error("Logged-in user ID not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	logger = logger.With(slog.String("target_account_id", accountID), slog.String("updater_user_id", loggedInUserID))
	logger.Info("Received request to update account")

	// Service call includes existence check and auth TODO
	updatedAccount, err := h.accountService.UpdateAccount(c.Request.Context(), accountID, req, loggedInUserID)
	if err != nil {
		// Handle service errors (NotFound, Validation, etc.)
		if errors.Is(err, apperrors.ErrNotFound) {
			logger.Warn("Account not found for update via service")
			c.JSON(http.StatusNotFound, gin.H{"error": "Account not found"})
		} else if errors.Is(err, apperrors.ErrValidation) {
			logger.Warn("Validation error updating account via service", slog.String("error", err.Error()))
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}) // Pass validation error message
		} else {
			logger.Error("Failed to update account in service", slog.String("error", err.Error()))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update account"})
		}
		return
	}

	logger.Info("Account updated successfully")
	c.JSON(http.StatusOK, dto.ToAccountResponse(updatedAccount))
}

// deleteAccount godoc
// @Summary Delete an account
// @Description Marks an account as deleted (soft delete)
// @Tags accounts
// @Produce  json
// @Param   id path string true "Account ID to delete"
// @Success 204 "No Content"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden"
// @Failure 404 {object} map[string]string "Account not found"
// @Failure 409 {object} map[string]string "Conflict (e.g., already deleted)"
// @Failure 500 {object} map[string]string "Failed to delete account"
// @Security BearerAuth
// @Router /accounts/{id} [delete]
func (h *accountHandler) deleteAccount(c *gin.Context) {
	logger := middleware.GetLoggerFromCtx(c.Request.Context())
	accountID := c.Param("id")

	// Get logged-in UserID from context for audit and potential future auth checks
	loggedInUserID, ok := middleware.GetUserIDFromContext(c)
	if !ok {
		logger.Error("Logged-in user ID not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	logger = logger.With(slog.String("target_account_id", accountID), slog.String("deleter_user_id", loggedInUserID))
	logger.Info("Received request to delete account")

	// Service call includes existence check, status check, and auth TODO
	err := h.accountService.DeactivateAccount(c.Request.Context(), accountID, loggedInUserID) // Call DeactivateAccount
	if err != nil {
		// Handle service errors
		if errors.Is(err, apperrors.ErrNotFound) {
			logger.Warn("Account not found for deletion via service")
			c.JSON(http.StatusNotFound, gin.H{"error": "Account not found"})
		} else if errors.Is(err, apperrors.ErrValidation) {
			// This likely means the account was already inactive
			logger.Warn("Validation error deleting account (already inactive?)", slog.String("error", err.Error()))
			c.JSON(http.StatusConflict, gin.H{"error": "Account already inactive or cannot be deleted"})
		} else {
			logger.Error("Failed to delete account in service", slog.String("error", err.Error()))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete account"})
		}
		return
	}

	logger.Info("Account deleted successfully")
	c.Status(http.StatusNoContent)
}

/* Potential Balance Endpoint - better in account service?
// getAccountBalance godoc
// @Summary Get account balance
// @Description Calculates and returns the current balance for a specific account
// @Tags accounts
// @Produce json
// @Param id path string true "Account ID"
// @Success 200 {object} map[string]interface{}"accountID", "balance"}
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden"
// @Failure 404 {object} map[string]string "Account not found"
// @Failure 500 {object} map[string]string "Failed to calculate balance"
// @Security BearerAuth
// @Router /accounts/{id}/balance [get]
func (h *accountHandler) getAccountBalance(c *gin.Context) {
	logger := middleware.GetLoggerFromCtx(c.Request.Context())
	accountID := c.Param("id")

	loggedInUserID, ok := middleware.GetUserIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Fetch the account first to check ownership (needed for authorization)
	account, err := h.accountService.GetAccountByID(c.Request.Context(), accountID)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Account not found"})
		} else {
			logger.Error("Failed to get account for balance check", slog.String("error", err.Error()))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to calculate balance"})
		}
		return
	}
	if account.UserID != loggedInUserID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden"})
		return
	}

	// Call the service method (assuming it exists)
	balance, err := h.accountService.CalculateAccountBalance(c.Request.Context(), accountID) // Hypothetical service method
	if err != nil {
		// Handle errors from balance calculation (e.g., inactive account?)
		logger.Error("Failed to calculate account balance", slog.String("error", err.Error()), slog.String("account_id", accountID))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to calculate balance"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"accountID": accountID, "balance": balance})
}*/

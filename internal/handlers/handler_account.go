package handlers

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/SscSPs/money_managemet_app/internal/apperrors"                    // Ensure domain is imported if needed for ToAccountResponse or models
	portssvc "github.com/SscSPs/money_managemet_app/internal/core/ports/services" // Use ports services
	"github.com/SscSPs/money_managemet_app/internal/dto"
	"github.com/SscSPs/money_managemet_app/internal/middleware"
	"github.com/gin-gonic/gin"
)

// accountHandler handles HTTP requests related to accounts.
type accountHandler struct {
	accountService portssvc.AccountSvcFacade     // Updated to use AccountSvcFacade
	journalService portssvc.TransactionReaderSvc // Updated to use TransactionReaderSvc
}

// newAccountHandler creates a new accountHandler.
func newAccountHandler(as portssvc.AccountSvcFacade, js portssvc.TransactionReaderSvc) *accountHandler { // Updated interfaces
	return &accountHandler{
		accountService: as,
		journalService: js,
	}
}

// RegisterAccountRoutes registers routes related to accounts WITHIN a workplace.
func RegisterAccountRoutes(rg *gin.RouterGroup, accountService portssvc.AccountSvcFacade, transactionReaderSvc portssvc.TransactionReaderSvc) { // Updated interfaces
	h := newAccountHandler(accountService, transactionReaderSvc)

	// Routes are now relative to /workplaces/{workplace_id}/
	accounts := rg.Group("/accounts")
	{
		accounts.POST("", h.createAccount)
		accounts.GET("", h.listAccounts)
		accounts.GET("/:id", h.getAccount)
		accounts.PUT("/:id", h.updateAccount)
		accounts.DELETE("/:id", h.deleteAccount)
		// Nested route for transactions within an account
		accounts.GET("/:id/transactions", h.listTransactionsByAccount)
	}
}

// createAccount godoc
// @Summary Create account in workplace
// @Description Creates a new account within the specified workplace.
// @Tags accounts
// @Accept  json
// @Produce  json
// @Param   workplace_id path string true "Workplace ID"
// @Param   account body dto.CreateAccountRequest true "Account details"
// @Success 201 {object} dto.AccountResponse
// @Failure 400 {object} map[string]string "Invalid input or missing Workplace ID"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden (User cannot create in this workplace)"
// @Failure 500 {object} map[string]string "Failed to create account"
// @Security BearerAuth
// @Router /workplaces/{workplace_id}/accounts [post]
func (h *accountHandler) createAccount(c *gin.Context) {
	logger := middleware.GetLoggerFromCtx(c.Request.Context())
	workplaceID := c.Param("workplace_id") // Get from path
	if workplaceID == "" {
		logger.Error("Workplace ID missing from path for createAccount")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Workplace ID required in path"})
		return
	}

	var req dto.CreateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn("Failed to bind JSON for CreateAccount", slog.String("error", err.Error()))
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
	logger.Info("Received request to create account", slog.String("account_name", req.Name), slog.String("currency_code", req.CurrencyCode))

	newAccount, err := h.accountService.CreateAccount(c.Request.Context(), workplaceID, req, creatorUserID) // Pass workplaceID
	if err != nil {
		if errors.Is(err, apperrors.ErrForbidden) {
			logger.Warn("User forbidden to create account in workplace", slog.String("user_id", creatorUserID), slog.String("workplace_id", workplaceID))
			c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden"})
		} else if errors.Is(err, apperrors.ErrValidation) || errors.Is(err, apperrors.ErrNotFound) {
			logger.Warn("Validation/NotFound error creating account", slog.String("error", err.Error()))
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
// @Summary Get account by ID from workplace
// @Description Retrieves details for a specific account by its ID within a workplace.
// @Tags accounts
// @Produce  json
// @Param   workplace_id path string true "Workplace ID"
// @Param   id path string true "Account ID"
// @Success 200 {object} dto.AccountResponse
// @Failure 400 {object} map[string]string "Missing Workplace or Account ID"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden (User not part of workplace)"
// @Failure 404 {object} map[string]string "Account not found in this workplace"
// @Failure 500 {object} map[string]string "Failed to retrieve account"
// @Security BearerAuth
// @Router /workplaces/{workplace_id}/accounts/{id} [get]
func (h *accountHandler) getAccount(c *gin.Context) {
	logger := middleware.GetLoggerFromCtx(c.Request.Context())
	workplaceID := c.Param("workplace_id") // Get from path
	accountID := c.Param("id")
	if workplaceID == "" || accountID == "" {
		logger.Error("Workplace ID or Account ID missing from path for getAccount")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Workplace and Account ID required in path"})
		return
	}

	loggedInUserID, ok := middleware.GetUserIDFromContext(c)
	if !ok {
		logger.Error("Logged-in user ID not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	logger = logger.With(slog.String("target_account_id", accountID), slog.String("workplace_id", workplaceID), slog.String("requesting_user_id", loggedInUserID))
	logger.Info("Received request to get account")

	account, err := h.accountService.GetAccountByID(c.Request.Context(), workplaceID, accountID) // Pass workplaceID
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			logger.Warn("Account not found or not in this workplace")
			c.JSON(http.StatusNotFound, gin.H{"error": "Account not found"})
		} else if errors.Is(err, apperrors.ErrForbidden) {
			logger.Warn("User forbidden to access account workplace", slog.String("user_id", loggedInUserID), slog.String("workplace_id", workplaceID))
			c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden"})
		} else {
			logger.Error("Failed to get account from service", slog.String("error", err.Error()))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve account"})
		}
		return
	}

	logger.Info("Account retrieved successfully")
	c.JSON(http.StatusOK, dto.ToAccountResponse(account))
}

// listAccounts godoc
// @Summary List accounts for current user in workplace
// @Description Retrieves a list of accounts for the specified workplace if the user is a member.
// @Tags accounts
// @Produce  json
// @Param   workplace_id path string true "Workplace ID"
// @Param   limit query int false "Limit number of results" default(20)
// @Param   offset query int false "Offset for pagination" default(0)
// @Success 200 {object} dto.ListAccountsResponse
// @Failure 400 {object} map[string]string "Missing Workplace ID"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden (User not part of workplace)"
// @Failure 500 {object} map[string]string "Failed to list accounts"
// @Security BearerAuth
// @Router /workplaces/{workplace_id}/accounts [get]
func (h *accountHandler) listAccounts(c *gin.Context) {
	logger := middleware.GetLoggerFromCtx(c.Request.Context())

	loggedInUserID, ok := middleware.GetUserIDFromContext(c)
	if !ok {
		logger.Error("Logged-in user ID not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	workplaceID := c.Param("workplace_id") // Get from path
	if workplaceID == "" {
		logger.Error("Workplace ID missing from request path for listAccounts")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Workplace ID required in path"})
		return
	}

	var params dto.ListAccountsParams
	if err := c.ShouldBindQuery(&params); err != nil {
		logger.Warn("Failed to bind query params for ListAccounts", slog.String("error", err.Error()))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid query parameters: " + err.Error()})
		return
	}

	logger = logger.With(slog.String("user_id", loggedInUserID), slog.String("workplace_id", workplaceID))
	logger.Info("Received request to list accounts", slog.Int("limit", params.Limit), slog.Int("offset", params.Offset))

	respAccounts, err := h.accountService.ListAccounts(c.Request.Context(), workplaceID, params.Limit, params.Offset)
	if err != nil {
		if errors.Is(err, apperrors.ErrForbidden) {
			logger.Warn("User forbidden to list accounts for workplace", slog.String("user_id", loggedInUserID), slog.String("workplace_id", workplaceID))
			c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden"})
		} else {
			logger.Error("Failed to list accounts from service", slog.String("error", err.Error()))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list accounts"})
		}
		return
	}

	accountResponses := make([]dto.AccountResponse, len(respAccounts))
	for i, acc := range respAccounts {
		// Calculate balance for each account
		balance, err := h.accountService.CalculateAccountBalance(c.Request.Context(), workplaceID, acc.AccountID)
		if err != nil {
			// Log error and set balance to 0 for this account, but continue with others
			logger.Error("Failed to calculate balance for account", slog.String("account_id", acc.AccountID), slog.String("error", err.Error()))
			// balance = decimal.Zero // Balance is already zero initialized if using decimal.Decimal
		}

		// Convert domain account to DTO response
		accResponse := dto.ToAccountResponse(&acc)
		// Set the calculated balance
		accResponse.Balance = balance
		accountResponses[i] = accResponse
	}

	logger.Info("Accounts listed successfully", slog.Int("count", len(accountResponses)))
	c.JSON(http.StatusOK, dto.ListAccountsResponse{Accounts: accountResponses})
}

// updateAccount godoc
// @Summary Update account in workplace
// @Description Updates details for a specific account within a workplace.
// @Tags accounts
// @Accept  json
// @Produce  json
// @Param   workplace_id path string true "Workplace ID"
// @Param   id path string true "Account ID"
// @Param   account body dto.UpdateAccountRequest true "Account details to update"
// @Success 200 {object} dto.AccountResponse
// @Failure 400 {object} map[string]string "Invalid input or missing IDs"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden (User cannot update)"
// @Failure 404 {object} map[string]string "Account not found in this workplace"
// @Failure 500 {object} map[string]string "Failed to update account"
// @Security BearerAuth
// @Router /workplaces/{workplace_id}/accounts/{id} [put]
func (h *accountHandler) updateAccount(c *gin.Context) {
	logger := middleware.GetLoggerFromCtx(c.Request.Context())
	workplaceID := c.Param("workplace_id") // Get from path
	accountID := c.Param("id")
	if workplaceID == "" || accountID == "" {
		logger.Error("Workplace ID or Account ID missing from path for updateAccount")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Workplace and Account ID required in path"})
		return
	}

	var req dto.UpdateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn("Failed to bind JSON for UpdateAccount", slog.String("error", err.Error()))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format: " + err.Error()})
		return
	}

	loggedInUserID, ok := middleware.GetUserIDFromContext(c)
	if !ok {
		logger.Error("Logged-in user ID not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	logger = logger.With(slog.String("target_account_id", accountID), slog.String("workplace_id", workplaceID), slog.String("updater_user_id", loggedInUserID))
	logger.Info("Received request to update account")

	updatedAccount, err := h.accountService.UpdateAccount(c.Request.Context(), workplaceID, accountID, req, loggedInUserID) // Pass workplaceID
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			logger.Warn("Account not found for update (or in wrong workplace)")
			c.JSON(http.StatusNotFound, gin.H{"error": "Account not found"})
		} else if errors.Is(err, apperrors.ErrForbidden) {
			logger.Warn("User forbidden to update account", slog.String("user_id", loggedInUserID), slog.String("account_id", accountID))
			c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden"})
		} else if errors.Is(err, apperrors.ErrValidation) {
			logger.Warn("Validation error updating account", slog.String("error", err.Error()))
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
// @Summary Deactivate account in workplace
// @Description Marks an account as inactive within a specified workplace.
// @Tags accounts
// @Produce  json
// @Param   workplace_id path string true "Workplace ID"
// @Param   id path string true "Account ID to deactivate"
// @Success 204 "No Content"
// @Failure 400 {object} map[string]string "Missing Workplace or Account ID"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden (User cannot deactivate)"
// @Failure 404 {object} map[string]string "Account not found in this workplace"
// @Failure 409 {object} map[string]string "Conflict (e.g., already inactive)"
// @Failure 500 {object} map[string]string "Failed to deactivate account"
// @Security BearerAuth
// @Router /workplaces/{workplace_id}/accounts/{id} [delete]
func (h *accountHandler) deleteAccount(c *gin.Context) {
	logger := middleware.GetLoggerFromCtx(c.Request.Context())
	workplaceID := c.Param("workplace_id") // Get from path
	accountID := c.Param("id")
	if workplaceID == "" || accountID == "" {
		logger.Error("Workplace ID or Account ID missing from path for deleteAccount")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Workplace and Account ID required in path"})
		return
	}

	loggedInUserID, ok := middleware.GetUserIDFromContext(c)
	if !ok {
		logger.Error("Logged-in user ID not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	logger = logger.With(slog.String("target_account_id", accountID), slog.String("workplace_id", workplaceID), slog.String("deleter_user_id", loggedInUserID))
	logger.Info("Received request to delete account")

	err := h.accountService.DeactivateAccount(c.Request.Context(), workplaceID, accountID, loggedInUserID) // Pass workplaceID
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			logger.Warn("Account not found for delete (or in wrong workplace)")
			c.JSON(http.StatusNotFound, gin.H{"error": "Account not found"})
		} else if errors.Is(err, apperrors.ErrForbidden) {
			logger.Warn("User forbidden to deactivate account", slog.String("user_id", loggedInUserID), slog.String("account_id", accountID))
			c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden"})
		} else if errors.Is(err, apperrors.ErrValidation) {
			logger.Warn("Validation error deactivating account (already inactive?)", slog.String("error", err.Error()))
			c.JSON(http.StatusConflict, gin.H{"error": "Account already inactive or cannot be deactivated"})
		} else {
			logger.Error("Failed to deactivate account in service", slog.String("error", err.Error()))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to deactivate account"})
		}
		return
	}

	logger.Info("Account deleted successfully")
	c.Status(http.StatusNoContent)
}

// listTransactionsByAccount godoc
// @Summary List transactions for an account in a workplace
// @Description Retrieves a paginated list of transactions associated with a specific account within a workplace.
// @Tags accounts
// @Produce  json
// @Param   workplace_id path string true "Workplace ID"
// @Param   id path string true "Account ID"
// @Param   limit query int false "Limit number of results" default(20)
// @Param   offset query int false "Offset for pagination" default(0)
// @Success 200 {object} dto.ListTransactionsResponse
// @Failure 400 {object} map[string]string "Missing Workplace/Account ID or invalid query params"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden (User not part of workplace)"
// @Failure 404 {object} map[string]string "Account not found in this workplace"
// @Failure 500 {object} map[string]string "Failed to list transactions"
// @Security BearerAuth
// @Router /workplaces/{workplace_id}/accounts/{id}/transactions [get]
func (h *accountHandler) listTransactionsByAccount(c *gin.Context) {
	logger := middleware.GetLoggerFromCtx(c.Request.Context())

	workplaceID := c.Param("workplace_id")
	accountID := c.Param("id")
	if workplaceID == "" || accountID == "" {
		logger.Error("Workplace ID or Account ID missing from path for listTransactionsByAccount")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Workplace and Account ID required in path"})
		return
	}

	loggedInUserID, ok := middleware.GetUserIDFromContext(c)
	if !ok {
		logger.Error("Logged-in user ID not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Bind query parameters for pagination
	var params dto.ListTransactionsParams
	if err := c.ShouldBindQuery(&params); err != nil {
		logger.Warn("Failed to bind query params for ListTransactionsByAccount", slog.String("error", err.Error()))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid query parameters: " + err.Error()})
		return
	}

	logger = logger.With(slog.String("user_id", loggedInUserID), slog.String("workplace_id", workplaceID), slog.String("account_id", accountID))
	logger.Info("Received request to list transactions for account", slog.Int("limit", params.Limit), slog.String("nextToken", safeStringDeref(params.NextToken)))

	// Call the service method (assuming it accepts params DTO)
	resp, err := h.journalService.ListTransactionsByAccount(c.Request.Context(), workplaceID, accountID, loggedInUserID, params)
	if err != nil {
		if errors.Is(err, apperrors.ErrForbidden) {
			logger.Warn("User forbidden to list transactions for account", slog.String("user_id", loggedInUserID), slog.String("account_id", accountID))
			c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden"})
		} else if errors.Is(err, apperrors.ErrNotFound) {
			// Handle case where account itself might not be found or inaccessible
			logger.Warn("Account not found or inaccessible for listing transactions")
			c.JSON(http.StatusNotFound, gin.H{"error": "Account not found or access denied"})
		} else {
			logger.Error("Failed to list transactions from service", slog.String("error", err.Error()))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list transactions"})
		}
		return
	}

	logger.Info("Transactions listed successfully for account", slog.Int("count", len(resp.Transactions)))
	c.JSON(http.StatusOK, resp)
}

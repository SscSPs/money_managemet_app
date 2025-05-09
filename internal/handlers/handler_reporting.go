package handlers

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/SscSPs/money_managemet_app/internal/apperrors"
	portssvc "github.com/SscSPs/money_managemet_app/internal/core/ports/services"
	"github.com/SscSPs/money_managemet_app/internal/dto"
	"github.com/SscSPs/money_managemet_app/internal/middleware"
	"github.com/gin-gonic/gin"
)

// reportingHandler handles HTTP requests related to financial reports
type reportingHandler struct {
	reportingService portssvc.ReportingService
}

// newReportingHandler creates a new reportingHandler
func newReportingHandler(rs portssvc.ReportingService) *reportingHandler {
	return &reportingHandler{
		reportingService: rs,
	}
}

// registerReportingRoutes registers routes related to financial reports
func registerReportingRoutes(rg *gin.RouterGroup, reportingService portssvc.ReportingService) {
	h := newReportingHandler(reportingService)

	// Routes for reports are nested under a specific workplace
	reportingGroup := rg.Group("/reports")
	{
		reportingGroup.GET("/trial-balance", h.getTrialBalance)
		reportingGroup.GET("/profit-and-loss", h.getProfitAndLoss)
		reportingGroup.GET("/balance-sheet", h.getBalanceSheet)
	}
}

// getTrialBalance godoc
// @Summary Generate trial balance report
// @Description Generates a trial balance report as of a specific date
// @Tags reports
// @Produce json
// @Param workplace_id path string true "Workplace ID"
// @Param asOf query string false "Report date (YYYY-MM-DD)" default(current date)
// @Success 200 {object} dto.TrialBalanceResponse
// @Failure 400 {object} map[string]string "Invalid input"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden (User not authorized)"
// @Failure 500 {object} map[string]string "Failed to generate report"
// @Security BearerAuth
// @Router /workplaces/{workplace_id}/reports/trial-balance [get]
func (h *reportingHandler) getTrialBalance(c *gin.Context) {
	logger := middleware.GetLoggerFromCtx(c.Request.Context())
	workplaceID := c.Param("workplace_id")
	if workplaceID == "" {
		logger.Error("Workplace ID missing from path for getTrialBalance")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Workplace ID required in path"})
		return
	}

	userID, ok := middleware.GetUserIDFromContext(c)
	if !ok {
		logger.Error("User ID not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Parse asOf date parameter
	asOfStr := c.DefaultQuery("asOf", time.Now().Format("2006-01-02"))
	asOf, err := time.Parse("2006-01-02", asOfStr)
	if err != nil {
		logger.Warn("Invalid asOf date format", slog.String("asOf", asOfStr), slog.String("error", err.Error()))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid date format. Use YYYY-MM-DD"})
		return
	}

	logger = logger.With(
		slog.String("user_id", userID),
		slog.String("workplace_id", workplaceID),
		slog.String("asOf", asOfStr),
	)
	logger.Info("Received request to generate trial balance report")

	// Call service to generate report
	trialBalanceRows, err := h.reportingService.TrialBalance(c.Request.Context(), workplaceID, asOf, userID)
	if err != nil {
		if errors.Is(err, apperrors.ErrForbidden) {
			logger.Warn("User forbidden to access trial balance report")
			c.JSON(http.StatusForbidden, gin.H{"error": "You do not have permission to access this report"})
		} else if errors.Is(err, apperrors.ErrNotFound) {
			logger.Warn("Workplace not found")
			c.JSON(http.StatusNotFound, gin.H{"error": "Workplace not found"})
		} else {
			logger.Error("Failed to generate trial balance report", slog.String("error", err.Error()))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate trial balance report"})
		}
		return
	}

	// Convert domain objects to DTO
	response := dto.ToTrialBalanceResponse(trialBalanceRows, asOf)

	logger.Info("Trial balance report generated successfully", slog.Int("row_count", len(trialBalanceRows)))
	c.JSON(http.StatusOK, response)
}

// getProfitAndLoss godoc
// @Summary Generate profit and loss report
// @Description Generates a profit and loss report for a specific period
// @Tags reports
// @Produce json
// @Param workplace_id path string true "Workplace ID"
// @Param fromDate query string false "Start date (YYYY-MM-DD)" default(first day of current month)
// @Param toDate query string false "End date (YYYY-MM-DD)" default(current date)
// @Success 200 {object} dto.ProfitAndLossResponse
// @Failure 400 {object} map[string]string "Invalid input"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden (User not authorized)"
// @Failure 500 {object} map[string]string "Failed to generate report"
// @Security BearerAuth
// @Router /workplaces/{workplace_id}/reports/profit-and-loss [get]
func (h *reportingHandler) getProfitAndLoss(c *gin.Context) {
	logger := middleware.GetLoggerFromCtx(c.Request.Context())
	workplaceID := c.Param("workplace_id")
	if workplaceID == "" {
		logger.Error("Workplace ID missing from path for getProfitAndLoss")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Workplace ID required in path"})
		return
	}

	userID, ok := middleware.GetUserIDFromContext(c)
	if !ok {
		logger.Error("User ID not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Get current time for default date calculations
	now := time.Now()

	// Default from date is first day of current month
	firstDayOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	fromStr := c.DefaultQuery("fromDate", firstDayOfMonth.Format("2006-01-02"))
	from, err := time.Parse("2006-01-02", fromStr)
	if err != nil {
		logger.Warn("Invalid from date format", slog.String("fromDate", fromStr), slog.String("error", err.Error()))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid fromDate format. Use YYYY-MM-DD"})
		return
	}

	// Default to date is today
	toStr := c.DefaultQuery("toDate", now.Format("2006-01-02"))
	to, err := time.Parse("2006-01-02", toStr)
	if err != nil {
		logger.Warn("Invalid to date format", slog.String("toDate", toStr), slog.String("error", err.Error()))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid toDate format. Use YYYY-MM-DD"})
		return
	}

	// Validate date range
	if from.After(to) {
		logger.Warn("Invalid date range", slog.String("fromDate", fromStr), slog.String("toDate", toStr))
		c.JSON(http.StatusBadRequest, gin.H{"error": "fromDate must be before or equal to toDate"})
		return
	}

	logger = logger.With(
		slog.String("user_id", userID),
		slog.String("workplace_id", workplaceID),
		slog.String("fromDate", fromStr),
		slog.String("toDate", toStr),
	)
	logger.Info("Received request to generate profit and loss report")

	// Call service to generate report
	report, err := h.reportingService.ProfitAndLoss(c.Request.Context(), workplaceID, from, to, userID)
	if err != nil {
		if errors.Is(err, apperrors.ErrForbidden) {
			logger.Warn("User forbidden to access profit and loss report")
			c.JSON(http.StatusForbidden, gin.H{"error": "You do not have permission to access this report"})
		} else if errors.Is(err, apperrors.ErrNotFound) {
			logger.Warn("Workplace not found")
			c.JSON(http.StatusNotFound, gin.H{"error": "Workplace not found"})
		} else {
			logger.Error("Failed to generate profit and loss report", slog.String("error", err.Error()))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate profit and loss report"})
		}
		return
	}

	// Convert domain objects to DTO
	response := dto.ToProfitAndLossResponse(report, from, to)

	logger.Info("Profit and loss report generated successfully",
		slog.Int("revenue_accounts", len(report.Revenue)),
		slog.Int("expense_accounts", len(report.Expenses)))
	c.JSON(http.StatusOK, response)
}

// getBalanceSheet godoc
// @Summary Generate balance sheet report
// @Description Generates a balance sheet report as of a specific date
// @Tags reports
// @Produce json
// @Param workplace_id path string true "Workplace ID"
// @Param asOf query string false "Report date (YYYY-MM-DD)" default(current date)
// @Success 200 {object} dto.BalanceSheetResponse
// @Failure 400 {object} map[string]string "Invalid input"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden (User not authorized)"
// @Failure 500 {object} map[string]string "Failed to generate report"
// @Security BearerAuth
// @Router /workplaces/{workplace_id}/reports/balance-sheet [get]
func (h *reportingHandler) getBalanceSheet(c *gin.Context) {
	logger := middleware.GetLoggerFromCtx(c.Request.Context())
	workplaceID := c.Param("workplace_id")
	if workplaceID == "" {
		logger.Error("Workplace ID missing from path for getBalanceSheet")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Workplace ID required in path"})
		return
	}

	userID, ok := middleware.GetUserIDFromContext(c)
	if !ok {
		logger.Error("User ID not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Parse asOf date parameter
	asOfStr := c.DefaultQuery("asOf", time.Now().Format("2006-01-02"))
	asOf, err := time.Parse("2006-01-02", asOfStr)
	if err != nil {
		logger.Warn("Invalid asOf date format", slog.String("asOf", asOfStr), slog.String("error", err.Error()))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid date format. Use YYYY-MM-DD"})
		return
	}

	logger = logger.With(
		slog.String("user_id", userID),
		slog.String("workplace_id", workplaceID),
		slog.String("asOf", asOfStr),
	)
	logger.Info("Received request to generate balance sheet report")

	// Call service to generate report
	report, err := h.reportingService.BalanceSheet(c.Request.Context(), workplaceID, asOf, userID)
	if err != nil {
		if errors.Is(err, apperrors.ErrForbidden) {
			logger.Warn("User forbidden to access balance sheet report")
			c.JSON(http.StatusForbidden, gin.H{"error": "You do not have permission to access this report"})
		} else if errors.Is(err, apperrors.ErrNotFound) {
			logger.Warn("Workplace not found")
			c.JSON(http.StatusNotFound, gin.H{"error": "Workplace not found"})
		} else {
			logger.Error("Failed to generate balance sheet report", slog.String("error", err.Error()))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate balance sheet report"})
		}
		return
	}

	// Convert domain objects to DTO
	response := dto.ToBalanceSheetResponse(report, asOf)

	logger.Info("Balance sheet report generated successfully",
		slog.Int("asset_accounts", len(report.Assets)),
		slog.Int("liability_accounts", len(report.Liabilities)),
		slog.Int("equity_accounts", len(report.Equity)))
	c.JSON(http.StatusOK, response)
}

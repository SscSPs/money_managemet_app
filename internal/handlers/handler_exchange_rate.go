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

const (
	queryParamFrom = "from"
	queryParamTo   = "to"
)

type ExchangeRateHandler struct {
	exchangeRateService *services.ExchangeRateService
}

func newExchangeRateHandler(exchangeRateService *services.ExchangeRateService) *ExchangeRateHandler {
	return &ExchangeRateHandler{
		exchangeRateService: exchangeRateService,
	}
}

// createExchangeRate godoc
// @Summary Create a new exchange rate
// @Description Adds a new conversion rate between two currencies for a specific date.
// @Tags exchange-rates
// @Accept  json
// @Produce  json
// @Param   rate body dto.CreateExchangeRateRequest true "Exchange Rate details"
// @Success 201 {object} dto.ExchangeRateResponse
// @Failure 400 {object} map[string]string "Invalid input format or validation error"
// @Failure 401 {object} map[string]string "Unauthorized - User ID not found in context"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /exchange-rates [post]
// @Security BearerAuth
func (h *ExchangeRateHandler) createExchangeRate(c *gin.Context) {
	logger := middleware.GetLoggerFromContext(c)

	var createReq dto.CreateExchangeRateRequest
	if err := c.ShouldBindJSON(&createReq); err != nil {
		logger.Error("Failed to bind JSON for CreateExchangeRate", slog.String("error", err.Error()))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format: " + err.Error()})
		return
	}

	// TODO: Consider adding more specific validation here if needed beyond DTO tags

	creatorUserID, ok := middleware.GetUserIDFromContext(c)
	if !ok {
		logger.Error("Creator user ID not found in context for CreateExchangeRate")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	logger = logger.With(slog.String("creator_user_id", creatorUserID))

	rate, err := h.exchangeRateService.CreateExchangeRate(c.Request.Context(), createReq, creatorUserID)
	if err != nil {
		// TODO: Add handling for specific errors like duplicate rate (apperrors.ErrDuplicate?)
		logger.Error("Failed to create exchange rate in service", slog.String("error", err.Error()),
			slog.String("from", createReq.FromCurrencyCode),
			slog.String("to", createReq.ToCurrencyCode),
			slog.Time("date", createReq.DateEffective))

		if errors.Is(err, apperrors.ErrValidation) { // Or other specific errors from service
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create exchange rate"})
		}
		return
	}

	logger.Info("Exchange rate created successfully", slog.String("rate_id", rate.ExchangeRateID))
	c.JSON(http.StatusCreated, dto.ToExchangeRateResponse(rate))
}

// getExchangeRate godoc
// @Summary Get an exchange rate
// @Description Retrieves a specific exchange rate based on from/to currencies and effective date.
// @Tags exchange-rates
// @Accept  json
// @Produce  json
// @Param   from query string true "From Currency Code (3 uppercase letters)" Format(string) example(USD)
// @Param   to query string true "To Currency Code (3 uppercase letters)" Format(string) example(EUR)
// @Success 200 {object} dto.ExchangeRateResponse
// @Failure 400 {object} map[string]string "Invalid query parameter format or value"
// @Failure 404 {object} map[string]string "Exchange rate not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /exchange-rates [get]
// @Security BearerAuth
func (h *ExchangeRateHandler) getExchangeRate(c *gin.Context) {
	logger := middleware.GetLoggerFromContext(c)

	fromCode := c.Query(queryParamFrom)
	toCode := c.Query(queryParamTo)

	if fromCode == "" || toCode == "" {
		logger.Warn("Missing required query parameters for GetExchangeRate",
			slog.String("from", fromCode), slog.String("to", toCode))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing required query parameters: from, to"})
		return
	}

	// Service layer handles code validation (case, length)
	rate, err := h.exchangeRateService.GetExchangeRate(c.Request.Context(), fromCode, toCode)
	if err != nil {
		logger.Error("Failed to get exchange rate from service", slog.String("error", err.Error()),
			slog.String("from", fromCode), slog.String("to", toCode))

		if errors.Is(err, apperrors.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Exchange rate not found"})
		} else if errors.Is(err, apperrors.ErrValidation) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve exchange rate"})
		}
		return
	}

	logger.Debug("Exchange rate retrieved successfully", slog.String("rate_id", rate.ExchangeRateID))
	c.JSON(http.StatusOK, dto.ToExchangeRateResponse(rate))
}

// registerExchangeRateRoutes registers exchange rate specific routes
func registerExchangeRateRoutes(rg *gin.RouterGroup, dbPool *pgxpool.Pool) {
	// Instantiate dependencies
	// Need CurrencyService for validation, which requires CurrencyRepository
	currencyRepo := pgsql.NewCurrencyRepository(dbPool)
	currencyService := services.NewCurrencyService(currencyRepo)

	exchangeRateRepo := pgsql.NewExchangeRateRepository(dbPool)
	// Pass CurrencyService to ExchangeRateService constructor
	exchangeRateService := services.NewExchangeRateService(exchangeRateRepo, currencyService)
	exchangeRateHandler := newExchangeRateHandler(exchangeRateService)

	// Define routes under /exchange-rates group
	exchangeRates := rg.Group("/exchange-rates")
	{
		exchangeRates.POST("/", exchangeRateHandler.createExchangeRate)
		exchangeRates.GET("/", exchangeRateHandler.getExchangeRate)
		// Add other routes like PUT, DELETE, GET by ID later if needed
	}
}

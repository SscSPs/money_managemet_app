package handlers

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/SscSPs/money_managemet_app/internal/apperrors"
	portssvc "github.com/SscSPs/money_managemet_app/internal/core/ports/services" // Use ports services

	// "github.com/SscSPs/money_managemet_app/internal/core/services" // Remove concrete services
	"github.com/SscSPs/money_managemet_app/internal/dto"
	"github.com/SscSPs/money_managemet_app/internal/middleware"
	"github.com/gin-gonic/gin"
	// Needed for request DTO
)

// exchangeRateHandler handles HTTP requests related to exchange rates.
type exchangeRateHandler struct {
	exchangeRateService portssvc.ExchangeRateSvcFacade // Updated to use ExchangeRateSvcFacade
}

// newExchangeRateHandler creates a new exchangeRateHandler.
func newExchangeRateHandler(ers portssvc.ExchangeRateSvcFacade) *exchangeRateHandler { // Updated interface
	return &exchangeRateHandler{
		exchangeRateService: ers,
	}
}

// registerExchangeRateRoutes registers routes related to exchange rates.
func registerExchangeRateRoutes(rg *gin.RouterGroup, exchangeRateService portssvc.ExchangeRateSvcFacade) { // Updated interface
	h := newExchangeRateHandler(exchangeRateService)

	exchangeRates := rg.Group("/exchange-rates")
	{
		exchangeRates.POST("", h.createExchangeRate)
		exchangeRates.GET("/:from/:to", h.getExchangeRate)
		exchangeRates.GET("/id/:id", h.getExchangeRateByID)
		exchangeRates.GET("/batch", h.getExchangeRatesByIDs)
		exchangeRates.GET("", h.listExchangeRates)
		exchangeRates.GET("/currency/:currencyCode", h.listExchangeRatesByCurrency)
	}
}

// createExchangeRate godoc
// @Summary Create a new exchange rate
// @Description Adds a new exchange rate between two currencies for a specific date
// @Tags exchange rates
// @Accept  json
// @Produce  json
// @Param   rate body dto.CreateExchangeRateRequest true "Exchange Rate details"
// @Success 201 {object} dto.ExchangeRateResponse
// @Failure 400 {object} map[string]string "Invalid input format or validation error"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 500 {object} map[string]string "Failed to create exchange rate"
// @Security BearerAuth
// @Router /exchange-rates [post]
func (h *exchangeRateHandler) createExchangeRate(c *gin.Context) {
	logger := middleware.GetLoggerFromCtx(c.Request.Context())
	var req dto.CreateExchangeRateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn("Failed to bind JSON for CreateExchangeRate", slog.String("error", err.Error()))
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

	// TODO: Add authorization check - does user have permission?

	logger = logger.With(slog.String("creator_user_id", creatorUserID))
	logger.Info("Received request to create exchange rate",
		slog.String("from", req.FromCurrencyCode),
		slog.String("to", req.ToCurrencyCode),
		slog.Any("rate", req.Rate),
		slog.Time("date_effective", req.DateEffective),
	)

	createdRate, err := h.exchangeRateService.CreateExchangeRate(c.Request.Context(), req, creatorUserID)
	if err != nil {
		if errors.Is(err, apperrors.ErrValidation) {
			logger.Warn("Validation error creating exchange rate", slog.String("error", err.Error()))
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		} else {
			logger.Error("Failed to create exchange rate in service", slog.String("error", err.Error()))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create exchange rate"})
		}
		return
	}

	logger.Info("Exchange rate created successfully", slog.String("rate_id", createdRate.ExchangeRateID))
	c.JSON(http.StatusCreated, dto.ToExchangeRateResponse(createdRate))
}

// getExchangeRate godoc
// @Summary Get an exchange rate
// @Description Retrieves the latest exchange rate for a given currency pair
// @Tags exchange rates
// @Produce  json
// @Param   from path string true "From Currency Code (3 letters)" MinLength(3) MaxLength(3)
// @Param   to   path string true "To Currency Code (3 letters)" MinLength(3) MaxLength(3)
// @Success 200 {object} dto.ExchangeRateResponse
// @Failure 400 {object} map[string]string "Invalid currency code format"
// @Failure 404 {object} map[string]string "Exchange rate not found"
// @Failure 500 {object} map[string]string "Failed to retrieve exchange rate"
// @Security BearerAuth
// @Router /exchange-rates/{from}/{to} [get]
func (h *exchangeRateHandler) getExchangeRate(c *gin.Context) {
	logger := middleware.GetLoggerFromCtx(c.Request.Context())
	fromCode := c.Param("from")
	toCode := c.Param("to")

	// Basic validation - service likely does more thorough validation
	if len(fromCode) != 3 || len(toCode) != 3 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Currency codes must be 3 letters"})
		return
	}

	logger = logger.With(slog.String("from_code", fromCode), slog.String("to_code", toCode))
	logger.Info("Received request to get exchange rate")

	rate, err := h.exchangeRateService.GetExchangeRate(c.Request.Context(), fromCode, toCode)
	if err != nil {
		if errors.Is(err, apperrors.ErrValidation) {
			logger.Warn("Validation error getting exchange rate", slog.String("error", err.Error()))
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		} else if errors.Is(err, apperrors.ErrNotFound) {
			logger.Warn("Exchange rate not found")
			c.JSON(http.StatusNotFound, gin.H{"error": "Exchange rate not found"})
		} else {
			logger.Error("Failed to get exchange rate from service", slog.String("error", err.Error()))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve exchange rate"})
		}
		return
	}

	logger.Info("Exchange rate retrieved successfully")
	c.JSON(http.StatusOK, dto.ToExchangeRateResponse(rate))
}

// getExchangeRateByID godoc
// @Summary Get exchange rate by ID
// @Description Retrieves an exchange rate by its unique identifier
// @Tags exchange rates
// @Produce  json
// @Param   id   path     string  true  "Exchange Rate ID"
// @Success 200 {object} dto.ExchangeRateResponse
// @Failure 400 {object} map[string]string "Invalid ID format"
// @Failure 404 {object} map[string]string "Exchange rate not found"
// @Failure 500 {object} map[string]string "Failed to retrieve exchange rate"
// @Security BearerAuth
// @Router /exchange-rates/id/{id} [get]
func (h *exchangeRateHandler) getExchangeRateByID(c *gin.Context) {
	logger := middleware.GetLoggerFromCtx(c.Request.Context())
	rateID := c.Param("id")

	if rateID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Exchange rate ID is required"})
		return
	}

	logger = logger.With(slog.String("rate_id", rateID))
	logger.Info("Received request to get exchange rate by ID")

	rate, err := h.exchangeRateService.GetExchangeRateByID(c.Request.Context(), rateID)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			logger.Warn("Exchange rate not found")
			c.JSON(http.StatusNotFound, gin.H{"error": "Exchange rate not found"})
		} else {
			logger.Error("Failed to get exchange rate by ID", slog.String("error", err.Error()))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve exchange rate"})
		}
		return
	}

	logger.Info("Exchange rate retrieved successfully by ID")
	c.JSON(http.StatusOK, dto.ToExchangeRateResponse(rate))
}

// getExchangeRatesByIDs godoc
// @Summary Get multiple exchange rates by their IDs
// @Description Retrieves multiple exchange rates by their unique identifiers
// @Tags exchange rates
// @Produce  json
// @Param   ids  query    []string  true  "Array of Exchange Rate IDs"
// @Success 200 {array}  dto.ExchangeRateResponse
// @Failure 400 {object} map[string]string "Invalid ID format"
// @Failure 500 {object} map[string]string "Failed to retrieve exchange rates"
// @Security BearerAuth
// @Router /exchange-rates/batch [get]
func (h *exchangeRateHandler) getExchangeRatesByIDs(c *gin.Context) {
	logger := middleware.GetLoggerFromCtx(c.Request.Context())

	// Get IDs from query parameters
	ids := c.QueryArray("ids")
	if len(ids) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "At least one rate ID is required"})
		return
	}

	logger = logger.With(slog.Any("rate_ids", ids))
	logger.Info("Received request to get multiple exchange rates by IDs")

	rates, err := h.exchangeRateService.GetExchangeRateByIDs(c.Request.Context(), ids)
	if err != nil {
		logger.Error("Failed to get exchange rates by IDs", slog.String("error", err.Error()))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve exchange rates"})
		return
	}

	responses := make([]dto.ExchangeRateResponse, len(rates))
	for i, rate := range rates {
		rateCopy := rate // Create a local copy to avoid taking the address of the loop variable
		responses[i] = dto.ToExchangeRateResponse(&rateCopy)
	}

	logger.Info("Successfully retrieved exchange rates by IDs")
	c.JSON(http.StatusOK, responses)
}

// listExchangeRates godoc
// @Summary List all exchange rates
// @Description Retrieves all available exchange rates
// @Tags exchange rates
// @Produce  json
// @Success 200 {array}  dto.ExchangeRateResponse
// @Failure 500 {object} map[string]string "Failed to retrieve exchange rates"
// @Security BearerAuth
// @Router /exchange-rates [get]
func (h *exchangeRateHandler) listExchangeRates(c *gin.Context) {
	logger := middleware.GetLoggerFromCtx(c.Request.Context())
	logger.Info("Received request to list all exchange rates")

	rates, err := h.exchangeRateService.ListExchangeRates(c.Request.Context())
	if err != nil {
		logger.Error("Failed to list exchange rates", slog.String("error", err.Error()))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve exchange rates"})
		return
	}

	responses := make([]dto.ExchangeRateResponse, len(rates))
	for i, rate := range rates {
		rateCopy := rate // Create a local copy to avoid taking the address of the loop variable
		responses[i] = dto.ToExchangeRateResponse(&rateCopy)
	}

	logger.Info("Successfully retrieved all exchange rates")
	c.JSON(http.StatusOK, responses)
}

// listExchangeRatesByCurrency godoc
// @Summary List exchange rates by currency
// @Description Retrieves all exchange rates for a specific currency
// @Tags exchange rates
// @Produce  json
// @Param   currencyCode path string true "Currency Code (3 letters)"
// @Success 200 {array}  dto.ExchangeRateResponse
// @Failure 400 {object} map[string]string "Invalid currency code format"
// @Failure 500 {object} map[string]string "Failed to retrieve exchange rates"
// @Security BearerAuth
// @Router /exchange-rates/currency/{currencyCode} [get]
func (h *exchangeRateHandler) listExchangeRatesByCurrency(c *gin.Context) {
	logger := middleware.GetLoggerFromCtx(c.Request.Context())
	currencyCode := c.Param("currencyCode")

	if len(currencyCode) != 3 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Currency code must be 3 letters"})
		return
	}

	logger = logger.With(slog.String("currency_code", currencyCode))
	logger.Info("Received request to list exchange rates by currency")

	rates, err := h.exchangeRateService.ListExchangeRatesByCurrency(c.Request.Context(), currencyCode)
	if err != nil {
		logger.Error("Failed to list exchange rates by currency", slog.String("error", err.Error()))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve exchange rates"})
		return
	}

	responses := make([]dto.ExchangeRateResponse, len(rates))
	for i, rate := range rates {
		rateCopy := rate // Create a local copy to avoid taking the address of the loop variable
		responses[i] = dto.ToExchangeRateResponse(&rateCopy)
	}

	logger.Info("Successfully retrieved exchange rates by currency")
	c.JSON(http.StatusOK, responses)
}

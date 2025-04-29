package handlers

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	// For uppercase conversion
	"github.com/SscSPs/money_managemet_app/internal/apperrors"
	portssvc "github.com/SscSPs/money_managemet_app/internal/core/ports/services" // Use ports services

	// "github.com/SscSPs/money_managemet_app/internal/core/services" // Remove concrete services
	"github.com/SscSPs/money_managemet_app/internal/dto"
	"github.com/SscSPs/money_managemet_app/internal/middleware"
	"github.com/gin-gonic/gin"
)

// currencyHandler handles HTTP requests related to currencies.
type currencyHandler struct {
	currencyService portssvc.CurrencyService // Use interface
}

// newCurrencyHandler creates a new currencyHandler.
func newCurrencyHandler(cs portssvc.CurrencyService) *currencyHandler { // Use interface
	return &currencyHandler{
		currencyService: cs,
	}
}

// registerCurrencyRoutes registers routes related to currencies.
func registerCurrencyRoutes(rg *gin.RouterGroup, currencyService portssvc.CurrencyService) { // Use interface
	h := newCurrencyHandler(currencyService) // Pass interface

	currencies := rg.Group("/currencies")
	{
		currencies.POST("", h.createCurrency)
		currencies.GET("", h.listCurrencies)
		currencies.GET("/:code", h.getCurrencyByCode)
	}
}

// createCurrency godoc
// @Summary Create a new currency
// @Description Adds a new currency to the system (admin operation)
// @Tags currencies
// @Accept  json
// @Produce  json
// @Param   currency body dto.CreateCurrencyRequest true "Currency details"
// @Success 201 {object} dto.CurrencyResponse
// @Failure 400 {object} map[string]string "Invalid input"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 409 {object} map[string]string "Currency code already exists"
// @Failure 500 {object} map[string]string "Failed to create currency"
// @Security BearerAuth
// @Router /currencies [post]
func (h *currencyHandler) createCurrency(c *gin.Context) {
	logger := middleware.GetLoggerFromCtx(c.Request.Context())
	var req dto.CreateCurrencyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn("Failed to bind JSON for CreateCurrency", slog.String("error", err.Error()))
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

	// TODO: Add authorization check - is the user an admin?

	logger = logger.With(slog.String("creator_user_id", creatorUserID))
	logger.Info("Received request to create currency", slog.String("currency_code", req.CurrencyCode))

	createdCurrency, err := h.currencyService.CreateCurrency(c.Request.Context(), req, creatorUserID)
	if err != nil {
		if errors.Is(err, apperrors.ErrDuplicate) {
			logger.Warn("Attempted to create duplicate currency", slog.String("currency_code", req.CurrencyCode))
			c.JSON(http.StatusConflict, gin.H{"error": fmt.Sprintf("Currency code '%s' already exists", req.CurrencyCode)})
		} else if errors.Is(err, apperrors.ErrValidation) {
			logger.Warn("Validation error creating currency", slog.String("error", err.Error()))
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		} else {
			logger.Error("Failed to create currency in service", slog.String("error", err.Error()))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create currency"})
		}
		return
	}

	logger.Info("Currency created successfully", slog.String("currency_code", createdCurrency.CurrencyCode))
	c.JSON(http.StatusCreated, dto.ToCurrencyResponse(createdCurrency))
}

// getCurrencyByCode godoc
// @Summary Get a currency by code
// @Description Retrieves details for a specific currency by its 3-letter code
// @Tags currencies
// @Produce  json
// @Param   code path string true "Currency Code (3 letters)" MinLength(3) MaxLength(3)
// @Success 200 {object} dto.CurrencyResponse
// @Failure 404 {object} map[string]string "Currency not found"
// @Failure 500 {object} map[string]string "Failed to retrieve currency"
// @Security BearerAuth
// @Router /currencies/{code} [get]
func (h *currencyHandler) getCurrencyByCode(c *gin.Context) {
	logger := middleware.GetLoggerFromCtx(c.Request.Context())
	currencyCode := c.Param("code")

	// Basic validation - service likely does more thorough validation
	if len(currencyCode) != 3 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Currency code must be 3 letters"})
		return
	}

	logger = logger.With(slog.String("currency_code", currencyCode))
	logger.Info("Received request to get currency by code")

	currency, err := h.currencyService.GetCurrencyByCode(c.Request.Context(), currencyCode)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			logger.Warn("Currency not found")
			c.JSON(http.StatusNotFound, gin.H{"error": "Currency not found"})
		} else {
			logger.Error("Failed to get currency from service", slog.String("error", err.Error()))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve currency"})
		}
		return
	}

	logger.Info("Currency retrieved successfully")
	c.JSON(http.StatusOK, dto.ToCurrencyResponse(currency))
}

// listCurrencies godoc
// @Summary List all currencies
// @Description Retrieves a list of all available currencies
// @Tags currencies
// @Produce  json
// @Success 200 {array} dto.CurrencyResponse
// @Failure 500 {object} map[string]string "Failed to list currencies"
// @Security BearerAuth
// @Router /currencies [get]
func (h *currencyHandler) listCurrencies(c *gin.Context) {
	logger := middleware.GetLoggerFromCtx(c.Request.Context())
	logger.Info("Received request to list currencies")

	currencies, err := h.currencyService.ListCurrencies(c.Request.Context())
	if err != nil {
		logger.Error("Failed to list currencies from service", slog.String("error", err.Error()))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list currencies"})
		return
	}

	// Convert domain currencies to DTOs
	currencyResponses := make([]dto.CurrencyResponse, len(currencies))
	for i, curr := range currencies {
		currencyResponses[i] = dto.ToCurrencyResponse(&curr) // Corrected: ToCurrencyResponse returns value
	}

	logger.Info("Currencies listed successfully", slog.Int("count", len(currencyResponses)))
	// Return the slice directly as ListCurrenciesResponse might not exist
	c.JSON(http.StatusOK, currencyResponses) // Return slice directly
}

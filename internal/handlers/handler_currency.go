package handlers

import (
	"errors"
	"log/slog"
	"net/http"
	"strings" // For uppercase conversion

	"github.com/SscSPs/money_managemet_app/internal/apperrors"
	"github.com/SscSPs/money_managemet_app/internal/core/services"
	"github.com/SscSPs/money_managemet_app/internal/dto"
	"github.com/SscSPs/money_managemet_app/internal/middleware"
	"github.com/gin-gonic/gin"
	// TODO: Add logging import
)

type CurrencyHandler struct {
	currencyService *services.CurrencyService
	// logger          *slog.Logger // Removed
}

// NewCurrencyHandler no longer needs logger
func NewCurrencyHandler(currencyService *services.CurrencyService) *CurrencyHandler {
	return &CurrencyHandler{
		currencyService: currencyService,
	}
}

// CreateCurrency godoc
// @Summary Create a new currency
// @Description Adds a new currency to the system (e.g., USD, EUR)
// @Tags currencies
// @Accept  json
// @Produce  json
// @Param   currency body dto.CreateCurrencyRequest true "Currency details"
// @Success 201 {object} dto.CurrencyResponse
// @Failure 400 {object} string "Invalid input"
// @Failure 500 {object} string "Internal server error"
// @Router /currencies [post]
func (h *CurrencyHandler) CreateCurrency(c *gin.Context) {
	logger := middleware.GetLoggerFromContext(c) // Get logger from context

	var createReq dto.CreateCurrencyRequest
	if err := c.ShouldBindJSON(&createReq); err != nil {
		logger.Error("Failed to bind JSON for CreateCurrency", slog.String("error", err.Error()))
		// Use generic error message
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	// Validate currency code format
	if len(createReq.CurrencyCode) != 3 || createReq.CurrencyCode != strings.ToUpper(createReq.CurrencyCode) {
		logger.Warn("Invalid currency code format in request body", slog.String("code", createReq.CurrencyCode))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Currency code must be 3 uppercase letters"})
		return
	}

	// Get creator user ID from context
	creatorUserID, ok := middleware.GetUserIDFromContext(c)
	if !ok {
		logger.Error("Creator user ID not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	logger = logger.With(slog.String("creator_user_id", creatorUserID))

	currency, err := h.currencyService.CreateCurrency(c.Request.Context(), createReq, creatorUserID)
	if err != nil {
		// TODO: Check for specific errors (e.g., duplicate currency code)
		logger.Error("Failed to create currency in service", slog.String("error", err.Error()), slog.String("code", createReq.CurrencyCode))
		// Use generic error message
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create currency"})
		return
	}

	logger.Info("Currency created successfully", slog.String("code", currency.CurrencyCode))
	c.JSON(http.StatusCreated, dto.ToCurrencyResponse(currency))
}

// GetCurrency godoc
// @Summary Get a currency by code
// @Description Retrieves details for a specific currency by its 3-letter code
// @Tags currencies
// @Accept  json
// @Produce  json
// @Param   currencyCode path string true "Currency Code (3 uppercase letters)"
// @Success 200 {object} dto.CurrencyResponse
// @Failure 400 {object} string "Invalid currency code format"
// @Failure 404 {object} string "Currency not found"
// @Failure 500 {object} string "Internal server error"
// @Router /currencies/{currencyCode} [get]
func (h *CurrencyHandler) GetCurrency(c *gin.Context) {
	logger := middleware.GetLoggerFromContext(c) // Get logger from context
	currencyCode := strings.ToUpper(c.Param("currencyCode"))

	if len(currencyCode) != 3 { // Basic validation
		logger.Warn("Invalid currency code format requested in path", slog.String("code", c.Param("currencyCode")))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Currency code must be 3 letters"})
		return
	}

	currency, err := h.currencyService.GetCurrencyByCode(c.Request.Context(), currencyCode)
	if err != nil {
		// Check for specific ErrNotFound
		if errors.Is(err, apperrors.ErrNotFound) {
			logger.Warn("Currency not found", slog.String("code", currencyCode))
			c.JSON(http.StatusNotFound, gin.H{"error": "Currency not found"})
			return
		}
		// Log other errors
		logger.Error("Failed to get currency from service", slog.String("error", err.Error()), slog.String("code", currencyCode))
		// Use generic error message
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve currency"})
		return
	}

	// No need for explicit nil check if service returns ErrNotFound

	logger.Debug("Currency retrieved successfully", slog.String("code", currency.CurrencyCode))
	c.JSON(http.StatusOK, dto.ToCurrencyResponse(currency))
}

// ListCurrencies godoc
// @Summary List all available currencies
// @Description Retrieves a list of all currencies supported by the system
// @Tags currencies
// @Accept  json
// @Produce  json
// @Success 200 {array} dto.CurrencyResponse
// @Failure 500 {object} string "Internal server error"
// @Router /currencies [get]
func (h *CurrencyHandler) ListCurrencies(c *gin.Context) {
	logger := middleware.GetLoggerFromContext(c) // Get logger from context

	currencies, err := h.currencyService.ListCurrencies(c.Request.Context())
	if err != nil {
		logger.Error("Failed to list currencies from service", slog.String("error", err.Error()))
		// Use generic error message
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list currencies"})
		return
	}

	logger.Debug("Currencies listed successfully", slog.Int("count", len(currencies)))
	c.JSON(http.StatusOK, dto.ToListCurrencyResponse(currencies))
}

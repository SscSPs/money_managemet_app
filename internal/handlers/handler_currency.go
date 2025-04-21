package handlers

import (
	"net/http"
	"strings" // For uppercase conversion

	"github.com/SscSPs/money_managemet_app/internal/core/services"
	"github.com/SscSPs/money_managemet_app/internal/dto"
	"github.com/gin-gonic/gin"
	// TODO: Add logging import
)

type CurrencyHandler struct {
	currencyService *services.CurrencyService
	// TODO: Inject logger
}

func NewCurrencyHandler(currencyService *services.CurrencyService) *CurrencyHandler {
	return &CurrencyHandler{currencyService: currencyService}
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
	var createReq dto.CreateCurrencyRequest
	if err := c.ShouldBindJSON(&createReq); err != nil {
		// TODO: Add structured logging
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input: " + err.Error()})
		return
	}

	// Although binding handles uppercase, explicit check is good practice
	if len(createReq.CurrencyCode) != 3 || createReq.CurrencyCode != strings.ToUpper(createReq.CurrencyCode) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Currency code must be 3 uppercase letters"})
		return
	}

	// TODO: Get actual creator UserID from request context
	creatorUserID := createReq.UserID

	currency, err := h.currencyService.CreateCurrency(c.Request.Context(), createReq, creatorUserID)
	if err != nil {
		// TODO: Add structured logging
		// TODO: Check for specific errors (e.g., duplicate currency code if not handled by DB upsert)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create currency: " + err.Error()})
		return
	}

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
	currencyCode := strings.ToUpper(c.Param("currencyCode")) // Ensure uppercase

	if len(currencyCode) != 3 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Currency code must be 3 letters"})
		return
	}

	currency, err := h.currencyService.GetCurrencyByCode(c.Request.Context(), currencyCode)
	if err != nil {
		// TODO: Add structured logging
		// TODO: The service should return a specific error for not found
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get currency: " + err.Error()})
		return
	}

	if currency == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Currency not found"})
		return
	}

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
	currencies, err := h.currencyService.ListCurrencies(c.Request.Context())
	if err != nil {
		// TODO: Add structured logging
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list currencies: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.ToListCurrencyResponse(currencies))
}

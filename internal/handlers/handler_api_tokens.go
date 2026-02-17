package handlers

import (
	"net/http"
	"time"

	"github.com/SscSPs/money_managemet_app/internal/core/ports/services"
	"github.com/SscSPs/money_managemet_app/internal/dto"
	"github.com/SscSPs/money_managemet_app/internal/middleware"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// APIErrorResponse represents a generic error response for API operations
// @Description Generic error response containing a message describing the error
// @Description This is used for all error responses in the API
type APIErrorResponse struct {
	// Message contains the error message
	Message string `json:"message" example:"An error occurred"`
}

// APITokenHandler handles HTTP requests for API token operations
type APITokenHandler struct {
	tokenSvc services.APITokenSvc
}

// NewAPITokenHandler creates a new APITokenHandler
func NewAPITokenHandler(tokenSvc services.APITokenSvc) *APITokenHandler {
	return &APITokenHandler{
		tokenSvc: tokenSvc,
	}
}

// RegisterAPITokenRoutes registers the API token routes
// @Summary Register API token routes
// @Description Registers all the API token related routes
// @Tags tokens
// @Router /tokens [post]
// @Router /tokens [get]
// @Router /tokens/{id} [delete]
// @Router /tokens [delete]
func RegisterAPITokenRoutes(router *gin.RouterGroup, tokenSvc services.APITokenSvc) {
	handler := NewAPITokenHandler(tokenSvc)

	tokensGroup := router.Group("/tokens")
	{
		tokensGroup.POST("", handler.CreateToken)
		tokensGroup.GET("", handler.ListTokens)
		tokensGroup.DELETE("/:id", handler.RevokeToken)
		tokensGroup.DELETE("", handler.RevokeAllTokens)
	}
}

// CreateToken handles the creation of a new API token
// @Summary Create a new API token
// @Description Creates a new API token for the authenticated user. The token will be shown only once upon creation.
// @Description The token can be used for API authentication by including it in the Authorization header as: `Authorization: Bearer <token>`
// @Tags tokens
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body dto.CreateAPITokenRequest true "Token creation details"
// @Success 201 {object} dto.CreateAPITokenResponse
// @Failure 400 {object} APIErrorResponse
// @Failure 401 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /tokens [post]
func (h *APITokenHandler) CreateToken(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	creatorUserID, ok := middleware.GetUserIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, APIErrorResponse{Message: "Unauthorized"})
		return
	}

	// Parse request body
	var req dto.CreateAPITokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, APIErrorResponse{Message: "Invalid request body: " + err.Error()})
		return
	}

	// Convert expiresIn from seconds to time.Duration if provided
	var expiresIn *time.Duration
	if req.ExpiresIn != nil {
		d := time.Duration(*req.ExpiresIn) * time.Second
		expiresIn = &d
	}

	// Create the token
	tokenStr, token, err := h.tokenSvc.CreateToken(c.Request.Context(), creatorUserID, req.Name, expiresIn)
	if err != nil {
		c.JSON(http.StatusInternalServerError, APIErrorResponse{Message: "Failed to create token: " + err.Error()})
		return
	}

	// Return the token details
	c.JSON(http.StatusCreated, dto.ToCreateAPITokenResponse(tokenStr, *token))
}

// ListTokens handles listing all API tokens for the authenticated user
// @Summary List all API tokens
// @Description Lists all API tokens for the authenticated user. Only returns token metadata, not the actual token values.
// @Tags tokens
// @Produce json
// @Security BearerAuth
// @Success 200 {object} dto.ListAPITokensResponse
// @Failure 401 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /tokens [get]
func (h *APITokenHandler) ListTokens(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	creatorUserID, ok := middleware.GetUserIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, APIErrorResponse{Message: "Unauthorized"})
		return
	}

	// Get tokens for the user
	tokens, err := h.tokenSvc.ListTokens(c.Request.Context(), creatorUserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, APIErrorResponse{Message: "Failed to list tokens: " + err.Error()})
		return
	}

	// Return the tokens
	c.JSON(http.StatusOK, dto.ToAPITokenResponseList(tokens))
}

// RevokeToken handles revoking a specific API token
// @Summary Revoke an API token
// @Description Revokes a specific API token by ID. The token will be immediately invalidated.
// @Description Only the token owner can revoke their own tokens.
// @Tags tokens
// @Produce json
// @Security BearerAuth
// @Param id path string true "Token ID (UUID format)" format(uuid)
// @Success 204 "Token revoked successfully"
// @Failure 400 {object} APIErrorResponse
// @Failure 401 {object} APIErrorResponse
// @Failure 403 {object} APIErrorResponse
// @Failure 404 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /tokens/{id} [delete]
func (h *APITokenHandler) RevokeToken(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	creatorUserID, ok := middleware.GetUserIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, APIErrorResponse{Message: "Unauthorized"})
		return
	}

	// Parse token ID from URL
	tokenID := c.Param("id")
	if _, err := uuid.Parse(tokenID); err != nil {
		c.JSON(http.StatusBadRequest, APIErrorResponse{Message: "Invalid token ID"})
		return
	}

	// Revoke the token
	err := h.tokenSvc.RevokeToken(c.Request.Context(), creatorUserID, tokenID)
	if err != nil {
		if err.Error() == "token not found" {
			c.JSON(http.StatusNotFound, APIErrorResponse{Message: "Token not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, APIErrorResponse{Message: "Failed to revoke token: " + err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// RevokeAllTokens handles revoking all API tokens for the authenticated user
// @Summary Revoke all API tokens
// @Description Revokes all API tokens for the authenticated user. This will immediately invalidate all tokens.
// @Description A new token will need to be generated for API access after this operation.
// @Tags tokens
// @Produce json
// @Security BearerAuth
// @Success 204 "All tokens revoked successfully"
// @Failure 401 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /tokens [delete]
func (h *APITokenHandler) RevokeAllTokens(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	creatorUserID, ok := middleware.GetUserIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, APIErrorResponse{Message: "Unauthorized"})
		return
	}

	// Revoke all tokens for the user
	if err := h.tokenSvc.RevokeAllTokens(c.Request.Context(), creatorUserID); err != nil {
		c.JSON(http.StatusInternalServerError, APIErrorResponse{Message: "Failed to revoke tokens: " + err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

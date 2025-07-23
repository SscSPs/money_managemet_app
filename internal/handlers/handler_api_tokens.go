package handlers

import (
	"net/http"

	"github.com/SscSPs/money_managemet_app/internal/core/ports/services"
	"github.com/SscSPs/money_managemet_app/internal/handlers/dto"
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

// APITokenResponse represents an API token in the API responses
// @Description API token details returned in API responses
type APITokenResponse struct {
	// ID is the unique identifier of the token
	ID string `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	// Name is the user-defined name for the token
	Name string `json:"name" example:"My API Token"`
	// LastUsedAt is the timestamp when the token was last used (optional)
	LastUsedAt *string `json:"lastUsedAt,omitempty" example:"2023-01-01T12:00:00Z"`
	// ExpiresAt is the timestamp when the token will expire (optional)
	ExpiresAt *string `json:"expiresAt,omitempty" example:"2024-01-01T12:00:00Z"`
	// CreatedAt is the timestamp when the token was created
	CreatedAt string `json:"createdAt" example:"2023-01-01T12:00:00Z"`
}

// ListAPITokensResponse represents a list of API tokens
// @Description A list of API tokens
type ListAPITokensResponse []APITokenResponse

// CreateAPITokenRequest represents the request body for creating a new API token
// @Description Request body for creating a new API token
type CreateAPITokenRequest struct {
	// Name is a user-defined name for the token (3-100 characters)
	Name string `json:"name" binding:"required,min=3,max=100" example:"My API Token"`
	// ExpiresIn is the duration in seconds after which the token will expire (optional)
	ExpiresIn *int64 `json:"expiresIn,omitempty" example:"2592000"` // 30 days in seconds
}

// CreateAPITokenResponse represents the response when creating a new API token
// @Description Response returned when a new API token is created
type CreateAPITokenResponse struct {
	// Token is the actual API token (only shown once at creation)
	Token string `json:"token" example:"mma_abc123..."`
	// Details contains the token metadata
	Details APITokenResponse `json:"details"`
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
// @Param request body CreateAPITokenRequest true "Token creation details"
// @Success 201 {object} CreateAPITokenResponse
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

	// Create the token
	tokenStr, token, err := h.tokenSvc.CreateToken(c.Request.Context(), creatorUserID, req.Name, req.ExpiresIn)
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
// @Success 200 {object} ListAPITokensResponse
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

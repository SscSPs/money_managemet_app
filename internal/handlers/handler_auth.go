package handlers

import (
	"net/http"
	"time"

	"github.com/SscSPs/money_managemet_app/internal/middleware" // For GetLoggerFromContext
	"github.com/SscSPs/money_managemet_app/pkg/config"          // For JWT config
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// AuthHandler handles authentication related requests.
type AuthHandler struct {
	cfg *config.Config // Needs config for JWT secret and expiry
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(cfg *config.Config) *AuthHandler {
	return &AuthHandler{cfg: cfg}
}

// LoginRequest represents the expected login payload (simplified).
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse represents the successful login response payload.
type LoginResponse struct {
	Token string `json:"token"`
}

// Login godoc
// @Summary Log in a user
// @Description Authenticates a user (dummy implementation) and returns a JWT.
// @Tags auth
// @Accept  json
// @Produce  json
// @Param   credentials body LoginRequest true "User credentials"
// @Success 200 {object} LoginResponse
// @Failure 400 {object} string "Invalid input"
// @Failure 401 {object} string "Invalid credentials (dummy check)"
// @Failure 500 {object} string "Internal server error (token generation failed)"
// @Router /auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	logger := middleware.GetLoggerFromContext(c)
	var req LoginRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error("Failed to bind JSON for Login", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input format"})
		return
	}

	// --- Dummy Authentication ---
	// In a real app, verify req.Username and req.Password against the database.
	// For now, we'll just check for a specific username and issue a token.
	dummyUserID := "41181354-419f-4847-8405-b10dfd04ccdf" // Hardcoded user ID
	if req.Username != "testuser" || req.Password != "password" {
		logger.Warn("Invalid login attempt", "username", req.Username)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}
	// --- End Dummy Authentication ---

	// Create JWT claims
	expirationTime := time.Now().Add(h.cfg.JWTExpiryDuration)
	claims := &jwt.RegisteredClaims{
		Subject:   dummyUserID,
		ExpiresAt: jwt.NewNumericDate(expirationTime),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		Issuer:    "mma-backend", // Optional: identify the issuer
	}

	// Create token with claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign the token with the secret
	tokenString, err := token.SignedString([]byte(h.cfg.JWTSecret))
	if err != nil {
		logger.Error("Failed to sign JWT token", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	logger.Info("User logged in successfully", "user_id", dummyUserID)
	c.JSON(http.StatusOK, LoginResponse{Token: tokenString})
}

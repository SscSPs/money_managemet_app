package handlers

import (
	"net/http"
	"time"

	"github.com/SscSPs/money_managemet_app/internal/middleware" // For GetLoggerFromCtx
	"github.com/SscSPs/money_managemet_app/pkg/config"          // For JWT config
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// AuthHandler handles authentication related requests.
type AuthHandler struct {
	cfg *config.Config // Needs config for JWT secret and expiry
}

// newAuthHandler creates a new AuthHandler.
func newAuthHandler(cfg *config.Config) *AuthHandler {
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

// login godoc
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
func (h *AuthHandler) login(c *gin.Context) {
	logger := middleware.GetLoggerFromCtx(c.Request.Context())
	var req LoginRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn("Failed to bind JSON for Login", "error", err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	// Dummy authentication - replace with actual user lookup and password check
	// IMPORTANT: Never log passwords in production!
	if req.Username != "user" || req.Password != "password" {
		logger.Warn("Invalid login attempt", "username", req.Username)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
		return
	}

	// --- Dummy User Info (Replace with actual user ID from DB lookup) ---
	dummyUserID := "user-123" // Example user ID
	// ----------------------------------------------------------------------

	// Create JWT claims
	claims := jwt.RegisteredClaims{
		Issuer:    "mma-backend",
		Subject:   dummyUserID,
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(h.cfg.JWTExpiryDuration)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		NotBefore: jwt.NewNumericDate(time.Now()),
	}

	// Create token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign token
	tsignedString, err := token.SignedString([]byte(h.cfg.JWTSecret))
	if err != nil {
		logger.Error("Failed to sign JWT token", "error", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	logger.Info("User logged in successfully", "user_id", dummyUserID)
	c.JSON(http.StatusOK, LoginResponse{Token: tsignedString})
}

// registerAuthRoutes registers authentication related routes (/auth)
func registerAuthRoutes(engine *gin.Engine, cfg *config.Config) {
	authHandler := newAuthHandler(cfg)

	authRoutes := engine.Group("/auth")
	{
		authRoutes.POST("/login", authHandler.login)
		// TODO: Add refresh token route later
	}
}

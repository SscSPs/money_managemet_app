package handlers

import (
	"log/slog"
	"net/http"
	"time"

	// For error checking

	// Use ports
	"github.com/SscSPs/money_managemet_app/internal/dto"
	"github.com/SscSPs/money_managemet_app/internal/middleware" // For logger/user context

	// "github.com/SscSPs/money_managemet_app/internal/models" // Models not needed directly here
	portssvc "github.com/SscSPs/money_managemet_app/internal/core/ports/services" // Use ports services
	"github.com/SscSPs/money_managemet_app/internal/platform/config"              // For JWT config access
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	// "github.com/google/uuid" // Use actual user ID from service
)

// AuthHandler handles authentication related requests.
type AuthHandler struct {
	userService portssvc.UserService // Use interface
	jwtSecret   string
	jwtDuration time.Duration
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(us portssvc.UserService, cfg *config.Config) *AuthHandler { // Use interface
	return &AuthHandler{
		userService: us,
		jwtSecret:   cfg.JWTSecret,         // Store secret
		jwtDuration: cfg.JWTExpiryDuration, // Store duration
	}
}

// LoginRequest defines the structure for the login request body.
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse defines the structure for the login response body.
type LoginResponse struct {
	Token string `json:"token"`
}

// ErrorResponse is a generic error response structure for handlers.
// Moved here or define globally if used by other handlers.
type ErrorResponse struct {
	Error string `json:"error"`
}

// registerAuthRoutes sets up the routes for authentication.
// Pass the instantiated handler.
func registerAuthRoutes(rg *gin.Engine, cfg *config.Config, userService portssvc.UserService) { // Use interface
	h := NewAuthHandler(userService, cfg) // Pass interface

	auth := rg.Group("/api/v1/auth")
	{
		auth.POST("/login", h.Login)
		auth.POST("/register", h.Register)
	}
}

// Login godoc
// @Summary User login
// @Description Authenticates a user and returns a JWT token.
// @Tags auth
// @Accept json
// @Produce json
// @Param login body LoginRequest true "Login Credentials"
// @Success 200 {object} LoginResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request body"})
		return
	}

	// TODO: The UserService currently lacks authentication logic.
	// Assuming a hypothetical AuthenticateUser method exists for now.
	// user, err := h.userService.AuthenticateUser(c.Request.Context(), req.Username, req.Password)
	// if err != nil {
	// 	if errors.Is(err, apperrors.ErrAuthentication) || errors.Is(err, apperrors.ErrNotFound) {
	// 		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Invalid username or password"})
	// 	} else {
	// 		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Login failed"})
	// 	}
	// 	return
	// }

	// --- Placeholder Login Success ---
	// Replace with actual user ID after implementing authentication in UserService
	dummyUserID := "41181354-419f-4847-8405-b10dfd04ccdf"
	if req.Username != "testuser" || req.Password != "password" { // Basic check for placeholder
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Invalid username or password (placeholder check)"})
		return
	}
	// --- End Placeholder ---

	// Generate JWT Token
	claims := jwt.RegisteredClaims{
		Issuer:    "mma-backend",
		Subject:   dummyUserID, // Use actual user.UserID here
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(h.jwtDuration)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		NotBefore: jwt.NewNumericDate(time.Now()),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tsignedString, err := token.SignedString([]byte(h.jwtSecret))
	if err != nil {
		logger := middleware.GetLoggerFromCtx(c.Request.Context())
		logger.Error("Failed to sign JWT token", slog.String("error", err.Error()))
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, LoginResponse{Token: tsignedString})
}

// Register godoc
// @Summary Register new user
// @Description Creates a new user account.
// @Tags auth
// @Accept json
// @Produce json
// @Param register body dto.CreateUserRequest true "User Registration Info"
// @Success 201 {object} dto.UserResponse
// @Failure 400 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse "Conflict (e.g., username exists)"
// @Failure 500 {object} ErrorResponse
// @Router /auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req dto.CreateUserRequest // Use DTO for request binding
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request body: " + err.Error()})
		return
	}

	// creatorUserID := "SYSTEM_SELF_REGISTER" // This variable is unused now based on the interface

	// Call the user service to create the user
	newUser, err := h.userService.CreateUser(c.Request.Context(), req)
	if err != nil {
		logger := middleware.GetLoggerFromCtx(c.Request.Context())
		// TODO: Add specific check for duplicate errors if the repo/service supports it
		// if errors.Is(err, apperrors.ErrDuplicate) {
		// 	 c.JSON(http.StatusConflict, ErrorResponse{Error: "User already exists"})
		// 	 return
		// }
		logger.Error("Failed to register user", slog.String("error", err.Error()))
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to register user"})
		return
	}

	// Return the created user details (using UserResponse DTO)
	c.JSON(http.StatusCreated, dto.ToUserResponse(newUser))
}

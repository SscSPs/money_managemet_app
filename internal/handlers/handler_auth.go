package handlers

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/ulule/limiter/v3"
	"github.com/ulule/limiter/v3/drivers/store/memory"
	limitergin "github.com/ulule/limiter/v3/drivers/middleware/gin"

	// For error checking

	// Use ports
	"github.com/SscSPs/money_managemet_app/internal/dto"
	"github.com/SscSPs/money_managemet_app/internal/middleware" // For logger/user context
	"github.com/SscSPs/money_managemet_app/internal/utils"

	// "github.com/SscSPs/money_managemet_app/internal/models" // Models not needed directly here
	portssvc "github.com/SscSPs/money_managemet_app/internal/core/ports/services" // Use ports services
	"github.com/SscSPs/money_managemet_app/internal/platform/config"              // For JWT config access
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	// "github.com/google/uuid" // Use actual user ID from service
	// Import for error handling
)

// AuthHandler handles authentication related requests.
type AuthHandler struct {
	userService portssvc.UserSvcFacade
	jwtSecret   string
	jwtDuration time.Duration
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(us portssvc.UserSvcFacade, cfg *config.Config) *AuthHandler {
	return &AuthHandler{
		userService: us,
		jwtSecret:   cfg.JWTSecret,         // Store secret
		jwtDuration: cfg.JWTExpiryDuration, // Store duration
	}
}

// ErrorResponse is a generic error response structure for handlers.
// Moved here or define globally if used by other handlers.
type ErrorResponse struct {
	Error string `json:"error"`
}

// registerAuthRoutes sets up the routes for authentication.
// Pass the instantiated handler.
func registerAuthRoutes(rg *gin.Engine, cfg *config.Config, userService portssvc.UserSvcFacade) {
	h := NewAuthHandler(userService, cfg)

	// Define rate limit: 5 requests per minute
	rate, _ := limiter.NewRateFromFormatted("5-M")
	store := memory.NewStore()
	ipLimiter := limiter.New(store, rate)
	limitMiddleware := limitergin.NewMiddleware(ipLimiter)

	auth := rg.Group("/api/v1/auth")
	{
		auth.POST("/login", limitMiddleware, h.Login) // Apply rate limiting middleware here
		auth.POST("/register", h.Register)
	}
}

// Login godoc
// @Summary User login
// @Description Authenticates a user and returns a JWT token.
// @Tags auth
// @Accept json
// @Produce json
// @Param login body dto.LoginRequest true "Login Credentials"
// @Success 200 {object} dto.LoginResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request body"})
		return
	}
	user, err := h.userService.GetUserByUsername(c.Request.Context(), req.Username)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Invalid username or password"})
		return
	}
	if !utils.CheckPasswordHash(req.Password, user.PasswordHash) {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Invalid username or password"})
		return
	}

	// Generate JWT Token
	claims := jwt.RegisteredClaims{
		Issuer:    "mma-backend",
		Subject:   user.UserID,
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

	c.JSON(http.StatusOK, dto.LoginResponse{Token: tsignedString})
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
	var req dto.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request body: " + err.Error()})
		return
	}
	if req.Username == "" || req.Password == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Username and password required"})
		return
	}
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

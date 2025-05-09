package handlers

import (
	"errors" // For error checking
	"log/slog"
	"net/http"
	"time"

	"github.com/ulule/limiter/v3"
	limitergin "github.com/ulule/limiter/v3/drivers/middleware/gin"
	"github.com/ulule/limiter/v3/drivers/store/memory"

	// Use ports
	"github.com/SscSPs/money_managemet_app/internal/apperrors"
	"github.com/SscSPs/money_managemet_app/internal/dto"
	"github.com/SscSPs/money_managemet_app/internal/middleware" // For logger/user context
	"github.com/SscSPs/money_managemet_app/internal/utils"

	// "github.com/SscSPs/money_managemet_app/internal/models" // Models not needed directly here
	portssvc "github.com/SscSPs/money_managemet_app/internal/core/ports/services" // Use ports services
	"github.com/SscSPs/money_managemet_app/internal/platform/config"              // For JWT config access
	"github.com/gin-gonic/gin"
	// "github.com/golang-jwt/jwt/v5" // No longer directly used here
	// "github.com/google/uuid" // Use actual user ID from service
	// Import for error handling
)

// AuthHandler handles authentication related requests.
type AuthHandler struct {
	userService            portssvc.UserSvcFacade
	jwtSecret              string
	jwtDuration            time.Duration
	refreshTokenDuration   time.Duration
	refreshTokenCookieName string
	refreshTokenSecret     string
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(us portssvc.UserSvcFacade, cfg *config.Config) *AuthHandler {
	return &AuthHandler{
		userService:            us,
		jwtSecret:              cfg.JWTSecret,
		jwtDuration:            cfg.JWTExpiryDuration,
		refreshTokenDuration:   cfg.RefreshTokenExpiryDuration,
		refreshTokenCookieName: cfg.RefreshTokenCookieName,
		refreshTokenSecret:     cfg.RefreshTokenSecret,
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
		auth.POST("/refresh_token", h.RefreshToken)
		auth.POST("/logout", middleware.AuthMiddleware(h.jwtSecret), h.Logout) // Protected by auth middleware
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
	token, err := utils.GenerateJWT(user.UserID, h.jwtSecret, h.jwtDuration, "mma-backend")
	if err != nil {
		logger := middleware.GetLoggerFromCtx(c.Request.Context())
		logger.Error("Failed to sign JWT token", slog.String("error", err.Error()))
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to generate token"})
		return
	}

	// Generate Refresh Token
	refreshTokenString, err := utils.GenerateJWT(user.UserID, h.refreshTokenSecret, h.refreshTokenDuration, "mma-backend-refresh")
	if err != nil {
		logger := middleware.GetLoggerFromCtx(c.Request.Context())
		logger.Error("Failed to sign refresh token", slog.String("error", err.Error()))
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to generate token"})
		return
	}

	// Hash and Store Refresh Token
	hashedRefreshToken := utils.HashRefreshToken(refreshTokenString)
	refreshTokenExpiryTime := time.Now().Add(h.refreshTokenDuration)
	if err := h.userService.UpdateRefreshToken(c.Request.Context(), user.UserID, hashedRefreshToken, refreshTokenExpiryTime); err != nil {
		logger := middleware.GetLoggerFromCtx(c.Request.Context())
		logger.Error("Failed to store refresh token hash", slog.String("error", err.Error()), slog.String("user_id", user.UserID))
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to store token"})
		return
	}

	// Set Refresh Token Cookie
	// TODO: Make Secure flag conditional on environment (e.g. cfg.Environment == "production")
	c.SetCookie(h.refreshTokenCookieName, refreshTokenString, int(h.refreshTokenDuration.Seconds()), "/api/v1/auth", "", true, true) // Secure=true, HttpOnly=true
	// Consider SameSite policy, e.g. http.SameSiteLaxMode
	// For gin, you might need to set the cookie fields more directly if SetCookie doesn't support SameSite:
	// http.SetCookie(c.Writer, &http.Cookie{
	// 	Name:     h.refreshTokenCookieName,
	// 	Value:    refreshTokenString,
	// 	Expires:  refreshTokenExpiryTime,
	// 	Path:     "/api/v1/auth",
	// 	Domain:   "", // Set your domain if needed
	// 	Secure:   true, // Should be true in production
	// 	HttpOnly: true,
	// 	SameSite: http.SameSiteLaxMode, // Or http.SameSiteStrictMode
	// })

	c.JSON(http.StatusOK, dto.LoginResponse{Token: token})
}

// RefreshToken godoc
// @Summary Refresh access token
// @Description Refreshes an access token using a valid refresh token provided as an HttpOnly cookie.
// @Tags auth
// @Produce json
// @Success 200 {object} dto.RefreshTokenResponse "New access token"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /auth/refresh_token [post]
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	logger := middleware.GetLoggerFromCtx(c.Request.Context())

	// 1. Extract refresh token from cookie
	refreshTokenString, err := c.Cookie(h.refreshTokenCookieName)
	if err != nil {
		logger.Warn("Refresh token cookie not found", slog.String("error", err.Error()))
		c.AbortWithStatusJSON(http.StatusUnauthorized, ErrorResponse{Error: "Refresh token not found"})
		return
	}

	// 2. Parse and validate the refresh token JWT
	claims, err := utils.ParseAndValidateJWT(refreshTokenString, h.refreshTokenSecret)
	if err != nil {
		logger.Warn("Invalid refresh token", slog.String("error", err.Error()))
		c.AbortWithStatusJSON(http.StatusUnauthorized, ErrorResponse{Error: "Invalid refresh token"})
		return
	}

	// 3. Get User ID from claims
	userID := claims.Subject
	if userID == "" {
		logger.Warn("User ID not found in refresh token claims")
		c.AbortWithStatusJSON(http.StatusUnauthorized, ErrorResponse{Error: "Invalid refresh token claims"})
		return
	}

	// 4. Fetch user from service
	user, err := h.userService.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			logger.Warn("User not found for refresh token", slog.String("user_id", userID))
			c.AbortWithStatusJSON(http.StatusUnauthorized, ErrorResponse{Error: "User not found"})
		} else {
			logger.Error("Failed to get user by ID for refresh token", slog.String("error", err.Error()), slog.String("user_id", userID))
			c.AbortWithStatusJSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to process request"})
		}
		return
	}

	// 5. Validate stored refresh token hash and expiry
	hashedReceivedToken := utils.HashRefreshToken(refreshTokenString)
	if hashedReceivedToken != user.RefreshTokenHash {
		logger.Warn("Refresh token mismatch", slog.String("user_id", userID))
		// This could be a sign of token theft or an old token being used.
		// Optionally, invalidate all refresh tokens for this user here.
		c.AbortWithStatusJSON(http.StatusUnauthorized, ErrorResponse{Error: "Invalid refresh token"})
		return
	}

	if user.RefreshTokenExpiryTime == nil || time.Now().After(*user.RefreshTokenExpiryTime) {
		logger.Warn("Refresh token expired in DB", slog.String("user_id", userID))
		c.AbortWithStatusJSON(http.StatusUnauthorized, ErrorResponse{Error: "Refresh token expired"})
		return
	}

	// 6. Generate new access token
	newAccessToken, err := utils.GenerateJWT(user.UserID, h.jwtSecret, h.jwtDuration, "mma-backend")
	if err != nil {
		logger.Error("Failed to generate new access token", slog.String("error", err.Error()), slog.String("user_id", userID))
		c.AbortWithStatusJSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to generate new token"})
		return
	}

	// 7. Return new access token
	c.JSON(http.StatusOK, dto.RefreshTokenResponse{Token: newAccessToken})
}

// Logout godoc
// @Summary User logout
// @Description Logs out the user and invalidates their refresh token.
// @Tags auth
// @Accept json
// @Produce json
// @Success 200 {object} object
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	logger := middleware.GetLoggerFromCtx(c)

	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists || userID == "" {
		logger.Error("User ID not found in context for logout")
		// This shouldn't happen if auth middleware is working correctly
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "User context not found"})
		return
	}

	// Clear the refresh token from the database
	err := h.userService.ClearRefreshToken(c.Request.Context(), userID)
	if err != nil {
		// Log the error, but proceed to clear cookie anyway. Clearing the cookie is the primary goal for the client.
		logger.Error("Failed to clear refresh token from DB during logout", slog.String("user_id", userID), slog.String("error", err.Error()))
		// Depending on the error, we might not want to abort, as clearing the cookie is still beneficial.
		// If err is critical (e.g., DB down), a 500 might be appropriate. For now, we log and continue.
	}

	// Clear the refresh token cookie
	// Use the same cookie name, path, and domain as when it was set during login.
	// Setting MaxAge to -1 tells the browser to delete the cookie immediately.
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     h.refreshTokenCookieName,
		Value:    "",
		Path:     "/api/v1/auth", // Ensure this matches the path used when setting the cookie
		Domain:   "",             // Set if you used a specific domain
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   c.Request.TLS != nil, // Secure flag based on connection (true if HTTPS)
		SameSite: http.SameSiteLaxMode, // Or http.SameSiteStrictMode, depending on your needs
	})

	logger.Info("User logged out successfully", slog.String("user_id", userID))
	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
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

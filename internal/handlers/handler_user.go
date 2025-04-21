package handlers

import (
	"net/http"

	"github.com/SscSPs/money_managemet_app/internal/core/services"
	"github.com/SscSPs/money_managemet_app/internal/dto"
	"github.com/gin-gonic/gin"
	// TODO: Add logging import
)

type UserHandler struct {
	userService *services.UserService
	// TODO: Inject logger
}

func NewUserHandler(userService *services.UserService) *UserHandler {
	return &UserHandler{userService: userService}
}

// CreateUser godoc
// @Summary Create a new user
// @Description Creates a new user account
// @Tags users
// @Accept  json
// @Produce  json
// @Param   user body dto.CreateUserRequest true "User details"
// @Success 201 {object} dto.UserResponse
// @Failure 400 {object} string "Invalid input"
// @Failure 500 {object} string "Internal server error"
// @Router /users [post]
func (h *UserHandler) CreateUser(c *gin.Context) {
	var createReq dto.CreateUserRequest
	if err := c.ShouldBindJSON(&createReq); err != nil {
		// TODO: Add structured logging
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input: " + err.Error()})
		return
	}

	// TODO: Get actual creator UserID from request context (e.g., JWT claims)
	// This ID represents the user *performing* the action.
	creatorUserID := "temp_admin" // Placeholder

	user, err := h.userService.CreateUser(c.Request.Context(), createReq, creatorUserID)
	if err != nil {
		// TODO: Add structured logging
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, dto.ToUserResponse(user))
}

// GetUser godoc
// @Summary Get a user by ID
// @Description Retrieves details for a specific user by their ID
// @Tags users
// @Accept  json
// @Produce  json
// @Param   userID path string true "User ID"
// @Success 200 {object} dto.UserResponse
// @Failure 404 {object} string "User not found"
// @Failure 500 {object} string "Internal server error"
// @Router /users/{userID} [get]
func (h *UserHandler) GetUser(c *gin.Context) {
	userID := c.Param("userID")

	user, err := h.userService.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		// TODO: Add structured logging
		// Note: The service currently returns (nil, nil) for not found, which isn't ideal here.
		// Ideally, the service should return a specific error type for "not found".
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user: " + err.Error()})
		return
	}

	if user == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, dto.ToUserResponse(user))
}

// TODO: Add other user handlers (List, Update, Delete) later

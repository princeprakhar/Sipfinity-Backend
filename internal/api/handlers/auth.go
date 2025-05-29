package handlers

import (
	"net/http"
	// "strconv"
	"github.com/gin-gonic/gin"
	"github.com/princeprakhar/ecommerce-backend/internal/services"
	"github.com/princeprakhar/ecommerce-backend/internal/utils"
)

type AuthHandler struct {
	authService *services.AuthService
}

func NewAuthHandler(authService *services.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

func (h *AuthHandler) Signup(c *gin.Context) {
	var req services.SignupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.SendValidationError(c, "Invalid request data")
		return
	}

	response, err := h.authService.Signup(req)
	if err != nil {
		utils.SendError(c, http.StatusBadRequest, "Signup failed", err)
		return
	}

	utils.SendSuccess(c, "User created successfully", response)
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req services.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.SendValidationError(c, "Invalid request data")
		return
	}

	response, err := h.authService.Login(req)
	if err != nil {
		utils.SendError(c, http.StatusUnauthorized, "Login failed", err)
		return
	}

	utils.SendSuccess(c, "Login successful", response)
}

func (h *AuthHandler) GetProfile(c *gin.Context) {
	userID := c.GetUint("user_id")
	
	user, err := h.authService.GetUserByID(userID)
	if err != nil {
		utils.SendError(c, http.StatusNotFound, "User not found", err)
		return
	}

	utils.SendSuccess(c, "Profile retrieved successfully", user)
}

func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req services.RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid request",
			"error":   err.Error(),
		})
		return
	}

	response, err := h.authService.RefreshToken(req)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "Token refresh failed",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Token refreshed successfully",
		"data":    response,
	})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	var req services.RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid request",
			"error":   err.Error(),
		})
		return
	}

	if err := h.authService.Logout(req.RefreshToken); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Logout failed",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Logged out successfully",
	})
}
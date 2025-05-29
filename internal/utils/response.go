package utils

import (
	"net/http"
	"github.com/gin-gonic/gin"
)

type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

func SendSuccess(c *gin.Context, message string, data interface{}) {
	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Message: message,
		Data:    data,
	})
}

func SendError(c *gin.Context, statusCode int, message string, err error) {
	response := APIResponse{
		Success: false,
		Message: message,
	}
	
	if err != nil {
		response.Error = err.Error()
	}

	c.JSON(statusCode, response)
}

func SendValidationError(c *gin.Context, message string) {
	SendError(c, http.StatusBadRequest, message, nil)
}

func SendUnauthorized(c *gin.Context, message string) {
	SendError(c, http.StatusUnauthorized, message, nil)
}

func SendForbidden(c *gin.Context, message string) {
	SendError(c, http.StatusForbidden, message, nil)
}

func SendInternalError(c *gin.Context, message string, err error) {
	SendError(c, http.StatusInternalServerError, message, err)
}
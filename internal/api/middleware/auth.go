package middleware

import (
	"strings"
	"github.com/gin-gonic/gin"
	"github.com/princeprakhar/ecommerce-backend/internal/config"
	"github.com/princeprakhar/ecommerce-backend/internal/utils"
)

func AuthMiddleware(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			utils.SendUnauthorized(c, "Authorization header required")
			c.Abort()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			utils.SendUnauthorized(c, "Bearer token required")
			c.Abort()
			return
		}

		claims, err := utils.ValidateToken(tokenString, cfg.JWTSecret)
		if err != nil {
			utils.SendUnauthorized(c, "Invalid token")
			c.Abort()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("user_email", claims.Email)
		c.Set("user_role", claims.Role)
		c.Next()
	}
}

func AdminOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		role := c.GetString("user_role")
		if role != "admin" {
			utils.SendForbidden(c, "Admin access required")
			c.Abort()
			return
		}
		c.Next()
	}
}

func CustomerOrAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		role := c.GetString("user_role")
		if role != "admin" && role != "customer" {
			utils.SendForbidden(c, "Valid user role required")
			c.Abort()
			return
		}
		c.Next()
	}
}
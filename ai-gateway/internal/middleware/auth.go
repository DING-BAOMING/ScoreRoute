package middleware

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"

	"ai-gateway/internal/service"
)

func AuthMiddleware() gin.HandlerFunc {
	authService := service.NewAuthService()

	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "未提供认证令牌"})
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "无效的认证格式"})
			c.Abort()
			return
		}

		claims, err := authService.ValidateToken(parts[1])
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "无效或过期的令牌"})
			c.Abort()
			return
		}

		c.Set("username", claims.Username)
		c.Next()
	}
}

func CORSMiddleware() gin.HandlerFunc {
	allowedOrigins := os.Getenv("CORS_ALLOWED_ORIGINS")

	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		
		// If allowedOrigins is set and not "*", validate the origin
		if allowedOrigins != "" && allowedOrigins != "*" {
			// Check if origin matches allowed list (comma-separated)
			allowedList := strings.Split(allowedOrigins, ",")
			originAllowed := false
			for _, allowed := range allowedList {
				if strings.TrimSpace(allowed) == origin {
					originAllowed = true
					break
				}
			}
			if originAllowed {
				c.Header("Access-Control-Allow-Origin", origin)
			}
		} else if allowedOrigins == "*" {
			// Only set Allow-Origin to * if explicitly configured
			c.Header("Access-Control-Allow-Origin", "*")
		} else if origin != "" {
			// For production without CORS config, don't set Allow-Origin
			// This prevents credential leakage to arbitrary origins
		}
		
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Header("Access-Control-Max-Age", "86400")
		c.Header("Access-Control-Allow-Credentials", "true")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

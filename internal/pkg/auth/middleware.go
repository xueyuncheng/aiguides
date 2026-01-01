package auth

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware 认证中间件
func AuthMiddleware(authService *AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从 Cookie 或 Authorization 头获取 token
		token := c.GetHeader("Authorization")
		if token == "" {
			// 尝试从 Cookie 获取
			token, _ = c.Cookie("auth_token")
		} else {
			// 移除 "Bearer " 前缀
			token = strings.TrimPrefix(token, "Bearer ")
		}

		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			c.Abort()
			return
		}

		// 验证 token
		claims, err := authService.ValidateJWT(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			c.Abort()
			return
		}

		// 将用户信息存储到上下文中
		c.Set("user_id", claims.UserID)
		c.Set("user_email", claims.Email)
		c.Set("user_name", claims.Name)

		c.Next()
	}
}

// OptionalAuthMiddleware 可选的认证中间件（不强制要求认证）
func OptionalAuthMiddleware(authService *AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从 Cookie 或 Authorization 头获取 token
		token := c.GetHeader("Authorization")
		if token == "" {
			// 尝试从 Cookie 获取
			token, _ = c.Cookie("auth_token")
		} else {
			// 移除 "Bearer " 前缀
			token = strings.TrimPrefix(token, "Bearer ")
		}

		if token != "" {
			// 验证 token
			claims, err := authService.ValidateJWT(token)
			if err == nil {
				// 将用户信息存储到上下文中
				c.Set("user_id", claims.UserID)
				c.Set("user_email", claims.Email)
				c.Set("user_name", claims.Name)
			}
		}

		c.Next()
	}
}

// GetUserID 从上下文中获取用户 ID
func GetUserID(c *gin.Context) (string, bool) {
	userID, exists := c.Get("user_id")
	if !exists {
		return "", false
	}
	id, ok := userID.(string)
	return id, ok
}

// GetUserEmail 从上下文中获取用户邮箱
func GetUserEmail(c *gin.Context) (string, bool) {
	email, exists := c.Get("user_email")
	if !exists {
		return "", false
	}
	e, ok := email.(string)
	return e, ok
}

// GetUserName 从上下文中获取用户名
func GetUserName(c *gin.Context) (string, bool) {
	name, exists := c.Get("user_name")
	if !exists {
		return "", false
	}
	n, ok := name.(string)
	return n, ok
}

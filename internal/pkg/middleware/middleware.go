package middleware

import (
	"aiguide/internal/pkg/auth"
	"aiguide/internal/pkg/constant"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// AuthMiddleware 认证中间件
func Auth(db *gorm.DB, authService *auth.AuthService) gin.HandlerFunc {
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

		c.Set(constant.ContextKeyTx, db)

		c.Next()
	}
}

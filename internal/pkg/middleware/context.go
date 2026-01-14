package middleware

import (
	"aiguide/internal/pkg/constant"
	"context"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// GetUserID 从上下文中获取用户 ID
func GetUserID(c *gin.Context) (string, bool) {
	userID, exists := c.Get("user_id")
	if !exists {
		return "", false
	}
	id, ok := userID.(string)
	return id, ok
}

// GetUserEmail 从上下文中获取 Google 用户邮箱
func GetUserEmail(c *gin.Context) (string, bool) {
	email, exists := c.Get("google_user_email")
	if !exists {
		return "", false
	}
	e, ok := email.(string)
	return e, ok
}

// GetGoogleUserID 从上下文中获取 Google 用户 ID
func GetGoogleUserID(c *gin.Context) (string, bool) {
	googleUserID, exists := c.Get("google_user_id")
	if !exists {
		return "", false
	}
	id, ok := googleUserID.(string)
	return id, ok
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

func GetTx(ctx context.Context) (*gorm.DB, bool) {
	tx, exists := ctx.Value(constant.ContextKeyTx).(*gorm.DB)
	if !exists {
		return nil, false
	}
	return tx, true
}

package middleware

import (
	"aiguide/internal/pkg/constant"
	"context"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// GetUserID 从上下文中获取用户 ID
func GetUserID(ctx context.Context) (int, bool) {
	userID, ok := ctx.Value("user_id").(int)

	return userID, ok
}

// GetUserEmail 从上下文中获取 Google 用户邮箱
func GetUserEmail(ctx context.Context) (string, bool) {
	email, ok := ctx.Value("google_user_email").(string)

	return email, ok
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

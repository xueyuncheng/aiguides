package aiguide

import (
	"aiguide/internal/app/aiguide/table"
	"aiguide/internal/pkg/auth"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

// logoutHandler 处理登出
func (a *AIGuide) logoutHandler(c *gin.Context) {
	c.SetCookie("auth_token", "", -1, "/", "", false, true)
	c.JSON(http.StatusOK, gin.H{"message": "logged out"})
}

// getUserHandler 获取当前用户信息
func (a *AIGuide) getUserHandler(c *gin.Context) {
	userID, _ := auth.GetUserID(c)

	var user table.User
	if err := a.db.Where("google_user_id = ?", userID).First(&user).Error; err != nil {
		slog.Error("db.First() error", "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"err": err})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user_id": userID,
		"email":   user.GoogleEmail,
		"name":    user.GoogleName,
		"picture": user.Picture,
	})
}

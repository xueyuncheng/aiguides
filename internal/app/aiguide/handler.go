package aiguide

import (
	"aiguide/internal/pkg/auth"
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
	email, _ := auth.GetUserEmail(c)
	name, _ := auth.GetUserName(c)

	c.JSON(http.StatusOK, gin.H{
		"user_id": userID,
		"email":   email,
		"name":    name,
	})
}

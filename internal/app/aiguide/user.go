package aiguide

import (
	"aiguide/internal/app/aiguide/table"
	"aiguide/internal/pkg/auth"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetUser 获取当前用户信息
func (a *AIGuide) GetUser(c *gin.Context) {
	userID, _ := auth.GetUserID(c)

	var user table.User
	if err := a.db.Where("google_user_id = ?", userID).First(&user).Error; err != nil {
		slog.Error("db.First() error", "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"err": err})
		return
	}

	// Construct avatar URL: use stored avatar endpoint if we have data, otherwise use original URL
	avatarURL := user.Picture
	if len(user.AvatarData) > 0 {
		avatarURL = fmt.Sprintf("/api/auth/avatar/%d", user.ID)
	}

	c.JSON(http.StatusOK, gin.H{
		"user_id": userID,
		"email":   user.GoogleEmail,
		"name":    user.GoogleName,
		"picture": avatarURL,
	})
}

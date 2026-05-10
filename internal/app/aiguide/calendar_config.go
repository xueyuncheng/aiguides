package aiguide

import (
	"aiguide/internal/app/aiguide/table"
	"aiguide/internal/pkg/middleware"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

type CalendarStatusResponse struct {
	Connected bool   `json:"connected"`
	Email     string `json:"email,omitempty"`
}

// GetCalendarStatus 返回当前用户的 Google Calendar 授权状态。
func (a *AIGuide) GetCalendarStatus(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var row struct {
		GoogleOAuthRefreshToken string `gorm:"column:google_oauth_refresh_token"`
		GoogleEmail             string `gorm:"column:google_email"`
	}
	err := a.db.Model(&table.User{}).
		Select("google_oauth_refresh_token", "google_email").
		Where("id = ?", userID).
		First(&row).Error
	if err != nil {
		slog.Error("failed to query user calendar status", "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get user"})
		return
	}

	c.JSON(http.StatusOK, CalendarStatusResponse{
		Connected: row.GoogleOAuthRefreshToken != "",
		Email:     row.GoogleEmail,
	})
}

// RevokeCalendarAccess 清除用户的 Google Calendar refresh token。
func (a *AIGuide) RevokeCalendarAccess(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	if err := a.db.Model(&table.User{}).Where("id = ?", userID).Update("google_oauth_refresh_token", "").Error; err != nil {
		slog.Error("failed to revoke calendar access", "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to revoke access"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Google Calendar access revoked"})
}

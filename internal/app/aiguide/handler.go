package aiguide

import (
	"aiguide/internal/app/aiguide/table"
	"aiguide/internal/pkg/auth"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

const (
	// AvatarCacheMaxAge is the cache duration for avatar images in seconds (1 day)
	AvatarCacheMaxAge = 24 * 60 * 60
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

// getAvatarHandler 获取用户头像
// Note: This endpoint is intentionally public (no auth required) as user avatars
// are typically public information, similar to profile pictures on social platforms.
func (a *AIGuide) getAvatarHandler(c *gin.Context) {
	userIDStr := c.Param("userId")
	userID, err := strconv.ParseUint(userIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	var user table.User
	if err := a.db.Where("id = ?", userID).First(&user).Error; err != nil {
		slog.Error("db.First() error", "err", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	// If no avatar data stored, redirect to original URL
	if len(user.AvatarData) == 0 {
		if user.Picture != "" {
			c.Redirect(http.StatusFound, user.Picture)
		} else {
			c.JSON(http.StatusNotFound, gin.H{"error": "avatar not found"})
		}
		return
	}

	// Validate stored MIME type against allowlist
	mimeType := user.AvatarMimeType
	if mimeType == "" || !isValidImageMimeType(mimeType) {
		// If stored MIME type is invalid, fall back to original URL
		if user.Picture != "" {
			c.Redirect(http.StatusFound, user.Picture)
		} else {
			c.JSON(http.StatusNotFound, gin.H{"error": "avatar not found"})
		}
		return
	}

	// Set cache headers for better performance
	c.Header("Cache-Control", fmt.Sprintf("public, max-age=%d", AvatarCacheMaxAge))
	c.Data(http.StatusOK, mimeType, user.AvatarData)
}

// isValidImageMimeType checks if the MIME type is in the allowlist of safe image types
func isValidImageMimeType(mimeType string) bool {
	allowedTypes := map[string]bool{
		"image/jpeg": true,
		"image/jpg":  true,
		"image/png":  true,
		"image/gif":  true,
		"image/webp": true,
	}
	return allowedTypes[mimeType]
}

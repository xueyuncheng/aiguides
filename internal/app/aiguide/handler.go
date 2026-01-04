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
	c.SetCookie("refresh_token", "", -1, "/api/auth", "", false, true)
	c.JSON(http.StatusOK, gin.H{"message": "logged out"})
}

// refreshTokenHandler 处理刷新令牌请求
func (a *AIGuide) refreshTokenHandler(c *gin.Context) {
	// 从 Cookie 或请求体获取刷新令牌
	refreshToken, err := c.Cookie("refresh_token")
	if err != nil || refreshToken == "" {
		// 尝试从请求体获取
		var req struct {
			RefreshToken string `json:"refresh_token"`
		}
		if err := c.ShouldBindJSON(&req); err != nil || req.RefreshToken == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "refresh token required"})
			return
		}
		refreshToken = req.RefreshToken
	}

	// 验证刷新令牌
	claims, err := a.authService.ValidateRefreshToken(refreshToken)
	if err != nil {
		slog.Error("failed to validate refresh token", "err", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid refresh token"})
		return
	}

	// 重建用户信息用于生成新令牌
	user := &auth.GoogleUser{
		ID:    claims.UserID,
		Email: claims.Email,
		Name:  claims.Name,
	}

	// 生成新的令牌对（包括新的刷新令牌，实现滑动过期）
	tokenPair, err := a.authService.GenerateTokenPair(user)
	if err != nil {
		slog.Error("failed to generate token pair", "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate tokens"})
		return
	}

	// 设置新的访问令牌 cookie (15分钟)
	c.SetCookie("auth_token", tokenPair.AccessToken, 900, "/", "", false, true)
	// 设置新的刷新令牌 cookie (7天) - 滑动过期机制，路径限制为 /api/auth 以减少暴露
	c.SetCookie("refresh_token", tokenPair.RefreshToken, 604800, "/api/auth", "", false, true)

	// 返回新的访问令牌和刷新令牌
	c.JSON(http.StatusOK, gin.H{
		"access_token":  tokenPair.AccessToken,
		"refresh_token": tokenPair.RefreshToken,
		"token_type":    "Bearer",
		"expires_in":    tokenPair.ExpiresIn,
	})
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
	return allowedImageMimeTypes[mimeType]
}

package aiguide

import (
	"aiguide/internal/pkg/auth"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

const (
	// AvatarCacheMaxAge is the cache duration for avatar images in seconds (1 day)
	AvatarCacheMaxAge = 24 * 60 * 60
)

// Logout 处理登出
func (a *AIGuide) Logout(c *gin.Context) {
	secure := a.secureCookie()
	c.SetCookie("auth_token", "", -1, "/", "", secure, true)
	c.SetCookie("refresh_token", "", -1, "/api/auth", "", secure, true)
	c.JSON(http.StatusOK, gin.H{"message": "logged out"})
}

// RefreshToken 处理刷新令牌请求
func (a *AIGuide) RefreshToken(c *gin.Context) {
	// 从 Cookie 或请求体获取刷新令牌
	refreshToken, err := c.Cookie("refresh_token")
	if err != nil || refreshToken == "" {
		// 尝试从请求体获取
		var req struct {
			RefreshToken string `json:"refresh_token"`
		}
		if err := c.ShouldBindJSON(&req); err != nil || req.RefreshToken == "" {
			slog.Error("refresh token required", "cookieErr", err, "hasRequestToken", req.RefreshToken != "")
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
		ID:    claims.GoogleUserID,
		Email: claims.Email,
		Name:  claims.Name,
	}

	// 生成新的令牌对（包括新的刷新令牌，实现滑动过期）
	tokenPair, err := a.authService.GenerateTokenPair(claims.UserID, user)
	if err != nil {
		slog.Error("failed to generate token pair", "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate tokens"})
		return
	}

	// 设置新的访问令牌 cookie (15分钟)
	secure := a.secureCookie()
	c.SetCookie("auth_token", tokenPair.AccessToken, 900, "/", "", secure, true)
	// 设置新的刷新令牌 cookie (7天) - 滑动过期机制，路径限制为 /api/auth 以减少暴露
	c.SetCookie("refresh_token", tokenPair.RefreshToken, 604800, "/api/auth", "", secure, true)

	// 返回新的访问令牌和刷新令牌
	c.JSON(http.StatusOK, gin.H{
		"access_token":  tokenPair.AccessToken,
		"refresh_token": tokenPair.RefreshToken,
		"token_type":    "Bearer",
		"expires_in":    tokenPair.ExpiresIn,
	})
}

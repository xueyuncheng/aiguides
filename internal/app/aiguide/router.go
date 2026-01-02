package aiguide

import (
	"aiguide/internal/app/aiguide/table"
	"aiguide/internal/pkg/auth"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const (
	// AvatarDownloadTimeout is the maximum time allowed for downloading an avatar
	AvatarDownloadTimeout = 10 * time.Second
	// MaxAvatarSizeBytes is the maximum size of avatar images (5MB)
	MaxAvatarSizeBytes = 5 * 1024 * 1024
)

// Allowlist of safe image MIME types
var allowedImageMimeTypes = map[string]bool{
	"image/jpeg": true,
	"image/jpg":  true,
	"image/png":  true,
	"image/gif":  true,
	"image/webp": true,
}

func (a *AIGuide) initRouter(engine *gin.Engine) error {
	// 健康检查
	engine.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// API 路由
	api := engine.Group("/api")

	// 认证路由
	authGroup := api.Group("/auth")
	{
		authGroup.GET("/login/google", a.googleLoginHandler)
		authGroup.GET("/callback/google", a.googleCallbackHandler)
		authGroup.POST("/logout", a.logoutHandler)
		authGroup.GET("/user", auth.AuthMiddleware(a.authService), a.getUserHandler)
		authGroup.GET("/avatar/:userId", a.getAvatarHandler)
	}

	api.POST("/travel/chats/:id", a.agentManager.TravelChatHandler)
	api.POST("/web_summary/chats/:id", a.agentManager.WebSummaryChatHandler)
	api.POST("/assistant/chats/:id", a.agentManager.AssistantChatHandler)
	api.POST("/email_summary/chats/:id", a.agentManager.EmailSummaryChatHandler)

	// 会话管理路由
	agentGroup := api.Group("/:agentId/sessions")
	{
		agentGroup.GET("", a.agentManager.ListSessionsHandler)
		agentGroup.POST("", a.agentManager.CreateSessionHandler)
		agentGroup.GET("/:sessionId/history", a.agentManager.GetSessionHistoryHandler)
		agentGroup.DELETE("/:sessionId", a.agentManager.DeleteSessionHandler)
		agentGroup.POST("/:sessionId/recall", a.agentManager.RecallLastMessageHandler)
	}

	return nil
}

// googleLoginHandler 处理 Google 登录请求
func (a *AIGuide) googleLoginHandler(c *gin.Context) {
	state, err := auth.GenerateStateToken()
	if err != nil {
		slog.Error("failed to generate state token", "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate state"})
		return
	}

	// 保存 state 到 cookie（用于 CSRF 保护）
	c.SetCookie("oauth_state", state, 600, "/", "", false, true)

	// 获取 Google OAuth URL
	url := a.authService.GetAuthURL(state)
	c.JSON(http.StatusOK, gin.H{"url": url})
}

// googleCallbackHandler 处理 Google OAuth 回调
func (a *AIGuide) googleCallbackHandler(c *gin.Context) {
	// 验证 state
	stateCookie, err := c.Cookie("oauth_state")
	if err != nil {
		slog.Error("failed to get state cookie", "err", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid state"})
		return
	}

	state := c.Query("state")
	if state != stateCookie {
		slog.Error("state mismatch")
		c.JSON(http.StatusBadRequest, gin.H{"error": "state mismatch"})
		return
	}

	// 清除 state cookie
	c.SetCookie("oauth_state", "", -1, "/", "", false, true)

	// 获取授权码
	code := c.Query("code")
	if code == "" {
		slog.Error("no code in callback")
		c.JSON(http.StatusBadRequest, gin.H{"error": "no code"})
		return
	}

	// 交换令牌
	token, err := a.authService.ExchangeCode(c.Request.Context(), code)
	if err != nil {
		slog.Error("failed to exchange token", "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to exchange token"})
		return
	}

	// 获取用户信息
	user, err := a.authService.GetGoogleUser(c.Request.Context(), token)
	if err != nil {
		slog.Error("failed to get user info", "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get user info"})
		return
	}

	// 保存用户信息到数据库
	if err := saveUser(a.db, user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save user info"})
		return
	}

	// 生成 JWT
	jwtToken, err := a.authService.GenerateJWT(user)
	if err != nil {
		slog.Error("failed to generate JWT", "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate JWT"})
		return
	}

	// 设置 JWT cookie
	c.SetCookie("auth_token", jwtToken, 86400, "/", "", false, true)

	// 重定向到前端
	frontendURL := a.config.FrontendURL
	if frontendURL == "" {
		frontendURL = "http://localhost:3000"
	}
	c.Redirect(http.StatusFound, frontendURL)
}

func saveUser(db *gorm.DB, user *auth.GoogleUser) error {
	var u table.User
	if err := db.Where("google_user_id = ?", user.ID).First(&u).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			slog.Error("db.First() error", "err", err)
			return fmt.Errorf("db.First() error, err = %w", err)
		}

		// Download avatar image
		avatarData, mimeType, err := downloadAvatar(user.Picture)
		if err != nil {
			slog.Warn("failed to download avatar", "err", err, "url", user.Picture)
			// Continue without avatar data - store the URL anyway
		}

		u = table.User{
			GoogleUserID:   user.ID,
			GoogleEmail:    user.Email,
			GoogleName:     user.Name,
			Picture:        user.Picture,
			AvatarData:     avatarData,
			AvatarMimeType: mimeType,
		}

		if err := db.Create(&u).Error; err != nil {
			slog.Error("db.Create() error", "err", err)
			return fmt.Errorf("db.Create() error, err = %w", err)
		}
	} else {
		// Update existing user info
		oldPictureURL := u.Picture // Store old URL for comparison
		u.GoogleEmail = user.Email
		u.GoogleName = user.Name
		u.Picture = user.Picture

		// Download and update avatar if URL changed or avatar data is missing
		if len(u.AvatarData) == 0 || oldPictureURL != user.Picture {
			avatarData, mimeType, err := downloadAvatar(user.Picture)
			if err != nil {
				slog.Warn("failed to download avatar", "err", err, "url", user.Picture)
				// Continue without updating avatar data
			} else {
				u.AvatarData = avatarData
				u.AvatarMimeType = mimeType
			}
		}

		if err := db.Save(&u).Error; err != nil {
			slog.Error("db.Save() error", "err", err)
			return fmt.Errorf("db.Save() error, err = %w", err)
		}
	}

	return nil
}

// downloadAvatar downloads the avatar image from the given URL
func downloadAvatar(urlStr string) ([]byte, string, error) {
	if urlStr == "" {
		return nil, "", fmt.Errorf("empty avatar URL")
	}

	// Validate URL scheme to prevent SSRF attacks
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return nil, "", fmt.Errorf("invalid URL: %w", err)
	}
	if parsedURL.Scheme != "https" {
		return nil, "", fmt.Errorf("only HTTPS URLs are allowed, got: %s", parsedURL.Scheme)
	}

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), AvatarDownloadTimeout)
	defer cancel()

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create request: %w", err)
	}

	// Execute request
	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("failed to download avatar: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Check Content-Length if available to avoid downloading oversized files
	if resp.ContentLength > 0 && resp.ContentLength > MaxAvatarSizeBytes {
		return nil, "", fmt.Errorf("avatar too large: %d bytes (max %d bytes)", resp.ContentLength, MaxAvatarSizeBytes)
	}

	// Get and validate MIME type from Content-Type header
	mimeType := resp.Header.Get("Content-Type")
	// Remove any charset or other parameters from Content-Type
	if idx := strings.Index(mimeType, ";"); idx != -1 {
		mimeType = strings.TrimSpace(mimeType[:idx])
	}

	if mimeType == "" {
		return nil, "", fmt.Errorf("missing Content-Type header")
	}

	// Validate MIME type is an allowed image format
	if !allowedImageMimeTypes[mimeType] {
		return nil, "", fmt.Errorf("invalid MIME type: %s (expected image/jpeg, image/png, image/gif, or image/webp)", mimeType)
	}

	// Read the response body, detecting truncation
	var buf bytes.Buffer
	// Limit reading to MaxAvatarSizeBytes+1 to detect oversized responses
	limitedReader := io.LimitReader(resp.Body, MaxAvatarSizeBytes+1)
	if _, err := io.Copy(&buf, limitedReader); err != nil {
		return nil, "", fmt.Errorf("failed to read avatar data: %w", err)
	}

	// Check if the image was truncated
	if buf.Len() > MaxAvatarSizeBytes {
		return nil, "", fmt.Errorf("avatar too large: exceeds %d bytes", MaxAvatarSizeBytes)
	}

	return buf.Bytes(), mimeType, nil
}

package aiguide

import (
	"aiguide/internal/app/aiguide/table"
	"aiguide/internal/pkg/auth"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

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

		u = table.User{
			GoogleUserID: user.ID,
			GoogleEmail:  user.Email,
			GoogleName:   user.Name,
			Picture:      user.Picture,
		}

		if err := db.Create(&u).Error; err != nil {
			slog.Error("db.Create() error", "err", err)
			return fmt.Errorf("db.Create() error, err = %w", err)
		}
	} else {
		// Update existing user info
		u.GoogleEmail = user.Email
		u.GoogleName = user.Name
		u.Picture = user.Picture
		if err := db.Save(&u).Error; err != nil {
			slog.Error("db.Save() error", "err", err)
			return fmt.Errorf("db.Save() error, err = %w", err)
		}
	}

	return nil
}

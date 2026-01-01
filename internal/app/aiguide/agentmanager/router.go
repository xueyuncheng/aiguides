package agentmanager

import (
	"aiguide/internal/pkg/auth"
	"aiguide/internal/pkg/constant"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

func (a *AgentManager) initRouter(engine *gin.Engine) error {
	// CORS middleware - restrict to localhost in development
	// TODO: Configure allowed origins based on environment
	engine.Use(func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		// Allow localhost for development
		if origin == "http://localhost:3000" || origin == "http://localhost:18080" {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		}
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// 健康检查
	engine.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// 配置信息（公开）
	engine.GET("/config", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"authentication_enabled": a.config.EnableAuthentication,
		})
	})

	// 认证路由
	if a.authService != nil {
		authGroup := engine.Group("/auth")
		{
			authGroup.GET("/google/login", a.googleLoginHandler)
			authGroup.GET("/google/callback", a.googleCallbackHandler)
			authGroup.POST("/logout", a.logoutHandler)
			authGroup.GET("/user", auth.AuthMiddleware(a.authService), a.getUserHandler)
		}
	}

	// API 路由
	api := engine.Group("/api")

	// 如果启用了认证，应用认证中间件
	if a.config.EnableAuthentication && a.authService != nil {
		api.Use(auth.AuthMiddleware(a.authService))
	}

	api.POST("/travel/chats/:id", a.travelChatHandler)
	api.POST("/web_summary/chats/:id", a.webSummaryChatHandler)
	api.POST("/assistant/chats/:id", a.assistantChatHandler)
	api.POST("/email_summary/chats/:id", a.emailSummaryChatHandler)

	return nil
}

// googleLoginHandler 处理 Google 登录请求
func (a *AgentManager) googleLoginHandler(c *gin.Context) {
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
func (a *AgentManager) googleCallbackHandler(c *gin.Context) {
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

// logoutHandler 处理登出
func (a *AgentManager) logoutHandler(c *gin.Context) {
	c.SetCookie("auth_token", "", -1, "/", "", false, true)
	c.JSON(http.StatusOK, gin.H{"message": "logged out"})
}

// getUserHandler 获取当前用户信息
func (a *AgentManager) getUserHandler(c *gin.Context) {
	userID, _ := auth.GetUserID(c)
	email, _ := auth.GetUserEmail(c)
	name, _ := auth.GetUserName(c)

	c.JSON(http.StatusOK, gin.H{
		"user_id": userID,
		"email":   email,
		"name":    name,
	})
}

// travelChatHandler 处理旅游 agent 的聊天请求
func (a *AgentManager) travelChatHandler(ctx *gin.Context) {
	a.HandleAgentChat(ctx, constant.AppNameTravel)
}

// webSummaryChatHandler 处理网页总结 agent 的聊天请求
func (a *AgentManager) webSummaryChatHandler(ctx *gin.Context) {
	a.HandleAgentChat(ctx, constant.AppNameWebSummary)
}

func (a *AgentManager) assistantChatHandler(ctx *gin.Context) {
	a.HandleAgentChat(ctx, constant.AppNameAssistant)
}

func (a *AgentManager) emailSummaryChatHandler(ctx *gin.Context) {
	a.HandleAgentChat(ctx, constant.AppNameEmailSummary)
}

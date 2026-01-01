package aiguide

import (
	"aiguide/internal/pkg/auth"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

func (a *AIGuide) initRouter(engine *gin.Engine) error {
	// // CORS middleware - restrict to localhost in development
	// // TODO: Configure allowed origins based on environment
	// engine.Use(func(c *gin.Context) {
	// 	origin := c.Request.Header.Get("Origin")
	// 	// Allow localhost for development
	// 	if origin == "http://localhost:3000" || origin == "http://localhost:18080" {
	// 		c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
	// 		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
	// 	}
	// 	c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
	// 	c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

	// 	if c.Request.Method == "OPTIONS" {
	// 		c.AbortWithStatus(204)
	// 		return
	// 	}

	// 	c.Next()
	// })

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

	// // 如果启用了认证，应用认证中间件
	// if a.config.EnableAuthentication && a.authService != nil {
	// 	api.Use(auth.AuthMiddleware(a.authService))
	// }

	api.POST("/travel/chats/:id", a.agentManager.TravelChatHandler)
	api.POST("/web_summary/chats/:id", a.agentManager.WebSummaryChatHandler)
	api.POST("/assistant/chats/:id", a.agentManager.AssistantChatHandler)
	api.POST("/email_summary/chats/:id", a.agentManager.EmailSummaryChatHandler)

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

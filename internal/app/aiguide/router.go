package aiguide

import (
	"aiguide/internal/pkg/auth"
	"net/http"

	"github.com/gin-gonic/gin"
)

func (a *AIGuide) initRouter(engine *gin.Engine) error {
	// API 路由
	api := engine.Group("/api")

	// 公开路由 (无需认证)
	// 健康检查
	api.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	api.GET("/auth/login/google", a.GoogleLogin)
	api.GET("/auth/callback/google", a.GoogleCallback)
	api.POST("/auth/logout", a.Logout)
	api.POST("/auth/refresh", a.RefreshToken)
	api.GET("/auth/avatar/:userId", a.GetAvatar)

	// 应用认证中间件到后续所有接口
	api.Use(auth.AuthMiddleware(a.authService))

	// 需要认证的用户信息接口
	api.GET("/auth/user", a.GetUser)

	// 邮件服务器配置路由
	api.POST("/email-servers", a.CreateEmailServer)
	api.GET("/email-servers", a.ListEmailServers)
	api.GET("/email-servers/:id", a.GetEmailServer)
	api.PUT("/email-servers/:id", a.UpdateEmailServer)
	api.DELETE("/email-servers/:id", a.DeleteEmailServer)

	// Agent 聊天路由
	api.POST("/assistant/chats/:id", a.assistant.Chat)

	// 会话管理路由
	agentGroup := api.Group("/:agentId/sessions")
	{
		agentGroup.GET("", a.assistant.ListSessions)
		agentGroup.POST("", a.assistant.CreateSession)
		agentGroup.GET("/:sessionId/history", a.assistant.GetSessionHistory)
		agentGroup.DELETE("/:sessionId", a.assistant.DeleteSession)
	}

	return nil
}

package aiguide

import (
	"aiguide/internal/app/aiguide/setting"
	"aiguide/internal/pkg/middleware"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
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
	api.Use(middleware.Auth(a.db, a.authService))

	// 需要认证的用户信息接口
	api.GET("/auth/user", a.GetUser)

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

	registerSettingRoutes(a.db, api)

	return nil
}

func registerSettingRoutes(db *gorm.DB, api *gin.RouterGroup) {
	s := setting.New(db)
	emailServerConfig := api.Group("/email-servers")
	{
		emailServerConfig.POST("", s.CreateEmailServer)
		emailServerConfig.GET("", s.ListEmailServers)
		emailServerConfig.GET("/:id", s.GetEmailServer)
		emailServerConfig.PUT("/:id", s.UpdateEmailServer)
		emailServerConfig.DELETE("/:id", s.DeleteEmailServer)
	}
}

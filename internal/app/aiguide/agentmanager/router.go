package agentmanager

import (
	"aiguide/internal/pkg/constant"

	"github.com/gin-gonic/gin"
)

func (a *AgentManager) initRouter(engine *gin.Engine) error {
	api := engine.Group("/api")

	// 聊天接口
	api.POST("/travel/chats/:id", a.travelChatHandler)
	api.POST("/web_summary/chats/:id", a.webSummaryChatHandler)
	api.POST("/assistant/chats/:id", a.assistantChatHandler)
	api.POST("/email_summary/chats/:id", a.emailSummaryChatHandler)

	// 会话管理接口
	api.GET("/:agentId/sessions", a.listSessionsHandler)
	api.GET("/:agentId/sessions/:sessionId/history", a.getSessionHistoryHandler)
	api.POST("/:agentId/sessions", a.createSessionHandler)
	api.DELETE("/:agentId/sessions/:sessionId", a.deleteSessionHandler)

	return nil
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

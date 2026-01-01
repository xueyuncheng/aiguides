package agentmanager

import (
	"aiguide/internal/pkg/constant"

	"github.com/gin-gonic/gin"
)

func (a *AgentManager) initRouter(engine *gin.Engine) error {
	api := engine.Group("/api")

	api.POST("/travel/chats/:id", a.travelChatHandler)
	api.POST("/web_summary/chats/:id", a.webSummaryChatHandler)
	api.POST("/assistant/chats/:id", a.assistantChatHandler)
	api.POST("/email_summary/chats/:id", a.emailSummaryChatHandler)

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

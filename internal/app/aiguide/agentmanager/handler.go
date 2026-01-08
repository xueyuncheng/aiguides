package agentmanager

import (
	"aiguide/internal/pkg/constant"

	"github.com/gin-gonic/gin"
)

// travelChatHandler 处理旅游 agent 的聊天请求
func (a *AgentManager) TravelChatHandler(ctx *gin.Context) {
	a.HandleAgentChat(ctx, constant.AppNameTravel)
}

// webSummaryChatHandler 处理网页总结 agent 的聊天请求
func (a *AgentManager) WebSummaryChatHandler(ctx *gin.Context) {
	a.HandleAgentChat(ctx, constant.AppNameWebSummary)
}

func (a *AgentManager) AssistantChatHandler(ctx *gin.Context) {
	a.HandleAgentChat(ctx, constant.AppNameAssistant)
}

func (a *AgentManager) EmailSummaryChatHandler(ctx *gin.Context) {
	a.HandleAgentChat(ctx, constant.AppNameEmailSummary)
}

func (a *AgentManager) ImageGenChatHandler(ctx *gin.Context) {
	a.HandleAgentChat(ctx, constant.AppNameImageGen)
}

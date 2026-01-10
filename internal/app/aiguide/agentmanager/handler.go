package agentmanager

import (
	"aiguide/internal/pkg/constant"

	"github.com/gin-gonic/gin"
)

func (a *AgentManager) AssistantChatHandler(ctx *gin.Context) {
	a.HandleAgentChat(ctx, constant.AppNameAssistant)
}

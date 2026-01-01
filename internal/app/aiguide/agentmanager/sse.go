package agentmanager

import (
	"aiguide/internal/pkg/constant"
	"errors"
	"log/slog"

	"github.com/gin-gonic/gin"
	"google.golang.org/adk/agent"
	"google.golang.org/adk/runner"
	"google.golang.org/adk/session"
	"google.golang.org/genai"
	"gorm.io/gorm"
)

// ChatRequest 定义通用的聊天请求结构
type ChatRequest struct {
	UserID    string `json:"user_id"`
	SessionID string `json:"session_id"`
	Message   string `json:"message"`
}

// HandleAgentChat 处理通用的 agent 聊天请求，支持 SSE 流式响应
// appName: 应用名称（如 "travel", "email" 等）
// runnerName: runner 的名称（用于从 runnerMap 中获取）
func (a *AgentManager) HandleAgentChat(ctx *gin.Context, appName constant.AppName) {
	runner, ok := a.runnerMap[appName]
	if !ok {
		slog.Error("runner name not found", "runner name", appName)
		ctx.JSON(500, gin.H{"error": appName + " runner not found"})
		return
	}

	var req ChatRequest
	if err := ctx.BindJSON(&req); err != nil {
		slog.Error("ctx.BindJSON() error", "err", err)
		ctx.JSON(400, gin.H{"error": "invalid request"})
		return
	}

	userID := req.UserID
	sessionID := req.SessionID
	message := genai.NewContentFromText(req.Message, genai.RoleUser)

	// 检查或创建 session
	if err := a.ensureSession(ctx, appName.String(), userID, sessionID); err != nil {
		slog.Error("ensureSession() error", "err", err)
		ctx.JSON(500, gin.H{"error": err.Error()})
		return
	}

	// 配置流式响应
	runConfig := agent.RunConfig{
		StreamingMode: agent.StreamingModeSSE,
	}

	// 设置 SSE 响应
	a.setupSSEResponse(ctx)

	// 处理流式事件
	a.streamAgentEvents(ctx, runner, userID, sessionID, message, runConfig)
}

// ensureSession 确保 session 存在，不存在则创建
func (a *AgentManager) ensureSession(ctx *gin.Context, appName, userID, sessionID string) error {
	sessionGetReq := &session.GetRequest{
		AppName:   appName,
		UserID:    userID,
		SessionID: sessionID,
	}

	if _, err := a.session.Get(ctx, sessionGetReq); err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		// Session 不存在，创建新的 session
		sessionCreateReq := &session.CreateRequest{
			AppName:   appName,
			UserID:    userID,
			SessionID: sessionID,
			State:     map[string]any{},
		}

		if _, err := a.session.Create(ctx, sessionCreateReq); err != nil {
			return err
		}
	}

	return nil
}

// setupSSEResponse 设置 SSE 响应头
func (a *AgentManager) setupSSEResponse(ctx *gin.Context) {
	ctx.Writer.Header().Set("Content-Type", "text/event-stream")
	ctx.Writer.Header().Set("Cache-Control", "no-cache")
	ctx.Writer.Header().Set("Connection", "keep-alive")
	ctx.Writer.Header().Set("Access-Control-Allow-Origin", "*")
	ctx.Writer.Header().Set("Content-Encoding", "none")

	// 强制把响应头发送给客户端，建立连接
	ctx.Writer.Flush()
}

// streamAgentEvents 处理 agent 的流式事件并发送给客户端
func (a *AgentManager) streamAgentEvents(
	ctx *gin.Context,
	runner *runner.Runner,
	userID, sessionID string,
	message *genai.Content,
	runConfig agent.RunConfig,
) {
	for event, err := range runner.Run(ctx, userID, sessionID, message, runConfig) {
		// 检查客户端是否断开连接
		select {
		case <-ctx.Request.Context().Done():
			return // 客户端断开，停止处理
		default:
		}

		if err != nil {
			// 发送错误事件
			slog.Error("runner.Run() error", "err", err)
			ctx.SSEvent("error", gin.H{"error": err.Error()})
			ctx.Writer.Flush()
			return
		}

		if event == nil {
			continue
		}

		// 提取事件中的文本内容
		if event.LLMResponse.Content != nil && len(event.LLMResponse.Content.Parts) > 0 {
			for _, part := range event.LLMResponse.Content.Parts {
				if part.Text != "" {
					// 发送数据事件
					ctx.SSEvent("data", gin.H{"content": part.Text})
					ctx.Writer.Flush()
				}
			}
		}
	}

	// 循环结束后，发送结束标记
	ctx.SSEvent("stop", gin.H{"status": "done"})
	ctx.Writer.Flush()
}

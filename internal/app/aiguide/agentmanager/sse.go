package agentmanager

import (
	"aiguide/internal/app/aiguide/table"
	"aiguide/internal/pkg/constant"
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/gin-gonic/gin"
	"google.golang.org/adk/agent"
	"google.golang.org/adk/model"
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

	// 异步生成标题
	go func() {
		// 创建一个新的 context，因为 request context 会被取消
		bgCtx := context.Background()
		if err := a.generateTitle(bgCtx, appName.String(), userID, sessionID, req.Message); err != nil {
			slog.Error("generateTitle failed", "err", err)
		}
	}()
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
			slog.Error("session.Get() error", "err", err)
			return fmt.Errorf("session.Get() error, err = %w", err)
		}

		// Session 不存在，创建新的 session
		sessionCreateReq := &session.CreateRequest{
			AppName:   appName,
			UserID:    userID,
			SessionID: sessionID,
			State:     map[string]any{},
		}

		if _, err := a.session.Create(ctx, sessionCreateReq); err != nil {
			slog.Error("session.Create() error", "err", err)
			return fmt.Errorf("session.Create() error, err = %w", err)
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

// hasTextContent checks if an event contains any text content
func hasTextContent(event *session.Event) bool {
	if event == nil || event.LLMResponse.Content == nil || len(event.LLMResponse.Content.Parts) == 0 {
		return false
	}
	for _, part := range event.LLMResponse.Content.Parts {
		if part.Text != "" {
			return true
		}
	}
	return false
}

// streamAgentEvents 处理 agent 的流式事件并发送给客户端
// 只发送最后一个 agent 的输出结果给前端
//
// 在多 agent 场景下（如 AssistantAgent 包含 SearchAgent 和 FactCheckAgent），
// 该函数会收集所有 agent 的输出，然后只将最后一个 agent 的结果发送给前端。
// 这样可以避免中间 agent 的输出干扰用户，只展示最终结果。
//
// 注意：这种实现方式会等待所有 agent 完成后才开始发送数据，
// 因此会牺牲一些实时性，但可以确保用户只看到最终的、最完整的答案。
func (a *AgentManager) streamAgentEvents(
	ctx *gin.Context,
	runner *runner.Runner,
	userID, sessionID string,
	message *genai.Content,
	runConfig agent.RunConfig,
) {
	// 收集所有事件，按照 Author 分组
	eventsByAuthor := make(map[string][]*session.Event)
	var authorOrder []string // 记录 author 出现的顺序

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

		// 只收集包含文本内容的事件
		if hasTextContent(event) {
			author := event.Author
			if author == "" {
				author = "unnamed-agent"
			}

			// 如果这是一个新的 author，记录其顺序
			if _, exists := eventsByAuthor[author]; !exists {
				authorOrder = append(authorOrder, author)
			}

			eventsByAuthor[author] = append(eventsByAuthor[author], event)
		}
	}

	// 只发送最后一个 author 的事件（这是最终的 agent 输出）
	if len(authorOrder) > 0 {
		lastAuthor := authorOrder[len(authorOrder)-1]
		slog.Info("Streaming events from final agent", "author", lastAuthor, "total_agents", len(authorOrder))

		for _, event := range eventsByAuthor[lastAuthor] {
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
	} else {
		// 如果没有收集到任何内容，记录警告
		slog.Warn("No content collected from any agent", "userID", userID, "sessionID", sessionID)
	}

	// 循环结束后，发送结束标记
	ctx.SSEvent("stop", gin.H{"status": "done"})
	ctx.Writer.Flush()
}

// generateTitle 生成会话标题
func (a *AgentManager) generateTitle(ctx context.Context, appName, userID, sessionID, firstMessage string) error {
	// 1. 检查数据库中是否已有标题
	var meta table.SessionMeta
	if err := a.db.Where("session_id = ?", sessionID).First(&meta).Error; err == nil && meta.Title != "" {
		return nil
	}

	// 2. 调用 LLM 生成标题
	prompt := "Please generate a short title (3-5 words) for this conversation based on the user's message: " + firstMessage
	content := genai.NewContentFromText(prompt, genai.RoleUser)

	req := &model.LLMRequest{
		Contents: []*genai.Content{content},
	}

	for resp, err := range a.model.GenerateContent(ctx, req, false) {
		if err != nil {
			slog.Error("a.model.GenerateContent() error", "err", err)
			return fmt.Errorf("a.model.GenerateContent() error, err = %w", err)
		}

		// Check response content
		if resp == nil || resp.Content == nil || len(resp.Content.Parts) == 0 {
			continue
		}

		generatedTitle := ""
		for _, part := range resp.Content.Parts {
			if part.Text != "" {
				generatedTitle = part.Text
				break
			}
		}

		if generatedTitle == "" {
			continue
		}

		// 3. 保存到数据库
		meta = table.SessionMeta{
			SessionID: sessionID,
			Title:     generatedTitle,
		}

		if err := a.db.Save(&meta).Error; err != nil {
			slog.Error("db.Save() error", "err", err)
			return fmt.Errorf("db.Save() error, err = %w", err)
		}

		// 只要有一个结果就返回（非流式）
		return nil
	}

	return errors.New("no content generated")
}

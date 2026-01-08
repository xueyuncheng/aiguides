package agentmanager

import (
	"aiguide/internal/app/aiguide/table"
	"aiguide/internal/pkg/constant"
	"aiguide/internal/pkg/tools"
	"context"
	"encoding/json"
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

	// 异步生成标题
	go func() {
		// 创建一个新的 context，因为 request context 会被取消
		bgCtx := context.Background()
		if err := a.generateTitle(bgCtx, sessionID, req.Message); err != nil {
			slog.Error("generateTitle failed", "err", err)
		}
	}()

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
			slog.Info("client abort connection", "err", ctx.Request.Context().Err())
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

		// 提取事件中的文本内容和图片数据
		if event.LLMResponse.Content != nil && len(event.LLMResponse.Content.Parts) > 0 {
			for _, part := range event.LLMResponse.Content.Parts {
				// 处理文本内容
				if part.Text != "" {
					data := gin.H{
						"author":     event.Author,
						"content":    part.Text,
						"is_thought": part.Thought,
					}
					ctx.SSEvent("data", data)
					ctx.Writer.Flush()
				}

				// 处理 FunctionResponse 中的图片数据
				if part.FunctionResponse != nil {
					response := part.FunctionResponse.Response
					// 将 map[string]any 转换为 ImageGenOutput
					var output tools.ImageGenOutput
					if jsonData, err := json.Marshal(response); err == nil {
						if err := json.Unmarshal(jsonData, &output); err == nil && output.Success && len(output.Images) > 0 {
							data := gin.H{
								"author": "model",
								"images": output.Images,
							}
							ctx.SSEvent("data", data)
							ctx.Writer.Flush()
						}
					}
				}
			}
		}
	}

	// 循环结束后，发送结束标记
	ctx.SSEvent("stop", gin.H{"status": "done"})
	ctx.Writer.Flush()
}

const titlePromptTemplate = `Generate a concise title for this conversation based on the user's message: "%s". 
Rules: 
1. Use the same language as the user's message. 
2. Do NOT use any Markdown formatting (no bold, no italics). 
3. Do NOT use quotes in the title. 
4. Output only the title text.`

// generateTitle 生成会话标题
func (a *AgentManager) generateTitle(ctx context.Context, sessionID, firstMessage string) error {
	// 1. 检查数据库中是否已有标题
	var meta table.SessionMeta
	if err := a.db.Where("session_id = ?", sessionID).First(&meta).Error; err == nil && meta.Title != "" {
		return nil
	}

	// 2. 调用 LLM 生成标题
	prompt := fmt.Sprintf(titlePromptTemplate, firstMessage)
	content := genai.NewContentFromText(prompt, genai.RoleUser)

	req := &model.LLMRequest{
		// Model:    "gemini-2.0-flash",
		Contents: []*genai.Content{content},
		Config: &genai.GenerateContentConfig{
			ThinkingConfig: &genai.ThinkingConfig{
				IncludeThoughts: false,
			},
		},
	}

	generatedTitle := ""
	for resp, err := range a.model.GenerateContent(ctx, req, false) {
		if err != nil {
			slog.Error("a.model.GenerateContent() error", "err", err)
			return fmt.Errorf("a.model.GenerateContent() error, err = %w", err)
		}

		// Check response content
		if resp == nil || resp.Content == nil || len(resp.Content.Parts) == 0 {
			continue
		}

		for _, part := range resp.Content.Parts {
			if part.Thought {
				continue
			}
			if part.Text != "" {
				generatedTitle += part.Text
			}
		}
	}

	if generatedTitle == "" {
		return errors.New("no content generated")
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

	return nil
}

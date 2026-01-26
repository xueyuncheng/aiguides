package assistant

import (
	"aiguide/internal/app/aiguide/table"
	"aiguide/internal/pkg/constant"
	"aiguide/internal/pkg/tools"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

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
	UserID    int      `json:"user_id"`
	SessionID string   `json:"session_id"`
	Message   string   `json:"message"`
	Images    []string `json:"images,omitempty"`
	FileNames []string `json:"file_names,omitempty"` // 文件名列表，与 Images 数组对应
}

const (
	// Limits align with frontend validation to prevent oversized uploads.
	maxUserImageSizeBytes = 5 * 1024 * 1024
	maxUserPDFSizeBytes   = 20 * 1024 * 1024
	maxUserFileCount      = 4
	pdfMimeType           = "application/pdf"
)

var allowedUserUploadMimeTypes = map[string]bool{
	"image/jpeg":      true,
	"image/png":       true,
	"image/gif":       true,
	"image/webp":      true,
	"application/pdf": true,
}

var imageMimeAliases = map[string]string{
	"image/jpg": "image/jpeg",
}

// Chat 处理通用的 agent 聊天请求，支持 SSE 流式响应
// appName: 应用名称（如 "travel", "email" 等）
// runnerName: runner 的名称（用于从 runnerMap 中获取）
func (a *Assistant) Chat(ctx *gin.Context) {
	var req ChatRequest
	if err := ctx.BindJSON(&req); err != nil {
		slog.Error("ctx.BindJSON() error", "err", err)
		ctx.JSON(400, gin.H{"error": "invalid request"})
		return
	}

	userID := strconv.Itoa(req.UserID)
	sessionID := req.SessionID
	// Only trim leading and trailing newlines, preserving internal line breaks
	messageText := strings.Trim(req.Message, "\n\r")

	if len(req.Images) > maxUserFileCount {
		slog.Error("too many images", "count", len(req.Images), "max", maxUserFileCount)
		ctx.JSON(400, gin.H{"error": fmt.Sprintf("too many images (max %d)", maxUserFileCount)})
		return
	}

	parts := make([]*genai.Part, 0, 1+len(req.Images))

	// 如果有文件名，添加到消息文本前面作为元数据
	actualMessageText := messageText
	if len(req.FileNames) > 0 && len(req.FileNames) == len(req.Images) {
		fileNamesJSON, _ := json.Marshal(req.FileNames)
		actualMessageText = fmt.Sprintf("<!-- FILE_NAMES: %s -->\n%s", fileNamesJSON, messageText)
	}

	if actualMessageText != "" {
		parts = append(parts, genai.NewPartFromText(actualMessageText))
	}

	for _, image := range req.Images {
		imageBytes, mimeType, err := parseDataURI(image)
		if err != nil {
			slog.Error("parseDataURI error", "err", err)
			ctx.JSON(400, gin.H{"error": err.Error()})
			return
		}
		parts = append(parts, genai.NewPartFromBytes(imageBytes, mimeType))
	}

	if len(parts) == 0 {
		slog.Error("message or images required")
		ctx.JSON(400, gin.H{"error": "message or images required"})
		return
	}

	message := genai.NewContentFromParts(parts, genai.RoleUser)
	titleMessage := messageText
	if titleMessage == "" && len(req.Images) > 0 {
		titleMessage = fmt.Sprintf("用户发送了 %d 个文件", len(req.Images))
	}

	// 检查或创建 session
	if err := a.ensureSession(ctx, userID, sessionID); err != nil {
		ctx.JSON(500, gin.H{"error": err.Error()})
		return
	}

	// 异步生成标题
	go func() {
		// 创建一个新的 context，因为 request context 会被取消
		bgCtx := context.Background()
		if err := a.generateTitle(bgCtx, sessionID, titleMessage); err != nil {
			slog.Error("a.generateTitle failed", "err", err)
		}
	}()

	// 配置流式响应
	runConfig := agent.RunConfig{
		StreamingMode: agent.StreamingModeSSE,
	}

	// 设置 SSE 响应
	a.setupSSEResponse(ctx)

	ctx.Set(constant.ContextKeySessionID, sessionID)

	a.streamAgentEvents(ctx, a.runner, userID, sessionID, message, runConfig)
}

func parseDataURI(dataURI string) ([]byte, string, error) {
	if dataURI == "" {
		slog.Error("empty file data")
		return nil, "", errors.New("empty file data")
	}
	if !strings.HasPrefix(dataURI, "data:") {
		slog.Error("invalid data URI", "prefix", "missing data:")
		return nil, "", errors.New("invalid data URI")
	}

	parts := strings.SplitN(dataURI, ",", 2)
	if len(parts) != 2 {
		slog.Error("invalid data URI", "parts_count", len(parts))
		return nil, "", errors.New("invalid data URI")
	}

	header := strings.TrimPrefix(parts[0], "data:")
	payload := parts[1]
	if header == "" || payload == "" {
		slog.Error("invalid data URI", "header_empty", header == "", "payload_empty", payload == "")
		return nil, "", errors.New("invalid data URI")
	}

	headerParts := strings.Split(header, ";")
	mimeType := strings.TrimSpace(headerParts[0])
	if mimeType == "" {
		slog.Error("missing file MIME type")
		return nil, "", errors.New("missing file MIME type")
	}
	if alias, ok := imageMimeAliases[mimeType]; ok {
		mimeType = alias
	}

	isBase64 := false
	for _, part := range headerParts[1:] {
		if strings.TrimSpace(part) == "base64" {
			isBase64 = true
			break
		}
	}
	if !isBase64 {
		slog.Error("file data must be base64 encoded")
		return nil, "", errors.New("file data must be base64 encoded")
	}
	if !allowedUserUploadMimeTypes[mimeType] {
		slog.Error("unsupported file type", "mime_type", mimeType)
		return nil, "", fmt.Errorf("unsupported file type: %s", mimeType)
	}

	decoded, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		slog.Error("base64.StdEncoding.DecodeString() error", "err", err)
		return nil, "", fmt.Errorf("invalid base64 data: %w", err)
	}
	if len(decoded) == 0 {
		slog.Error("empty file data after decoding")
		return nil, "", errors.New("empty file data")
	}
	maxSize := maxUserImageSizeBytes
	if mimeType == pdfMimeType {
		if !bytes.HasPrefix(decoded, []byte("%PDF-")) {
			slog.Error("invalid PDF data")
			return nil, "", errors.New("invalid PDF data")
		}
		maxSize = maxUserPDFSizeBytes
	}
	if len(decoded) > maxSize {
		slog.Error("file size exceeds limit", "size", len(decoded), "max", maxSize)
		return nil, "", fmt.Errorf("file size exceeds %d bytes", maxSize)
	}

	return decoded, mimeType, nil
}

// ensureSession 确保 session 存在，不存在则创建
func (a *Assistant) ensureSession(ctx *gin.Context, userID, sessionID string) error {
	sessionGetReq := &session.GetRequest{
		AppName:   constant.AppNameAssistant.String(),
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
			AppName:   constant.AppNameAssistant.String(),
			UserID:    userID,
			SessionID: sessionID,
			State:     map[string]any{},
		}

		if _, err := a.session.Create(ctx, sessionCreateReq); err != nil {
			slog.Error("session.Create() error", "err", err)
			return fmt.Errorf("session.Create() error, err = %w", err)
		}

		// 创建后验证 session 是否成功保存
		// 这有助于捕捉数据库同步或创建失败的情况
		if _, err := a.session.Get(ctx, sessionGetReq); err != nil {
			slog.Error("session.Get() after create error", "err", err, "appName", constant.AppNameAssistant.String(), "userID", userID, "sessionID", sessionID)
			return fmt.Errorf("session.Get() after create error, err = %w", err)
		}
	}

	return nil
}

// setupSSEResponse 设置 SSE 响应头
func (a *Assistant) setupSSEResponse(ctx *gin.Context) {
	ctx.Writer.Header().Set("Content-Type", "text/event-stream")
	ctx.Writer.Header().Set("Cache-Control", "no-cache")
	ctx.Writer.Header().Set("Connection", "keep-alive")
	ctx.Writer.Header().Set("Access-Control-Allow-Origin", "*")
	ctx.Writer.Header().Set("Content-Encoding", "none")

	// 强制把响应头发送给客户端，建立连接
	ctx.Writer.Flush()
}

// streamAgentEvents 处理 agent 的流式事件并发送给客户端
func (a *Assistant) streamAgentEvents(
	ctx *gin.Context,
	runner *runner.Runner,
	userID, sessionID string,
	message *genai.Content,
	runConfig agent.RunConfig,
) {
	// 追踪当前的 agent author，用于 FunctionResponse
	// FunctionResponse 的 event.Author 是 "user"（GenAI 协议），但我们需要使用调用工具的 agent 名称
	var currentAgentAuthor string

	// 启动心跳，防止长时间无响应导致连接超时
	cancelHeartbeat := startHeartbeat(ctx, 30*time.Second)
	defer cancelHeartbeat()

	for event, err := range runner.Run(ctx, userID, sessionID, message, runConfig) {
		// 检查客户端是否断开连接
		select {
		case <-ctx.Request.Context().Done():
			slog.Debug("客户端断开连接（可能是用户主动取消或关闭页面）", "err", ctx.Request.Context().Err())
			return // 客户端断开，停止处理
		default:
		}

		if err != nil {
			// 发送错误事件，包含详细的错误信息
			slog.Error("runner.Run() error", "err", err, "userID", userID, "sessionID", sessionID)
			errorMsg := err.Error()
			if errors.Is(err, gorm.ErrRecordNotFound) {
				errorMsg = "Session 不存在或已被删除，请重新创建"
			}
			ctx.SSEvent("error", gin.H{"error": errorMsg})
			ctx.Writer.Flush()
			return
		}

		if event == nil {
			continue
		}

		// 更新当前 agent author（非 "user" 的才是真正的 agent）
		if event.Author != "" && event.Author != "user" {
			currentAgentAuthor = event.Author
		}

		// 提取事件中的文本内容和图片数据
		if event.LLMResponse.Content != nil && len(event.LLMResponse.Content.Parts) > 0 {
			for _, part := range event.LLMResponse.Content.Parts {
				// 处理文本内容
				if part.Text != "" && event.Partial {
					// 这里只返回 partial 的文本内容。
					// LLM 通常会先返回 partial 的文本内容，然后再返回这些 partial 组合而成的完整内容。
					// 所以我们只需要返回一份就可以了，因为 partial 更快生成，所以我们返回 partial 内容。
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
							// FunctionResponse 的 event.Author 是 "user"（GenAI 协议）
							// 使用追踪的 currentAgentAuthor 作为图片数据的作者
							author := currentAgentAuthor
							if author == "" {
								author = "model" // 降级方案
							}
							data := gin.H{
								"author": author,
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

// startHeartbeat 启动心跳 goroutine，防止 SSE 连接因长时间无响应而超时
// 返回一个 cancel 函数，调用它可以停止心跳
func startHeartbeat(ctx *gin.Context, interval time.Duration) context.CancelFunc {
	heartbeatCtx, cancel := context.WithCancel(ctx.Request.Context())

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-heartbeatCtx.Done():
				return
			case <-ticker.C:
				// 发送心跳事件（客户端应忽略此事件）
				ctx.SSEvent("heartbeat", gin.H{"timestamp": time.Now().Unix()})
				ctx.Writer.Flush()
			}
		}
	}()

	return cancel
}

const titlePromptTemplate = `Generate a concise title for this conversation based on the user's message: "%s". 
Rules: 
1. Use the same language as the user's message. 
2. Do NOT use any Markdown formatting (no bold, no italics). 
3. Do NOT use quotes in the title. 
4. Output only the title text.`

// generateTitle 生成会话标题
func (a *Assistant) generateTitle(ctx context.Context, sessionID, firstMessage string) error {
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
		slog.Error("no content generated for title")
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

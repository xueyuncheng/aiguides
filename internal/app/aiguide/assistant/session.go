package assistant

import (
	"aiguide/internal/app/aiguide/table"
	"aiguide/internal/pkg/middleware"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/gin-gonic/gin"
	"google.golang.org/adk/session"
)

const defaultImageMimeType = "image/png"
const (
	defaultHistoryLimit = 50
	maxHistoryLimit     = 100
)

// SessionInfo 定义会话信息的响应结构
type SessionInfo struct {
	SessionID      string    `json:"session_id"`
	AppName        string    `json:"app_name"`
	UserID         int       `json:"user_id"`
	ThreadID       string    `json:"thread_id,omitempty"`
	ProjectID      int       `json:"project_id"`
	ProjectName    string    `json:"project_name,omitempty"`
	Version        int       `json:"version,omitempty"`
	LastUpdateTime time.Time `json:"last_update_time"`
	MessageCount   int       `json:"message_count"`
	FirstMessage   string    `json:"first_message"`
	Title          string    `json:"title"`
}

// SessionHistoryResponse 定义会话历史的响应结构
type SessionHistoryResponse struct {
	SessionID string         `json:"session_id"`
	AppName   string         `json:"app_name"`
	UserID    int            `json:"user_id"`
	Messages  []MessageEvent `json:"messages"`
	Total     int            `json:"total,omitempty"`
	Limit     int            `json:"limit,omitempty"`
	Offset    int            `json:"offset,omitempty"`
	HasMore   bool           `json:"has_more,omitempty"`
}

// MessageEvent 定义消息事件结构
type MessageEvent struct {
	ID        string        `json:"id"`
	Timestamp time.Time     `json:"timestamp"`
	Role      string        `json:"role"` // "user" or "assistant"
	Content   string        `json:"content"`
	Thought   string        `json:"thought,omitempty"`
	Images    []string      `json:"images,omitempty"`     // Base64编码的图片列表
	Videos    []string      `json:"videos,omitempty"`     // 视频文件URL列表
	FileNames []string      `json:"file_names,omitempty"` // 文件名列表，与 Images 对应
	Files     []MessageFile `json:"files,omitempty"`
	ToolCalls []ToolCall    `json:"tool_calls,omitempty"`
}

type MessageFile struct {
	MimeType string `json:"mime_type"`
	Name     string `json:"name,omitempty"`
	Label    string `json:"label,omitempty"`
}

type ToolCall struct {
	CallID   string         `json:"tool_call_id,omitempty"`
	ToolName string         `json:"tool_name"`
	Label    string         `json:"label"`
	Args     map[string]any `json:"args,omitempty"`
	Result   map[string]any `json:"result,omitempty"`
}

type toolCallLocation struct {
	messageIndex  int
	toolCallIndex int
}

// CreateSessionRequest 定义创建会话的请求结构
type CreateSessionRequest struct {
	UserID    int `json:"user_id" binding:"required"`
	ProjectID int `json:"project_id"`
}

// CreateSessionResponse 定义创建会话的响应结构
type CreateSessionResponse struct {
	SessionID string    `json:"session_id"`
	CreatedAt time.Time `json:"created_at"`
}

// ListSessions 处理获取会话列表的请求
// GET /api/:agentId/sessions?user_id=xxx
func (a *Assistant) ListSessions(ctx *gin.Context) {
	agentID := ctx.Param("agentId")
	userIDStr := ctx.Query("user_id")

	if userIDStr == "" {
		slog.Error("user_id is required")
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
		return
	}

	userIDInt, err := strconv.Atoi(userIDStr)
	if err != nil {
		slog.Error("invalid user_id", "user_id", userIDStr, "err", err)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	listReq := &session.ListRequest{
		AppName: agentID,
		UserID:  userIDStr,
	}

	listResp, err := a.session.List(ctx, listReq)
	if err != nil {
		slog.Error("session.List() error", "err", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 批量获取所有会话的元数据，避免 N+1 查询问题
	sessionIDs := make([]string, 0, len(listResp.Sessions))
	for _, sess := range listResp.Sessions {
		sessionIDs = append(sessionIDs, sess.ID())
	}

	type sessionMetaInfo struct {
		Title       string
		ThreadID    string
		ProjectID   int
		ProjectName string
		Version     int
	}
	metadataMap := make(map[string]sessionMetaInfo) // sessionID -> metadata
	if len(sessionIDs) > 0 {
		var metadataList []table.SessionMeta
		if err := a.db.Where("session_id IN ?", sessionIDs).Find(&metadataList).Error; err == nil {
			projectNameMap := make(map[int]string)
			projectIDs := make([]int, 0)
			projectIDSet := make(map[int]struct{})
			for _, meta := range metadataList {
				if meta.ProjectID == 0 {
					continue
				}
				if _, ok := projectIDSet[meta.ProjectID]; ok {
					continue
				}
				projectIDSet[meta.ProjectID] = struct{}{}
				projectIDs = append(projectIDs, meta.ProjectID)
			}
			if len(projectIDs) > 0 {
				var projects []table.Project
				if err := a.db.Where("id IN ? AND user_id = ?", projectIDs, userIDInt).Find(&projects).Error; err == nil {
					for _, project := range projects {
						projectNameMap[project.ID] = project.Name
					}
				}
			}
			for _, meta := range metadataList {
				projectName := ""
				if meta.ProjectID != 0 {
					projectName = projectNameMap[meta.ProjectID]
				}
				candidate := sessionMetaInfo{
					Title:       meta.Title,
					ThreadID:    meta.ThreadID,
					ProjectID:   meta.ProjectID,
					ProjectName: projectName,
					Version:     meta.Version,
				}
				existing, ok := metadataMap[meta.SessionID]
				if !ok || shouldReplaceSessionMeta(existing, candidate) {
					metadataMap[meta.SessionID] = candidate
				}
			}
		}
	}

	// 将会话转换为响应格式
	sessions := make([]SessionInfo, 0, len(listResp.Sessions))
	for _, sess := range listResp.Sessions {
		events := sess.Events()
		messageCount := 0
		firstMessage := ""

		// 遍历事件以统计消息数量和获取第一条消息
		for event := range events.All() {
			if event.Content != nil {
				messageCount++
				if firstMessage == "" && len(event.Content.Parts) > 0 {
					// 获取第一条消息的内容
					for _, part := range event.Content.Parts {
						if part.Text != "" {
							firstMessage = part.Text
							if len(firstMessage) > 50 {
								firstMessage = firstMessage[:50] + "..."
							}
							break
						}
					}
				}
			}
		}

		meta := metadataMap[sess.ID()]
		threadID := meta.ThreadID
		if threadID == "" {
			threadID = sess.ID()
		}
		version := meta.Version
		if version <= 0 {
			version = 1
		}

		sessions = append(sessions, SessionInfo{
			SessionID:      sess.ID(),
			AppName:        sess.AppName(),
			UserID:         userIDInt,
			ThreadID:       threadID,
			ProjectID:      meta.ProjectID,
			ProjectName:    meta.ProjectName,
			Version:        version,
			LastUpdateTime: sess.LastUpdateTime(),
			MessageCount:   messageCount,
			FirstMessage:   firstMessage,
			Title:          meta.Title,
		})
	}

	ctx.JSON(http.StatusOK, sessions)
}

func shouldReplaceSessionMeta(existing, candidate struct {
	Title       string
	ThreadID    string
	ProjectID   int
	ProjectName string
	Version     int
}) bool {
	if existing.ThreadID == "" && candidate.ThreadID != "" {
		return true
	}
	if existing.ProjectID == 0 && candidate.ProjectID != 0 {
		return true
	}
	if candidate.Version > existing.Version {
		return true
	}
	if existing.Title == "" && candidate.Title != "" {
		return true
	}
	if existing.ProjectName == "" && candidate.ProjectName != "" {
		return true
	}
	return false
}

// GetSessionHistory 处理获取会话历史的请求
// GET /api/:agentId/sessions/:sessionId/history?user_id=xxx&limit=50&offset=0
func (a *Assistant) GetSessionHistory(ctx *gin.Context) {
	agentID := ctx.Param("agentId")
	sessionID := ctx.Param("sessionId")
	userIDStr, userIDInt, ok := parseUserID(ctx)
	if !ok {
		return
	}

	limit, offset := parsePagination(ctx)

	getReq := &session.GetRequest{
		AppName:   agentID,
		UserID:    userIDStr,
		SessionID: sessionID,
	}

	getResp, err := a.session.Get(ctx, getReq)
	if err != nil {
		slog.Error("session.Get() error", "err", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	sess := getResp.Session

	allMessages := buildMessageEvents(sess.Events(), middleware.GetLocale(ctx))
	totalCount := len(allMessages)
	if offset >= totalCount {
		response := SessionHistoryResponse{
			SessionID: sess.ID(),
			AppName:   sess.AppName(),
			UserID:    userIDInt,
			Messages:  []MessageEvent{},
			Total:     totalCount,
			Limit:     limit,
			Offset:    offset,
			HasMore:   false,
		}
		ctx.JSON(http.StatusOK, response)
		return
	}

	messages, hasMore := paginateMessages(allMessages, limit, offset)

	response := SessionHistoryResponse{
		SessionID: sess.ID(),
		AppName:   sess.AppName(),
		UserID:    userIDInt,
		Messages:  messages,
		Total:     totalCount,
		Limit:     limit,
		Offset:    offset,
		HasMore:   hasMore,
	}

	ctx.JSON(http.StatusOK, response)
}

func parseUserID(ctx *gin.Context) (string, int, bool) {
	userIDStr := ctx.Query("user_id")
	if userIDStr == "" {
		slog.Error("user_id is required")
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
		return "", 0, false
	}

	userIDInt, err := strconv.Atoi(userIDStr)
	if err != nil {
		slog.Error("invalid user_id", "user_id", userIDStr, "err", err)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return "", 0, false
	}

	return userIDStr, userIDInt, true
}

func parsePagination(ctx *gin.Context) (int, int) {
	limit := defaultHistoryLimit
	if limitStr := ctx.Query("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = min(parsedLimit, maxHistoryLimit)
		}
	}

	offset := 0
	if offsetStr := ctx.Query("offset"); offsetStr != "" {
		if parsedOffset, err := strconv.Atoi(offsetStr); err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}
	}

	return limit, offset
}

func buildMessageEvents(events session.Events, locale string) []MessageEvent {
	allMessages := make([]MessageEvent, 0)
	toolCallLocations := make(map[string]toolCallLocation)
	for event := range events.All() {
		if event.Content == nil {
			continue
		}

		role := "assistant"
		if event.Content.Role == "user" {
			role = "user"
		}

		content := ""
		thought := ""
		var images []string
		var videos []string
		var fileNames []string
		var files []MessageFile
		var toolCalls []ToolCall
		localToolCallIDs := make([]string, 0)
		localFunctionResponses := make(map[string]map[string]any)
		hasFunctionResponse := false

		for _, part := range event.Content.Parts {
			if part.Thought {
				thought += part.Text
			} else if part.Text != "" {
				if fileName, ok := extractPDFFileNameFromText(part.Text); ok {
					label := fileName
					if label == "" {
						label = fmt.Sprintf("PDF 文件 %d", len(files)+1)
					}
					files = append(files, MessageFile{
						MimeType: pdfMimeType,
						Name:     fileName,
						Label:    label,
					})
					continue
				}
				text := part.Text
				// 移除自动注入的用户记忆上下文，不暴露给前端
				text = stripUserContext(text)
				// 解析文件名元数据
				if parsedText, parsedFileNames, ok := extractFileNamesMetadata(text); ok {
					text = parsedText
					fileNames = parsedFileNames
				}
				content += text
			}

			if part.InlineData != nil && len(part.InlineData.Data) > 0 {
				mimeType := strings.TrimSpace(part.InlineData.MIMEType)
				if mimeType == "" {
					mimeType = defaultImageMimeType
				}
				if strings.HasPrefix(mimeType, "image/") {
					base64Image := base64.StdEncoding.EncodeToString(part.InlineData.Data)
					imageDataURI := fmt.Sprintf("data:%s;base64,%s", mimeType, base64Image)
					images = append(images, imageDataURI)
				} else if mimeType == pdfMimeType {
					fileName := ""
					if len(fileNames) > len(files) {
						fileName = fileNames[len(files)]
					}
					label := fileName
					if strings.TrimSpace(label) == "" {
						label = fmt.Sprintf("PDF 文件 %d", len(files)+1)
					}
					files = append(files, MessageFile{
						MimeType: mimeType,
						Name:     fileName,
						Label:    label,
					})
				}
			}

			if part.FunctionResponse != nil {
				hasFunctionResponse = true
				response := part.FunctionResponse.Response
				if location, ok := toolCallLocations[part.FunctionResponse.ID]; ok {
					allMessages[location.messageIndex].ToolCalls[location.toolCallIndex].Result = response
				} else {
					localFunctionResponses[part.FunctionResponse.ID] = response
				}
				if imageList, ok := response["images"].([]any); ok {
					for _, img := range imageList {
						if imgStr, ok := img.(string); ok {
							images = append(images, imgStr)
						}
					}
				}
				if videoList, ok := response["videos"].([]any); ok {
					for _, vid := range videoList {
						if vidStr, ok := vid.(string); ok {
							videos = append(videos, vidStr)
						}
					}
				}
			}

			if part.FunctionCall != nil {
				toolCall := ToolCall{
					CallID:   part.FunctionCall.ID,
					ToolName: part.FunctionCall.Name,
					Label:    toolCallLabel(locale, part.FunctionCall.Name, part.FunctionCall.Args),
					Args:     part.FunctionCall.Args,
				}
				if response, ok := localFunctionResponses[part.FunctionCall.ID]; ok {
					toolCall.Result = response
				}
				toolCalls = append(toolCalls, ToolCall{
					CallID:   toolCall.CallID,
					ToolName: toolCall.ToolName,
					Label:    toolCall.Label,
					Args:     toolCall.Args,
					Result:   toolCall.Result,
				})
				localToolCallIDs = append(localToolCallIDs, part.FunctionCall.ID)
			}
		}

		if hasFunctionResponse {
			role = "assistant"
		}

		if content != "" || thought != "" || len(images) > 0 || len(videos) > 0 || len(files) > 0 || len(toolCalls) > 0 {
			message := MessageEvent{
				ID:        event.ID,
				Timestamp: event.Timestamp,
				Role:      role,
				Content:   content,
				Thought:   thought,
				Images:    images,
				Videos:    videos,
				FileNames: fileNames,
				Files:     files,
				ToolCalls: toolCalls,
			}
			if isDuplicateRetryUserMessage(allMessages, message) {
				continue
			}

			messageIndex := len(allMessages)
			if shouldMergeAssistantMessage(allMessages, message) {
				messageIndex = len(allMessages) - 1
				allMessages[messageIndex].Content += message.Content
				allMessages[messageIndex].Thought += message.Thought
				allMessages[messageIndex].Images = append(allMessages[messageIndex].Images, message.Images...)
			allMessages[messageIndex].Videos = append(allMessages[messageIndex].Videos, message.Videos...)
				allMessages[messageIndex].FileNames = append(allMessages[messageIndex].FileNames, message.FileNames...)
				allMessages[messageIndex].Files = append(allMessages[messageIndex].Files, message.Files...)
				toolCallOffset := len(allMessages[messageIndex].ToolCalls)
				allMessages[messageIndex].ToolCalls = append(allMessages[messageIndex].ToolCalls, message.ToolCalls...)
				for idx, callID := range localToolCallIDs {
					if strings.TrimSpace(callID) == "" {
						continue
					}
					toolCallLocations[callID] = toolCallLocation{messageIndex: messageIndex, toolCallIndex: toolCallOffset + idx}
				}
				continue
			}

			allMessages = append(allMessages, message)
			for idx, callID := range localToolCallIDs {
				if strings.TrimSpace(callID) == "" {
					continue
				}
				toolCallLocations[callID] = toolCallLocation{messageIndex: messageIndex, toolCallIndex: idx}
			}
		}
	}

	return allMessages
}

func shouldMergeAssistantMessage(allMessages []MessageEvent, current MessageEvent) bool {
	if current.Role != "assistant" || len(allMessages) == 0 {
		return false
	}

	previous := allMessages[len(allMessages)-1]
	if previous.Role != "assistant" {
		return false
	}

	if previous.Content != "" || previous.Thought != "" || len(previous.Images) > 0 || len(previous.Videos) > 0 || len(previous.FileNames) > 0 || len(previous.Files) > 0 || len(previous.ToolCalls) == 0 {
		return false
	}

	return current.Content != "" || current.Thought != "" || len(current.Images) > 0 || len(current.Videos) > 0 || len(current.FileNames) > 0 || len(current.Files) > 0
}

func isDuplicateRetryUserMessage(allMessages []MessageEvent, current MessageEvent) bool {
	if current.Role != "user" || len(allMessages) == 0 {
		return false
	}

	last := allMessages[len(allMessages)-1]
	if last.Role != "user" {
		return false
	}

	return last.Content == current.Content &&
		last.Thought == current.Thought &&
		slices.Equal(last.Images, current.Images) &&
		slices.Equal(last.FileNames, current.FileNames) &&
		slices.Equal(last.Files, current.Files)
}

// stripUserContext 移除消息文本中自动注入的 <user_context> 块
func stripUserContext(text string) string {
	const openTag = "<user_context>\n"
	const closeTag = "</user_context>\n"
	for {
		start := strings.Index(text, openTag)
		if start == -1 {
			break
		}
		end := strings.Index(text[start:], closeTag)
		if end == -1 {
			// 没有闭合标签，截断到 openTag 之前
			text = text[:start]
			break
		}
		text = text[:start] + text[start+end+len(closeTag):]
	}
	return text
}

func extractFileNamesMetadata(text string) (string, []string, bool) {
	trimmed := strings.TrimLeftFunc(text, unicode.IsSpace)
	if !strings.HasPrefix(trimmed, "<!-- FILE_NAMES:") {
		return text, nil, false
	}

	endIdx := strings.Index(trimmed, "-->")
	if endIdx <= 0 {
		return text, nil, false
	}

	metaStr := strings.TrimSpace(trimmed[len("<!-- FILE_NAMES:"):endIdx])
	var fileNames []string
	if err := json.Unmarshal([]byte(metaStr), &fileNames); err != nil {
		return text, nil, false
	}

	return strings.TrimPrefix(trimmed[endIdx+3:], "\n"), fileNames, true
}

func paginateMessages(allMessages []MessageEvent, limit, offset int) ([]MessageEvent, bool) {
	startIdx := max(len(allMessages)-offset-limit, 0)
	endIdx := len(allMessages) - offset
	if startIdx >= endIdx {
		return []MessageEvent{}, false
	}

	messages := allMessages[startIdx:endIdx]
	hasMore := startIdx > 0
	return messages, hasMore
}

// CreateSession 处理创建新会话的请求
// POST /api/:agentId/sessions
func (a *Assistant) CreateSession(ctx *gin.Context) {
	agentID := ctx.Param("agentId")

	var req CreateSessionRequest
	if err := ctx.BindJSON(&req); err != nil {
		slog.Error("ctx.BindJSON() error", "err", err)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	if err := a.ensureProjectOwnership(req.UserID, req.ProjectID); err != nil {
		if errors.Is(err, errProjectNotFound) {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid project_id"})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to validate project"})
		return
	}

	// 生成新的会话 ID
	sessionID := generateSessionID()

	createReq := &session.CreateRequest{
		AppName:   agentID,
		UserID:    strconv.Itoa(req.UserID),
		SessionID: sessionID,
		State:     map[string]any{},
	}

	if _, err := a.session.Create(ctx, createReq); err != nil {
		slog.Error("session.Create() error", "err", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if req.ProjectID != 0 {
		if err := a.upsertSessionProjectMeta(sessionID, req.ProjectID); err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save session project"})
			return
		}
	}

	response := CreateSessionResponse{
		SessionID: sessionID,
		CreatedAt: time.Now(),
	}

	ctx.JSON(http.StatusOK, response)
}

// DeleteSession 处理删除会话的请求
// DELETE /api/:agentId/sessions/:sessionId?user_id=xxx
func (a *Assistant) DeleteSession(ctx *gin.Context) {
	agentID := ctx.Param("agentId")
	sessionID := ctx.Param("sessionId")
	userIDStr := ctx.Query("user_id")

	if userIDStr == "" {
		slog.Error("user_id is required")
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
		return
	}

	if _, err := strconv.Atoi(userIDStr); err != nil {
		slog.Error("invalid user_id", "user_id", userIDStr, "err", err)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	deleteReq := &session.DeleteRequest{
		AppName:   agentID,
		UserID:    userIDStr,
		SessionID: sessionID,
	}

	if err := a.session.Delete(ctx, deleteReq); err != nil {
		slog.Error("session.Delete() error", "err", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "session deleted successfully"})
}

// generateSessionID 生成唯一的会话 ID
func generateSessionID() string {
	return "session-" + time.Now().Format("20060102-150405") + "-" + randomString(8)
}

// randomString 生成指定长度的随机字符串（使用加密安全的随机数生成器）
func randomString(length int) string {
	bytes := make([]byte, length/2+1) // hex encoding doubles the length
	if _, err := rand.Read(bytes); err != nil {
		// 降级到基于时间的方法（不应该发生）
		slog.Error("crypto/rand.Read() failed", "err", err)
		return time.Now().Format("150405999999")[:length]
	}
	return hex.EncodeToString(bytes)[:length]
}

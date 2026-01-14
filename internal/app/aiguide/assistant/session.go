package assistant

import (
	"aiguide/internal/app/aiguide/table"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"google.golang.org/adk/session"
)

// SessionInfo 定义会话信息的响应结构
type SessionInfo struct {
	SessionID      string    `json:"session_id"`
	AppName        string    `json:"app_name"`
	UserID         int       `json:"user_id"`
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
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Role      string    `json:"role"` // "user" or "assistant"
	Content   string    `json:"content"`
	Thought   string    `json:"thought,omitempty"`
	Images    []string  `json:"images,omitempty"` // Base64编码的图片数据列表
}

// CreateSessionRequest 定义创建会话的请求结构
type CreateSessionRequest struct {
	UserID int `json:"user_id" binding:"required"`
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
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
		return
	}

	userIDInt, err := strconv.Atoi(userIDStr)
	if err != nil {
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

	metadataMap := make(map[string]string) // sessionID -> title
	if len(sessionIDs) > 0 {
		var metadataList []table.SessionMeta
		if err := a.db.Where("session_id IN ?", sessionIDs).Find(&metadataList).Error; err == nil {
			for _, meta := range metadataList {
				metadataMap[meta.SessionID] = meta.Title
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

		sessions = append(sessions, SessionInfo{
			SessionID:      sess.ID(),
			AppName:        sess.AppName(),
			UserID:         userIDInt,
			LastUpdateTime: sess.LastUpdateTime(),
			MessageCount:   messageCount,
			FirstMessage:   firstMessage,
			Title:          metadataMap[sess.ID()],
		})
	}

	ctx.JSON(http.StatusOK, sessions)
}

// GetSessionHistory 处理获取会话历史的请求
// GET /api/:agentId/sessions/:sessionId/history?user_id=xxx&limit=50&offset=0
func (a *Assistant) GetSessionHistory(ctx *gin.Context) {
	agentID := ctx.Param("agentId")
	sessionID := ctx.Param("sessionId")
	userIDStr := ctx.Query("user_id")

	if userIDStr == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
		return
	}

	userIDInt, err := strconv.Atoi(userIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	// 解析分页参数
	limit := 50 // 默认返回最近50条消息
	if limitStr := ctx.Query("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			// 限制最大值防止性能问题
			limit = min(parsedLimit, 100)
		}
	}

	offset := 0
	if offsetStr := ctx.Query("offset"); offsetStr != "" {
		if parsedOffset, err := strconv.Atoi(offsetStr); err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}
	}

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
	events := sess.Events()

	// 将所有事件转换为消息格式
	allMessages := make([]MessageEvent, 0)
	for event := range events.All() {
		if event.Content != nil {
			role := "assistant"
			if event.Content.Role == "user" {
				role = "user"
			}

			content := ""
			thought := ""
			var images []string
			// 如果本条消息包含 FunctionResponse，则将其视为 assistant 输出进行展示
			hasFunctionResponse := false
			for _, part := range event.Content.Parts {
				if part.Thought {
					thought += part.Text
				} else if part.Text != "" {
					content += part.Text
				}

				// 处理 FunctionResponse 中的图片数据
				if part.FunctionResponse != nil {
					hasFunctionResponse = true
					response := part.FunctionResponse.Response
					if imageList, ok := response["images"].([]any); ok {
						for _, img := range imageList {
							if imgStr, ok := img.(string); ok {
								images = append(images, imgStr)
							}
						}
					}
				}
			}

			// GenAI 协议中工具响应以 user 角色回传，但前端展示应归为 assistant
			if hasFunctionResponse {
				role = "assistant"
			}

			if content != "" || thought != "" || len(images) > 0 {
				allMessages = append(allMessages, MessageEvent{
					ID:        event.ID,
					Timestamp: event.Timestamp,
					Role:      role,
					Content:   content,
					Thought:   thought,
					Images:    images,
				})
			}
		}
	}

	totalCount := len(allMessages)

	// 验证 offset 是否超出范围
	if offset >= totalCount {
		// offset 超出范围，返回空消息列表
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

	// 应用分页：从最新消息开始，offset=0表示最新的消息
	// 为了返回最近的消息，我们从末尾开始取
	messages := make([]MessageEvent, 0)
	startIdx := max(totalCount-offset-limit, 0)
	endIdx := totalCount - offset

	if startIdx < endIdx {
		messages = allMessages[startIdx:endIdx]
	}

	hasMore := startIdx > 0

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
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
		return
	}

	if _, err := strconv.Atoi(userIDStr); err != nil {
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

package agentmanager

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"google.golang.org/adk/session"
)

// SessionInfo 定义会话信息的响应结构
type SessionInfo struct {
	SessionID      string    `json:"session_id"`
	AppName        string    `json:"app_name"`
	UserID         string    `json:"user_id"`
	LastUpdateTime time.Time `json:"last_update_time"`
	MessageCount   int       `json:"message_count"`
	FirstMessage   string    `json:"first_message,omitempty"`
}

// SessionHistoryResponse 定义会话历史的响应结构
type SessionHistoryResponse struct {
	SessionID string         `json:"session_id"`
	AppName   string         `json:"app_name"`
	UserID    string         `json:"user_id"`
	Messages  []MessageEvent `json:"messages"`
}

// MessageEvent 定义消息事件结构
type MessageEvent struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Role      string    `json:"role"` // "user" or "assistant"
	Content   string    `json:"content"`
}

// CreateSessionRequest 定义创建会话的请求结构
type CreateSessionRequest struct {
	UserID string `json:"user_id" binding:"required"`
}

// CreateSessionResponse 定义创建会话的响应结构
type CreateSessionResponse struct {
	SessionID string    `json:"session_id"`
	CreatedAt time.Time `json:"created_at"`
}

// listSessionsHandler 处理获取会话列表的请求
// GET /api/:agentId/sessions?user_id=xxx
func (a *AgentManager) listSessionsHandler(ctx *gin.Context) {
	agentID := ctx.Param("agentId")
	userID := ctx.Query("user_id")

	if userID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
		return
	}

	listReq := &session.ListRequest{
		AppName: agentID,
		UserID:  userID,
	}

	listResp, err := a.session.List(ctx, listReq)
	if err != nil {
		slog.Error("session.List() error", "err", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
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
			UserID:         sess.UserID(),
			LastUpdateTime: sess.LastUpdateTime(),
			MessageCount:   messageCount,
			FirstMessage:   firstMessage,
		})
	}

	ctx.JSON(http.StatusOK, sessions)
}

// getSessionHistoryHandler 处理获取会话历史的请求
// GET /api/:agentId/sessions/:sessionId/history?user_id=xxx
func (a *AgentManager) getSessionHistoryHandler(ctx *gin.Context) {
	agentID := ctx.Param("agentId")
	sessionID := ctx.Param("sessionId")
	userID := ctx.Query("user_id")

	if userID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
		return
	}

	getReq := &session.GetRequest{
		AppName:   agentID,
		UserID:    userID,
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

	// 将事件转换为消息格式
	messages := make([]MessageEvent, 0)
	for event := range events.All() {
		if event.Content != nil {
			role := "assistant"
			if event.Content.Role == "user" {
				role = "user"
			}

			content := ""
			for _, part := range event.Content.Parts {
				if part.Text != "" {
					content += part.Text
				}
			}

			if content != "" {
				messages = append(messages, MessageEvent{
					ID:        event.ID,
					Timestamp: event.Timestamp,
					Role:      role,
					Content:   content,
				})
			}
		}
	}

	response := SessionHistoryResponse{
		SessionID: sess.ID(),
		AppName:   sess.AppName(),
		UserID:    sess.UserID(),
		Messages:  messages,
	}

	ctx.JSON(http.StatusOK, response)
}

// createSessionHandler 处理创建新会话的请求
// POST /api/:agentId/sessions
func (a *AgentManager) createSessionHandler(ctx *gin.Context) {
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
		UserID:    req.UserID,
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

// deleteSessionHandler 处理删除会话的请求
// DELETE /api/:agentId/sessions/:sessionId?user_id=xxx
func (a *AgentManager) deleteSessionHandler(ctx *gin.Context) {
	agentID := ctx.Param("agentId")
	sessionID := ctx.Param("sessionId")
	userID := ctx.Query("user_id")

	if userID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
		return
	}

	deleteReq := &session.DeleteRequest{
		AppName:   agentID,
		UserID:    userID,
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

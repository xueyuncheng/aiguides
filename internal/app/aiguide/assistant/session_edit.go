package assistant

import (
	"aiguide/internal/app/aiguide/table"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"google.golang.org/adk/session"
	"google.golang.org/genai"
	"gorm.io/gorm"
)

// EditSessionRequest 定义编辑会话消息的请求结构
type EditSessionRequest struct {
	UserID     int      `json:"user_id" binding:"required"`
	MessageID  string   `json:"message_id" binding:"required"`
	NewContent string   `json:"new_content"`
	Images     []string `json:"images,omitempty"`
	FileNames  []string `json:"file_names,omitempty"`
}

// EditSessionResponse 定义编辑会话消息的响应结构
type EditSessionResponse struct {
	ThreadID            string `json:"thread_id"`
	NewSessionID        string `json:"new_session_id"`
	Version             int    `json:"version"`
	EditedFromMessageID string `json:"edited_from_message_id"`
}

// EditSession 编辑历史用户消息并创建新会话版本
// POST /api/:agentId/sessions/:sessionId/edit
func (a *Assistant) EditSession(ctx *gin.Context) {
	agentID := ctx.Param("agentId")
	sessionID := ctx.Param("sessionId")

	var req EditSessionRequest
	if err := ctx.BindJSON(&req); err != nil {
		slog.Error("ctx.BindJSON() error", "err", err)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid request", "code": "invalid_edit_payload"})
		return
	}

	trimmedContent := strings.Trim(req.NewContent, "\n\r")
	if len(req.Images) > maxUserFileCount {
		slog.Error("too many images", "count", len(req.Images), "max", maxUserFileCount)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("too many images (max %d)", maxUserFileCount), "code": "invalid_edit_payload"})
		return
	}
	if len(req.FileNames) > 0 && len(req.FileNames) != len(req.Images) {
		slog.Error("file_names count mismatch", "file_names", len(req.FileNames), "images", len(req.Images))
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "file_names length must match images", "code": "invalid_edit_payload"})
		return
	}

	_, err := buildUserMessageParts(trimmedContent, req.Images, req.FileNames)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "code": "invalid_edit_payload"})
		return
	}

	getReq := &session.GetRequest{
		AppName:   agentID,
		UserID:    strconv.Itoa(req.UserID),
		SessionID: sessionID,
	}

	getResp, err := a.session.Get(ctx, getReq)
	if err != nil {
		slog.Error("session.Get() error", "err", err, "session_id", sessionID, "user_id", req.UserID)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load session"})
		return
	}

	eventsToCopy, found, editable := collectEventsBeforeTarget(getResp.Session.Events(), req.MessageID)
	if !found {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "message not found", "code": "message_not_found"})
		return
	}
	if !editable {
		ctx.JSON(http.StatusConflict, gin.H{"error": "message is not editable", "code": "message_not_editable"})
		return
	}

	newSessionID := generateSessionID()
	state := cloneSessionState(getResp.Session.State())
	createReq := &session.CreateRequest{
		AppName:   agentID,
		UserID:    strconv.Itoa(req.UserID),
		SessionID: newSessionID,
		State:     state,
	}
	createResp, err := a.session.Create(ctx, createReq)
	if err != nil {
		slog.Error("session.Create() error", "err", err, "session_id", newSessionID, "user_id", req.UserID)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create edited session"})
		return
	}

	for _, event := range eventsToCopy {
		if err := a.session.AppendEvent(ctx, createResp.Session, event); err != nil {
			slog.Error("session.AppendEvent() error", "err", err, "session_id", newSessionID)
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to replay session history"})
			return
		}
	}

	threadID, version, err := a.createEditedSessionMeta(sessionID, newSessionID, req.MessageID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to persist edit metadata"})
		return
	}

	ctx.JSON(http.StatusOK, EditSessionResponse{
		ThreadID:            threadID,
		NewSessionID:        newSessionID,
		Version:             version,
		EditedFromMessageID: req.MessageID,
	})
}

func buildUserMessageParts(message string, images, fileNames []string) ([]*genai.Part, error) {
	parts := make([]*genai.Part, 0, 1+len(images))

	actualMessage := message
	if len(fileNames) > 0 && len(fileNames) == len(images) {
		fileNamesJSON, _ := json.Marshal(fileNames)
		actualMessage = fmt.Sprintf("<!-- FILE_NAMES: %s -->\n%s", fileNamesJSON, message)
	}
	if actualMessage != "" {
		parts = append(parts, genai.NewPartFromText(actualMessage))
	}

	for _, image := range images {
		imageBytes, mimeType, err := parseDataURI(image)
		if err != nil {
			slog.Error("parseDataURI error", "err", err)
			return nil, err
		}
		parts = append(parts, genai.NewPartFromBytes(imageBytes, mimeType))
	}

	if len(parts) == 0 {
		slog.Error("message or images required")
		return nil, errors.New("message or images required")
	}

	return parts, nil
}

func collectEventsBeforeTarget(events session.Events, messageID string) ([]*session.Event, bool, bool) {
	eventsToCopy := make([]*session.Event, 0)
	found := false
	editable := false

	for event := range events.All() {
		if event.ID == messageID {
			found = true
			editable = event.Content != nil && event.Content.Role == genai.RoleUser
			break
		}
		eventsToCopy = append(eventsToCopy, cloneEvent(event))
	}

	return eventsToCopy, found, editable
}

func cloneEvent(event *session.Event) *session.Event {
	cloned := *event
	cloned.ID = uuid.NewString()
	if cloned.Timestamp.IsZero() {
		cloned.Timestamp = time.Now()
	}
	cloned.Content = cloneContent(event.Content)
	return &cloned
}

func cloneContent(content *genai.Content) *genai.Content {
	if content == nil {
		return nil
	}

	clonedContent := *content
	clonedParts := make([]*genai.Part, 0, len(content.Parts))
	for _, part := range content.Parts {
		if part == nil {
			continue
		}
		clonedPart := *part
		if part.InlineData != nil {
			clonedInlineData := *part.InlineData
			clonedInlineData.Data = append([]byte(nil), part.InlineData.Data...)
			clonedPart.InlineData = &clonedInlineData
		}
		clonedParts = append(clonedParts, &clonedPart)
	}
	clonedContent.Parts = clonedParts

	return &clonedContent
}

func cloneSessionState(state session.State) map[string]any {
	clonedState := map[string]any{}
	for key, value := range state.All() {
		clonedState[key] = value
	}
	return clonedState
}

func (a *Assistant) createEditedSessionMeta(parentSessionID, newSessionID, messageID string) (string, int, error) {
	var parentMeta table.SessionMeta
	if err := a.db.Where("session_id = ?", parentSessionID).First(&parentMeta).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			slog.Error("db.First() error", "err", err, "session_id", parentSessionID)
			return "", 0, fmt.Errorf("db.First() error: %w", err)
		}
	}

	threadID := parentMeta.ThreadID
	if threadID == "" {
		threadID = parentSessionID
	}
	parentVersion := parentMeta.Version
	if parentVersion == 0 {
		parentVersion = 1
	}

	if parentMeta.ID == 0 {
		parentMeta = table.SessionMeta{
			SessionID: parentSessionID,
			ThreadID:  threadID,
			Version:   parentVersion,
		}
		if err := a.db.Create(&parentMeta).Error; err != nil {
			slog.Error("db.Create() error", "err", err, "session_id", parentSessionID)
			return "", 0, fmt.Errorf("db.Create() error: %w", err)
		}
	} else if parentMeta.ThreadID == "" || parentMeta.Version == 0 {
		if err := a.db.Model(&parentMeta).Updates(map[string]any{
			"thread_id": threadID,
			"version":   parentVersion,
		}).Error; err != nil {
			slog.Error("db.Model().Updates() error", "err", err, "session_id", parentSessionID)
			return "", 0, fmt.Errorf("db.Model().Updates() error: %w", err)
		}
	}

	newVersion := parentVersion + 1
	newMeta := table.SessionMeta{
		SessionID:           newSessionID,
		Title:               parentMeta.Title,
		ThreadID:            threadID,
		Version:             newVersion,
		ParentSessionID:     parentSessionID,
		EditedFromMessageID: messageID,
	}
	if err := a.db.Create(&newMeta).Error; err != nil {
		slog.Error("db.Create() error", "err", err, "session_id", newSessionID)
		return "", 0, fmt.Errorf("db.Create() error: %w", err)
	}

	return threadID, newVersion, nil
}

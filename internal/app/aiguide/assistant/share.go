package assistant

import (
	"aiguide/internal/app/aiguide/table"
	"aiguide/internal/pkg/middleware"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"google.golang.org/adk/session"
)

const (
	defaultShareExpiryDays = 7
)

// CreateShareRequest defines the request structure for creating a share link
type CreateShareRequest struct {
	SessionID  string `json:"session_id" binding:"required"`
	AgentID    string `json:"agent_id" binding:"required"`
	ExpiryDays int    `json:"expiry_days,omitempty"`
}

// CreateShareResponse defines the response structure for creating a share link
type CreateShareResponse struct {
	ShareID   string    `json:"share_id"`
	ShareURL  string    `json:"share_url"`
	ExpiresAt time.Time `json:"expires_at"`
}

// SharedConversationResponse defines the response structure for accessing a shared conversation
type SharedConversationResponse struct {
	ShareID   string         `json:"share_id"`
	SessionID string         `json:"session_id"`
	AppName   string         `json:"app_name"`
	Messages  []MessageEvent `json:"messages"`
	ExpiresAt time.Time      `json:"expires_at"`
	IsExpired bool           `json:"is_expired"`
}

// CreateShare handles creating a shareable link for a conversation
// POST /api/assistant/share
func (a *Assistant) CreateShare(ctx *gin.Context) {
	userID, ok := middleware.GetUserID(ctx)
	if !ok {
		slog.Error("failed to get user ID from context")
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req CreateShareRequest
	if err := ctx.BindJSON(&req); err != nil {
		slog.Error("ctx.BindJSON() error", "err", err)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	// Validate expiry days
	expiryDays := req.ExpiryDays
	if expiryDays <= 0 || expiryDays > 30 {
		expiryDays = defaultShareExpiryDays
	}

	// Verify that the user owns the session
	userIDStr := fmt.Sprintf("%d", userID)
	getReq := &session.GetRequest{
		AppName:   req.AgentID,
		UserID:    userIDStr,
		SessionID: req.SessionID,
	}

	getResp, err := a.session.Get(ctx, getReq)
	if err != nil {
		slog.Error("session.Get() error", "err", err)
		ctx.JSON(http.StatusNotFound, gin.H{"error": "session not found or access denied"})
		return
	}

	sess := getResp.Session
	if sess.ID() != req.SessionID {
		slog.Error("session ID mismatch", "expected", req.SessionID, "actual", sess.ID())
		ctx.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		return
	}

	// Generate a unique share ID
	shareID := uuid.New().String()
	expiresAt := time.Now().Add(time.Duration(expiryDays) * 24 * time.Hour)

	// Create shared conversation record
	sharedConv := table.SharedConversation{
		ShareID:   shareID,
		SessionID: req.SessionID,
		UserID:    userID,
		AppName:   req.AgentID,
		ExpiresAt: expiresAt,
	}

	if err := a.db.Create(&sharedConv).Error; err != nil {
		slog.Error("a.db.Create() error", "err", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create share link"})
		return
	}

	// Build share URL (assuming the frontend will handle routing)
	shareURL := fmt.Sprintf("/share/%s", shareID)

	response := CreateShareResponse{
		ShareID:   shareID,
		ShareURL:  shareURL,
		ExpiresAt: expiresAt,
	}

	ctx.JSON(http.StatusOK, response)
}

// GetSharedConversation handles retrieving a shared conversation (public access, no authentication required)
// GET /api/share/:shareId
func (a *Assistant) GetSharedConversation(ctx *gin.Context) {
	shareID := ctx.Param("shareId")
	if shareID == "" {
		slog.Error("share_id is required")
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "share_id is required"})
		return
	}

	// Look up the shared conversation
	var sharedConv table.SharedConversation
	if err := a.db.Where("share_id = ?", shareID).First(&sharedConv).Error; err != nil {
		slog.Error("a.db.Where().First() error", "share_id", shareID, "err", err)
		ctx.JSON(http.StatusNotFound, gin.H{"error": "shared conversation not found"})
		return
	}

	// Update last accessed time
	a.db.Model(&sharedConv).Update("accessed_at", time.Now())

	// Check if the link has expired
	isExpired := time.Now().After(sharedConv.ExpiresAt)
	if isExpired {
		ctx.JSON(http.StatusGone, gin.H{
			"error":      "shared link has expired",
			"expires_at": sharedConv.ExpiresAt,
		})
		return
	}

	// Retrieve the session data
	userIDStr := fmt.Sprintf("%d", sharedConv.UserID)
	getReq := &session.GetRequest{
		AppName:   sharedConv.AppName,
		UserID:    userIDStr,
		SessionID: sharedConv.SessionID,
	}

	getResp, err := a.session.Get(ctx, getReq)
	if err != nil {
		slog.Error("session.Get() error", "err", err)
		ctx.JSON(http.StatusNotFound, gin.H{"error": "conversation not found"})
		return
	}

	sess := getResp.Session

	// Build message events from session
	allMessages := buildMessageEvents(sess.Events())

	response := SharedConversationResponse{
		ShareID:   shareID,
		SessionID: sharedConv.SessionID,
		AppName:   sharedConv.AppName,
		Messages:  allMessages,
		ExpiresAt: sharedConv.ExpiresAt,
		IsExpired: isExpired,
	}

	ctx.JSON(http.StatusOK, response)
}

// DeleteShare handles deleting a share link (only the owner can delete)
// DELETE /api/assistant/share/:shareId
func (a *Assistant) DeleteShare(ctx *gin.Context) {
	userID, ok := middleware.GetUserID(ctx)
	if !ok {
		slog.Error("failed to get user ID from context")
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	shareID := ctx.Param("shareId")
	if shareID == "" {
		slog.Error("share_id is required")
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "share_id is required"})
		return
	}

	// Find the shared conversation and verify ownership
	var sharedConv table.SharedConversation
	if err := a.db.Where("share_id = ? AND user_id = ?", shareID, userID).First(&sharedConv).Error; err != nil {
		slog.Error("a.db.Where().First() error", "share_id", shareID, "user_id", userID, "err", err)
		ctx.JSON(http.StatusNotFound, gin.H{"error": "shared conversation not found or access denied"})
		return
	}

	// Delete the share
	if err := a.db.Delete(&sharedConv).Error; err != nil {
		slog.Error("a.db.Delete() error", "err", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete share link"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "share link deleted successfully"})
}

// ListShares lists all active share links for the current user's sessions
// GET /api/assistant/share?session_id=xxx
func (a *Assistant) ListShares(ctx *gin.Context) {
	userID, ok := middleware.GetUserID(ctx)
	if !ok {
		slog.Error("failed to get user ID from context")
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	sessionID := ctx.Query("session_id")

	var shares []table.SharedConversation
	query := a.db.Where("user_id = ?", userID)

	if sessionID != "" {
		query = query.Where("session_id = ?", sessionID)
	}

	if err := query.Order("created_at DESC").Find(&shares).Error; err != nil {
		slog.Error("query.Find() error", "err", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list share links"})
		return
	}

	ctx.JSON(http.StatusOK, shares)
}

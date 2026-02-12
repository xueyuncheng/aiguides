package assistant

import (
	"aiguide/internal/app/aiguide/table"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func TestSharedConversationTable(t *testing.T) {
	db := setupTestDB(t)

	// Create a shared conversation
	shareID := uuid.New().String()
	sharedConv := table.SharedConversation{
		ShareID:   shareID,
		SessionID: "session-123",
		UserID:    1,
		AppName:   "test-agent",
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}

	if err := db.Create(&sharedConv).Error; err != nil {
		t.Fatalf("failed to create shared conversation: %v", err)
	}

	// Retrieve the shared conversation
	var retrieved table.SharedConversation
	if err := db.Where("share_id = ?", shareID).First(&retrieved).Error; err != nil {
		t.Fatalf("failed to retrieve shared conversation: %v", err)
	}

	if retrieved.ShareID != shareID {
		t.Errorf("expected share_id %s, got %s", shareID, retrieved.ShareID)
	}

	if retrieved.SessionID != "session-123" {
		t.Errorf("expected session_id session-123, got %s", retrieved.SessionID)
	}

	if retrieved.UserID != 1 {
		t.Errorf("expected user_id 1, got %d", retrieved.UserID)
	}
}

func TestSharedConversationExpiry(t *testing.T) {
	db := setupTestDB(t)

	// Create an expired shared conversation
	expiredShareID := uuid.New().String()
	expiredConv := table.SharedConversation{
		ShareID:   expiredShareID,
		SessionID: "session-expired",
		UserID:    1,
		AppName:   "test-agent",
		ExpiresAt: time.Now().Add(-1 * time.Hour), // expired 1 hour ago
	}

	if err := db.Create(&expiredConv).Error; err != nil {
		t.Fatalf("failed to create expired shared conversation: %v", err)
	}

	// Create a valid shared conversation
	validShareID := uuid.New().String()
	validConv := table.SharedConversation{
		ShareID:   validShareID,
		SessionID: "session-valid",
		UserID:    1,
		AppName:   "test-agent",
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}

	if err := db.Create(&validConv).Error; err != nil {
		t.Fatalf("failed to create valid shared conversation: %v", err)
	}

	// Retrieve and check expiry status
	var retrievedExpired table.SharedConversation
	if err := db.Where("share_id = ?", expiredShareID).First(&retrievedExpired).Error; err != nil {
		t.Fatalf("failed to retrieve expired conversation: %v", err)
	}

	if !time.Now().After(retrievedExpired.ExpiresAt) {
		t.Errorf("expected conversation to be expired")
	}

	var retrievedValid table.SharedConversation
	if err := db.Where("share_id = ?", validShareID).First(&retrievedValid).Error; err != nil {
		t.Fatalf("failed to retrieve valid conversation: %v", err)
	}

	if time.Now().After(retrievedValid.ExpiresAt) {
		t.Errorf("expected conversation to be valid")
	}
}

func TestListSharesByUser(t *testing.T) {
	db := setupTestDB(t)

	// Create shares for different users
	user1Shares := []table.SharedConversation{
		{
			ShareID:   uuid.New().String(),
			SessionID: "session-1",
			UserID:    1,
			AppName:   "test-agent",
			ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
		},
		{
			ShareID:   uuid.New().String(),
			SessionID: "session-2",
			UserID:    1,
			AppName:   "test-agent",
			ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
		},
	}

	user2Share := table.SharedConversation{
		ShareID:   uuid.New().String(),
		SessionID: "session-3",
		UserID:    2,
		AppName:   "test-agent",
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}

	for _, share := range user1Shares {
		if err := db.Create(&share).Error; err != nil {
			t.Fatalf("failed to create share: %v", err)
		}
	}

	if err := db.Create(&user2Share).Error; err != nil {
		t.Fatalf("failed to create share: %v", err)
	}

	// Query shares for user 1
	var user1Results []table.SharedConversation
	if err := db.Where("user_id = ?", 1).Find(&user1Results).Error; err != nil {
		t.Fatalf("failed to query shares: %v", err)
	}

	if len(user1Results) != 2 {
		t.Errorf("expected 2 shares for user 1, got %d", len(user1Results))
	}

	// Query shares for user 2
	var user2Results []table.SharedConversation
	if err := db.Where("user_id = ?", 2).Find(&user2Results).Error; err != nil {
		t.Fatalf("failed to query shares: %v", err)
	}

	if len(user2Results) != 1 {
		t.Errorf("expected 1 share for user 2, got %d", len(user2Results))
	}
}

func TestListSharesBySession(t *testing.T) {
	db := setupTestDB(t)

	sessionID := "session-123"

	// Create multiple shares for the same session
	shares := []table.SharedConversation{
		{
			ShareID:   uuid.New().String(),
			SessionID: sessionID,
			UserID:    1,
			AppName:   "test-agent",
			ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
		},
		{
			ShareID:   uuid.New().String(),
			SessionID: sessionID,
			UserID:    1,
			AppName:   "test-agent",
			ExpiresAt: time.Now().Add(3 * 24 * time.Hour),
		},
	}

	// Create a share for a different session
	otherShare := table.SharedConversation{
		ShareID:   uuid.New().String(),
		SessionID: "session-456",
		UserID:    1,
		AppName:   "test-agent",
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}

	for _, share := range shares {
		if err := db.Create(&share).Error; err != nil {
			t.Fatalf("failed to create share: %v", err)
		}
	}

	if err := db.Create(&otherShare).Error; err != nil {
		t.Fatalf("failed to create share: %v", err)
	}

	// Query shares for specific session
	var sessionResults []table.SharedConversation
	if err := db.Where("session_id = ?", sessionID).Find(&sessionResults).Error; err != nil {
		t.Fatalf("failed to query shares: %v", err)
	}

	if len(sessionResults) != 2 {
		t.Errorf("expected 2 shares for session %s, got %d", sessionID, len(sessionResults))
	}
}

func TestDeleteShare(t *testing.T) {
	db := setupTestDB(t)

	shareID := uuid.New().String()
	sharedConv := table.SharedConversation{
		ShareID:   shareID,
		SessionID: "session-123",
		UserID:    1,
		AppName:   "test-agent",
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}

	if err := db.Create(&sharedConv).Error; err != nil {
		t.Fatalf("failed to create shared conversation: %v", err)
	}

	// Verify it exists
	var retrieved table.SharedConversation
	if err := db.Where("share_id = ?", shareID).First(&retrieved).Error; err != nil {
		t.Fatalf("failed to retrieve shared conversation: %v", err)
	}

	// Delete the share
	if err := db.Delete(&retrieved).Error; err != nil {
		t.Fatalf("failed to delete shared conversation: %v", err)
	}

	// Verify it's deleted
	var notFound table.SharedConversation
	err := db.Where("share_id = ?", shareID).First(&notFound).Error
	if err == nil {
		t.Errorf("expected share to be deleted, but it still exists")
	}
	if err != gorm.ErrRecordNotFound {
		t.Errorf("expected ErrRecordNotFound, got %v", err)
	}
}

func TestGetSharedConversationNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB(t)

	assistant := &Assistant{
		db: db,
	}

	router := gin.New()
	router.GET("/api/share/:shareId", assistant.GetSharedConversation)

	// Try to get a non-existent share
	req := httptest.NewRequest(http.MethodGet, "/api/share/non-existent-id", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}

	var response map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if response["error"] != "shared conversation not found" {
		t.Errorf("expected error 'shared conversation not found', got %v", response["error"])
	}
}

func TestGetSharedConversationExpired(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB(t)

	// Create an expired share
	expiredShareID := uuid.New().String()
	expiredConv := table.SharedConversation{
		ShareID:   expiredShareID,
		SessionID: "session-expired",
		UserID:    1,
		AppName:   "test-agent",
		ExpiresAt: time.Now().Add(-1 * time.Hour),
	}

	if err := db.Create(&expiredConv).Error; err != nil {
		t.Fatalf("failed to create expired shared conversation: %v", err)
	}

	assistant := &Assistant{
		db: db,
	}

	router := gin.New()
	router.GET("/api/share/:shareId", assistant.GetSharedConversation)

	// Try to get the expired share
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/share/%s", expiredShareID), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusGone {
		t.Errorf("expected status %d, got %d", http.StatusGone, w.Code)
	}

	var response map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if response["error"] != "shared link has expired" {
		t.Errorf("expected error 'shared link has expired', got %v", response["error"])
	}
}

func TestCreateShareRequestValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB(t)

	assistant := &Assistant{
		db:      db,
		session: nil, // session will be nil, causing expected errors
	}

	tests := []struct {
		name           string
		requestBody    map[string]any
		expectedStatus int
		skipTest       bool // Skip tests that require session
	}{
		{
			name: "missing session_id",
			requestBody: map[string]any{
				"agent_id": "test-agent",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "missing agent_id",
			requestBody: map[string]any{
				"session_id": "session-123",
			},
			expectedStatus: http.StatusBadRequest,
		},
		// Skip tests that require actual session as they need a full setup
	}

	for _, tt := range tests {
		if tt.skipTest {
			continue
		}
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()

			// Add middleware to set user ID
			router.Use(func(c *gin.Context) {
				c.Set("user_id", 1)
				c.Next()
			})

			router.POST("/api/assistant/share", assistant.CreateShare)

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/api/assistant/share", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d for test %s (body: %s)", tt.expectedStatus, w.Code, tt.name, w.Body.String())
			}
		})
	}
}

package assistant

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"aiguide/internal/app/aiguide/table"
	"aiguide/internal/pkg/constant"

	"github.com/gin-gonic/gin"
)

func TestListMemoriesAndSummary(t *testing.T) {
	gin.SetMode(gin.TestMode)
	assistant := setupProjectTestAssistant(t)

	seedMemories := []table.UserMemory{
		{UserID: 1, MemoryType: constant.MemoryTypeFact, Content: "用户是 Go 工程师", Importance: 9},
		{UserID: 1, MemoryType: constant.MemoryTypePreference, Content: "偏好简洁回答", Importance: 7},
		{UserID: 1, MemoryType: constant.MemoryTypeContext, Content: "正在开发 AIGuides", Importance: 8},
		{UserID: 2, MemoryType: constant.MemoryTypeFact, Content: "其他用户数据", Importance: 10},
	}
	for _, memory := range seedMemories {
		if err := assistant.db.Create(&memory).Error; err != nil {
			t.Fatalf("failed to create memory: %v", err)
		}
	}

	router := newProjectTestRouter(assistant, func(router *gin.Engine) {
		router.GET("/api/assistant/memories", assistant.ListMemories)
		router.GET("/api/assistant/memories/summary", assistant.GetMemorySummary)
	})

	listReq := httptest.NewRequest(http.MethodGet, "/api/assistant/memories?type=fact&keyword=Go", nil)
	listResp := httptest.NewRecorder()
	router.ServeHTTP(listResp, listReq)

	if listResp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d, body=%s", http.StatusOK, listResp.Code, listResp.Body.String())
	}

	var listResponse ListMemoriesResponse
	if err := json.Unmarshal(listResp.Body.Bytes(), &listResponse); err != nil {
		t.Fatalf("failed to unmarshal list response: %v", err)
	}
	if listResponse.Total != 1 {
		t.Fatalf("expected total 1, got %d", listResponse.Total)
	}
	if len(listResponse.Memories) != 1 {
		t.Fatalf("expected 1 memory, got %d", len(listResponse.Memories))
	}
	if listResponse.Memories[0].Content != "用户是 Go 工程师" {
		t.Fatalf("unexpected memory content: %+v", listResponse.Memories[0])
	}

	summaryReq := httptest.NewRequest(http.MethodGet, "/api/assistant/memories/summary", nil)
	summaryResp := httptest.NewRecorder()
	router.ServeHTTP(summaryResp, summaryReq)

	if summaryResp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d, body=%s", http.StatusOK, summaryResp.Code, summaryResp.Body.String())
	}

	var summary MemorySummaryResponse
	if err := json.Unmarshal(summaryResp.Body.Bytes(), &summary); err != nil {
		t.Fatalf("failed to unmarshal summary response: %v", err)
	}
	if summary.Total != 3 {
		t.Fatalf("expected total 3, got %d", summary.Total)
	}
	if summary.Counts[constant.MemoryTypeFact.String()] != 1 {
		t.Fatalf("expected fact count 1, got %d", summary.Counts[constant.MemoryTypeFact.String()])
	}
	if summary.Counts[constant.MemoryTypePreference.String()] != 1 {
		t.Fatalf("expected preference count 1, got %d", summary.Counts[constant.MemoryTypePreference.String()])
	}
	if summary.LastUpdatedAt == nil {
		t.Fatal("expected last_updated_at to be set")
	}
}

func TestCreateUpdateAndDeleteMemory(t *testing.T) {
	gin.SetMode(gin.TestMode)
	assistant := setupProjectTestAssistant(t)
	router := newProjectTestRouter(assistant, func(router *gin.Engine) {
		router.POST("/api/assistant/memories", assistant.CreateMemory)
		router.PATCH("/api/assistant/memories/:memoryId", assistant.UpdateMemory)
		router.DELETE("/api/assistant/memories/:memoryId", assistant.DeleteMemory)
	})

	createBody, err := json.Marshal(CreateMemoryRequest{
		MemoryType: constant.MemoryTypeFact,
		Content:    "擅长 Go 和微服务",
		Importance: 9,
	})
	if err != nil {
		t.Fatalf("failed to marshal create request: %v", err)
	}

	createReq := httptest.NewRequest(http.MethodPost, "/api/assistant/memories", bytes.NewBuffer(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createResp := httptest.NewRecorder()
	router.ServeHTTP(createResp, createReq)

	if createResp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d, body=%s", http.StatusOK, createResp.Code, createResp.Body.String())
	}

	var created MemoryInfo
	if err := json.Unmarshal(createResp.Body.Bytes(), &created); err != nil {
		t.Fatalf("failed to unmarshal create response: %v", err)
	}
	if created.ID == 0 {
		t.Fatal("expected created memory id to be set")
	}
	if created.Importance != 9 {
		t.Fatalf("expected importance 9, got %d", created.Importance)
	}

	updatedContent := "擅长 Go、微服务和 AI 工具开发"
	updatedType := constant.MemoryTypeContext
	updatedImportance := 10
	updateBody, err := json.Marshal(UpdateMemoryRequest{
		MemoryType: &updatedType,
		Content:    &updatedContent,
		Importance: &updatedImportance,
	})
	if err != nil {
		t.Fatalf("failed to marshal update request: %v", err)
	}

	updateReq := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/api/assistant/memories/%d", created.ID), bytes.NewBuffer(updateBody))
	updateReq.Header.Set("Content-Type", "application/json")
	updateResp := httptest.NewRecorder()
	router.ServeHTTP(updateResp, updateReq)

	if updateResp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d, body=%s", http.StatusOK, updateResp.Code, updateResp.Body.String())
	}

	var updated MemoryInfo
	if err := json.Unmarshal(updateResp.Body.Bytes(), &updated); err != nil {
		t.Fatalf("failed to unmarshal update response: %v", err)
	}
	if updated.MemoryType != constant.MemoryTypeContext {
		t.Fatalf("expected memory type %q, got %q", constant.MemoryTypeContext, updated.MemoryType)
	}
	if updated.Content != updatedContent {
		t.Fatalf("expected content %q, got %q", updatedContent, updated.Content)
	}
	if updated.Importance != updatedImportance {
		t.Fatalf("expected importance %d, got %d", updatedImportance, updated.Importance)
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/assistant/memories/%d", created.ID), nil)
	deleteResp := httptest.NewRecorder()
	router.ServeHTTP(deleteResp, deleteReq)

	if deleteResp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d, body=%s", http.StatusOK, deleteResp.Code, deleteResp.Body.String())
	}

	var count int64
	if err := assistant.db.Model(&table.UserMemory{}).Where("id = ?", created.ID).Count(&count).Error; err != nil {
		t.Fatalf("failed to count memories: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected memory to be deleted, count=%d", count)
	}
}

func TestMemoryEndpointsValidateOwnershipAndInput(t *testing.T) {
	gin.SetMode(gin.TestMode)
	assistant := setupProjectTestAssistant(t)
	otherUserMemory := table.UserMemory{
		UserID:     2,
		MemoryType: constant.MemoryTypeFact,
		Content:    "其他用户的记忆",
		Importance: 6,
	}
	if err := assistant.db.Create(&otherUserMemory).Error; err != nil {
		t.Fatalf("failed to create memory: %v", err)
	}

	router := newProjectTestRouter(assistant, func(router *gin.Engine) {
		router.GET("/api/assistant/memories", assistant.ListMemories)
		router.POST("/api/assistant/memories", assistant.CreateMemory)
		router.PATCH("/api/assistant/memories/:memoryId", assistant.UpdateMemory)
		router.DELETE("/api/assistant/memories/:memoryId", assistant.DeleteMemory)
	})

	invalidListReq := httptest.NewRequest(http.MethodGet, "/api/assistant/memories?type=invalid", nil)
	invalidListResp := httptest.NewRecorder()
	router.ServeHTTP(invalidListResp, invalidListReq)
	if invalidListResp.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, invalidListResp.Code)
	}

	invalidCreateBody := []byte(`{"memory_type":"fact","content":"   ","importance":11}`)
	invalidCreateReq := httptest.NewRequest(http.MethodPost, "/api/assistant/memories", bytes.NewBuffer(invalidCreateBody))
	invalidCreateReq.Header.Set("Content-Type", "application/json")
	invalidCreateResp := httptest.NewRecorder()
	router.ServeHTTP(invalidCreateResp, invalidCreateReq)
	if invalidCreateResp.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, invalidCreateResp.Code)
	}

	updateContent := "不能更新别人的记忆"
	updateBody, err := json.Marshal(UpdateMemoryRequest{Content: &updateContent})
	if err != nil {
		t.Fatalf("failed to marshal update request: %v", err)
	}
	updateReq := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/api/assistant/memories/%d", otherUserMemory.ID), bytes.NewBuffer(updateBody))
	updateReq.Header.Set("Content-Type", "application/json")
	updateResp := httptest.NewRecorder()
	router.ServeHTTP(updateResp, updateReq)
	if updateResp.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, updateResp.Code)
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/assistant/memories/%d", otherUserMemory.ID), nil)
	deleteResp := httptest.NewRecorder()
	router.ServeHTTP(deleteResp, deleteReq)
	if deleteResp.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, deleteResp.Code)
	}

	var memory table.UserMemory
	if err := assistant.db.First(&memory, otherUserMemory.ID).Error; err != nil {
		t.Fatalf("expected memory to remain, got error: %v", err)
	}
	if memory.DeletedAt != nil && memory.DeletedAt.Before(time.Now()) {
		t.Fatal("expected other user's memory to remain undeleted")
	}
}

package assistant

import (
	"aiguide/internal/app/aiguide/table"
	"aiguide/internal/pkg/constant"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const (
	defaultMemoryLimit      = 50
	maxMemoryLimit          = 200
	defaultMemoryImportance = 5
	maxMemoryContentLength  = 2000
)

type MemoryInfo struct {
	ID         int                 `json:"id"`
	MemoryType constant.MemoryType `json:"memory_type"`
	Content    string              `json:"content"`
	Importance int                 `json:"importance"`
	CreatedAt  time.Time           `json:"created_at"`
	UpdatedAt  time.Time           `json:"updated_at"`
}

type ListMemoriesResponse struct {
	Memories []MemoryInfo `json:"memories"`
	Total    int64        `json:"total"`
	Limit    int          `json:"limit"`
	Offset   int          `json:"offset"`
}

type CreateMemoryRequest struct {
	MemoryType constant.MemoryType `json:"memory_type"`
	Content    string              `json:"content" binding:"required"`
	Importance int                 `json:"importance"`
}

type UpdateMemoryRequest struct {
	MemoryType *constant.MemoryType `json:"memory_type"`
	Content    *string              `json:"content"`
	Importance *int                 `json:"importance"`
}

type MemorySummaryResponse struct {
	Total         int64            `json:"total"`
	Counts        map[string]int64 `json:"counts"`
	LastUpdatedAt *time.Time       `json:"last_updated_at,omitempty"`
}

// ListMemories 返回当前用户的记忆列表。
func (a *Assistant) ListMemories(ctx *gin.Context) {
	userID, ok := getContextUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	memoryType, hasType, valid := parseMemoryTypeQuery(ctx.Query("type"))
	if hasType && !valid {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid memory type"})
		return
	}

	limit := parseQueryInt(ctx.Query("limit"), defaultMemoryLimit)
	if limit <= 0 {
		limit = defaultMemoryLimit
	}
	if limit > maxMemoryLimit {
		limit = maxMemoryLimit
	}

	offset := parseQueryInt(ctx.Query("offset"), 0)
	if offset < 0 {
		offset = 0
	}

	keyword := strings.TrimSpace(ctx.Query("keyword"))

	query := a.db.Model(&table.UserMemory{}).Where("user_id = ?", userID)
	if hasType {
		query = query.Where("memory_type = ?", memoryType)
	}
	if keyword != "" {
		query = query.Where("content LIKE ?", "%"+keyword+"%")
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		slog.Error("query.Count() error", "err", err, "user_id", userID)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load memories"})
		return
	}

	var memories []table.UserMemory
	if err := query.Order("importance DESC, updated_at DESC, id DESC").Limit(limit).Offset(offset).Find(&memories).Error; err != nil {
		slog.Error("query.Find() error", "err", err, "user_id", userID)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load memories"})
		return
	}

	response := make([]MemoryInfo, 0, len(memories))
	for _, memory := range memories {
		response = append(response, newMemoryInfo(memory))
	}

	ctx.JSON(http.StatusOK, ListMemoriesResponse{
		Memories: response,
		Total:    total,
		Limit:    limit,
		Offset:   offset,
	})
}

// CreateMemory 为当前用户创建一条记忆。
func (a *Assistant) CreateMemory(ctx *gin.Context) {
	userID, ok := getContextUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req CreateMemoryRequest
	if err := ctx.BindJSON(&req); err != nil {
		slog.Error("ctx.BindJSON() error", "err", err)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	content := strings.TrimSpace(req.Content)
	if content == "" || len([]rune(content)) > maxMemoryContentLength {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid memory content"})
		return
	}

	memoryType := req.MemoryType
	if memoryType == "" {
		memoryType = constant.MemoryTypeFact
	}
	if !memoryType.Valid() {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid memory type"})
		return
	}

	importance := normalizeMemoryImportance(req.Importance)

	memory := table.UserMemory{
		UserID:     userID,
		MemoryType: memoryType,
		Content:    content,
		Importance: importance,
	}
	if err := a.db.Create(&memory).Error; err != nil {
		slog.Error("db.Create() error", "err", err, "user_id", userID)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create memory"})
		return
	}

	ctx.JSON(http.StatusOK, newMemoryInfo(memory))
}

// UpdateMemory 更新当前用户的一条记忆。
func (a *Assistant) UpdateMemory(ctx *gin.Context) {
	userID, ok := getContextUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	memoryID, err := strconv.Atoi(ctx.Param("memoryId"))
	if err != nil || memoryID <= 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid memory id"})
		return
	}

	var req UpdateMemoryRequest
	if err := ctx.BindJSON(&req); err != nil {
		slog.Error("ctx.BindJSON() error", "err", err)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	if req.MemoryType == nil && req.Content == nil && req.Importance == nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "no fields to update"})
		return
	}

	var memory table.UserMemory
	if err := a.db.Where("id = ? AND user_id = ?", memoryID, userID).First(&memory).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "memory not found"})
			return
		}
		slog.Error("db.First() error", "err", err, "user_id", userID, "memory_id", memoryID)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update memory"})
		return
	}

	if req.MemoryType != nil {
		if !req.MemoryType.Valid() {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid memory type"})
			return
		}
		memory.MemoryType = *req.MemoryType
	}

	if req.Content != nil {
		content := strings.TrimSpace(*req.Content)
		if content == "" || len([]rune(content)) > maxMemoryContentLength {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid memory content"})
			return
		}
		memory.Content = content
	}

	if req.Importance != nil {
		if *req.Importance < 1 || *req.Importance > 10 {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid importance"})
			return
		}
		memory.Importance = *req.Importance
	}

	if err := a.db.Save(&memory).Error; err != nil {
		slog.Error("db.Save() error", "err", err, "user_id", userID, "memory_id", memoryID)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update memory"})
		return
	}

	ctx.JSON(http.StatusOK, newMemoryInfo(memory))
}

// DeleteMemory 删除当前用户的一条记忆。
func (a *Assistant) DeleteMemory(ctx *gin.Context) {
	userID, ok := getContextUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	memoryID, err := strconv.Atoi(ctx.Param("memoryId"))
	if err != nil || memoryID <= 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid memory id"})
		return
	}

	result := a.db.Where("id = ? AND user_id = ?", memoryID, userID).Delete(&table.UserMemory{})
	if result.Error != nil {
		slog.Error("db.Delete() error", "err", result.Error, "user_id", userID, "memory_id", memoryID)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete memory"})
		return
	}
	if result.RowsAffected == 0 {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "memory not found"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"id": memoryID})
}

// GetMemorySummary 返回当前用户记忆概览。
func (a *Assistant) GetMemorySummary(ctx *gin.Context) {
	userID, ok := getContextUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	type countRow struct {
		MemoryType constant.MemoryType `json:"memory_type"`
		Count      int64               `json:"count"`
	}

	counts := map[string]int64{
		constant.MemoryTypeFact.String():       0,
		constant.MemoryTypePreference.String(): 0,
		constant.MemoryTypeContext.String():    0,
	}

	var total int64
	if err := a.db.Model(&table.UserMemory{}).Where("user_id = ?", userID).Count(&total).Error; err != nil {
		slog.Error("db.Count() error", "err", err, "user_id", userID)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load memory summary"})
		return
	}

	var rows []countRow
	if err := a.db.Model(&table.UserMemory{}).
		Select("memory_type, COUNT(*) AS count").
		Where("user_id = ?", userID).
		Group("memory_type").
		Scan(&rows).Error; err != nil {
		slog.Error("db.Scan() error", "err", err, "user_id", userID)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load memory summary"})
		return
	}

	for _, row := range rows {
		counts[row.MemoryType.String()] = row.Count
	}

	var latestMemory table.UserMemory
	var lastUpdatedAt *time.Time
	if err := a.db.Where("user_id = ?", userID).Order("updated_at DESC, id DESC").First(&latestMemory).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			slog.Error("db.First() error", "err", err, "user_id", userID)
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load memory summary"})
			return
		}
	} else {
		lastUpdatedAt = &latestMemory.UpdatedAt
	}

	ctx.JSON(http.StatusOK, MemorySummaryResponse{
		Total:         total,
		Counts:        counts,
		LastUpdatedAt: lastUpdatedAt,
	})
}

func newMemoryInfo(memory table.UserMemory) MemoryInfo {
	return MemoryInfo{
		ID:         memory.ID,
		MemoryType: memory.MemoryType,
		Content:    memory.Content,
		Importance: memory.Importance,
		CreatedAt:  memory.CreatedAt,
		UpdatedAt:  memory.UpdatedAt,
	}
}

func parseMemoryTypeQuery(raw string) (constant.MemoryType, bool, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", false, true
	}

	memoryType := constant.MemoryType(trimmed)
	return memoryType, true, memoryType.Valid()
}

func parseQueryInt(raw string, fallback int) int {
	if strings.TrimSpace(raw) == "" {
		return fallback
	}

	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}

	return value
}

func normalizeMemoryImportance(importance int) int {
	if importance < 1 || importance > 10 {
		return defaultMemoryImportance
	}
	return importance
}

package assistant

import (
	"aiguide/internal/app/aiguide/table"
	"aiguide/internal/pkg/constant"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"google.golang.org/adk/session"
	"gorm.io/gorm"
)

const maxProjectNameLength = 80

var errProjectNotFound = errors.New("project not found")

type ProjectInfo struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type CreateProjectRequest struct {
	Name string `json:"name" binding:"required"`
}

type UpdateSessionProjectRequest struct {
	ProjectID *int `json:"project_id"`
}

// ListProjects 返回当前用户的项目列表。
func (a *Assistant) ListProjects(ctx *gin.Context) {
	userID, ok := getContextUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var projects []table.Project
	if err := a.db.Where("user_id = ?", userID).Order("updated_at DESC, id DESC").Find(&projects).Error; err != nil {
		slog.Error("db.Find() error", "err", err, "user_id", userID)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load projects"})
		return
	}

	response := make([]ProjectInfo, 0, len(projects))
	for _, project := range projects {
		response = append(response, ProjectInfo{
			ID:   project.ID,
			Name: project.Name,
		})
	}

	ctx.JSON(http.StatusOK, response)
}

// CreateProject 为当前用户创建一个项目。
func (a *Assistant) CreateProject(ctx *gin.Context) {
	userID, ok := getContextUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req CreateProjectRequest
	if err := ctx.BindJSON(&req); err != nil {
		slog.Error("ctx.BindJSON() error", "err", err)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	projectName := strings.TrimSpace(req.Name)
	if projectName == "" || len([]rune(projectName)) > maxProjectNameLength {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid project name"})
		return
	}

	var existing table.Project
	if err := a.db.Where("user_id = ? AND name = ?", userID, projectName).First(&existing).Error; err == nil {
		ctx.JSON(http.StatusOK, ProjectInfo{ID: existing.ID, Name: existing.Name})
		return
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		slog.Error("db.First() error", "err", err, "user_id", userID, "project_name", projectName)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create project"})
		return
	}

	project := table.Project{
		UserID: userID,
		Name:   projectName,
	}
	if err := a.db.Create(&project).Error; err != nil {
		slog.Error("db.Create() error", "err", err, "user_id", userID, "project_name", projectName)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create project"})
		return
	}

	ctx.JSON(http.StatusOK, ProjectInfo{ID: project.ID, Name: project.Name})
}

// UpdateSessionProject 更新会话所属项目。
func (a *Assistant) UpdateSessionProject(ctx *gin.Context) {
	userID, ok := getContextUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	agentID := ctx.Param("agentId")
	sessionID := ctx.Param("sessionId")

	var req UpdateSessionProjectRequest
	if err := ctx.BindJSON(&req); err != nil {
		slog.Error("ctx.BindJSON() error", "err", err)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	if err := a.ensureProjectOwnership(userID, req.ProjectID); err != nil {
		if errors.Is(err, errProjectNotFound) {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid project_id"})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to validate project"})
		return
	}

	getReq := &session.GetRequest{
		AppName:   agentID,
		UserID:    strconv.Itoa(userID),
		SessionID: sessionID,
	}
	if _, err := a.session.Get(ctx, getReq); err != nil {
		slog.Error("session.Get() error", "err", err, "session_id", sessionID, "user_id", userID)
		ctx.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		return
	}

	if err := a.upsertSessionProjectMeta(sessionID, req.ProjectID); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update session project"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"session_id": sessionID,
		"project_id": req.ProjectID,
	})
}

func getContextUserID(ctx *gin.Context) (int, bool) {
	userIDValue, exists := ctx.Get(constant.ContextKeyUserID)
	if !exists {
		return 0, false
	}

	userID, ok := userIDValue.(int)
	return userID, ok
}

func (a *Assistant) ensureProjectOwnership(userID int, projectID *int) error {
	if projectID == nil {
		return nil
	}

	var project table.Project
	if err := a.db.Where("id = ? AND user_id = ?", *projectID, userID).First(&project).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errProjectNotFound
		}
		slog.Error("db.First() error", "err", err, "user_id", userID, "project_id", *projectID)
		return fmt.Errorf("db.First() error: %w", err)
	}

	return nil
}

func (a *Assistant) upsertSessionProjectMeta(sessionID string, projectID *int) error {
	var meta table.SessionMeta
	if err := a.db.Where("session_id = ?", sessionID).First(&meta).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			slog.Error("db.First() error", "err", err, "session_id", sessionID)
			return fmt.Errorf("db.First() error: %w", err)
		}

		meta = table.SessionMeta{
			SessionID: sessionID,
			ThreadID:  sessionID,
			ProjectID: projectID,
			Version:   1,
		}
		if err := a.db.Create(&meta).Error; err != nil {
			slog.Error("db.Create() error", "err", err, "session_id", sessionID)
			return fmt.Errorf("db.Create() error: %w", err)
		}
		return nil
	}

	updates := map[string]any{
		"project_id": projectID,
	}
	if meta.ThreadID == "" {
		updates["thread_id"] = sessionID
	}
	if meta.Version == 0 {
		updates["version"] = 1
	}

	if err := a.db.Model(&meta).Updates(updates).Error; err != nil {
		slog.Error("db.Model().Updates() error", "err", err, "session_id", sessionID)
		return fmt.Errorf("db.Model().Updates() error: %w", err)
	}

	return nil
}

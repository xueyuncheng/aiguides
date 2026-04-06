package assistant

import (
	"aiguide/internal/app/aiguide/table"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ScheduledTaskInfo is the API response shape for a single scheduled task.
type ScheduledTaskInfo struct {
	ID           int        `json:"id"`
	Title        string     `json:"title"`
	Action       string     `json:"action"`
	ScheduleType string     `json:"schedule_type"`
	RunAt        string     `json:"run_at"`
	Weekday      int        `json:"weekday"`
	Timezone     string     `json:"timezone"`
	TargetEmail  string     `json:"target_email,omitempty"`
	Enabled      bool       `json:"enabled"`
	LastRunAt    *time.Time `json:"last_run_at,omitempty"`
	NextRunAt    time.Time  `json:"next_run_at"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// ListScheduledTasksResponse is the response for listing scheduled tasks.
type ListScheduledTasksResponse struct {
	Tasks []ScheduledTaskInfo `json:"tasks"`
	Total int64               `json:"total"`
}

// UpdateScheduledTaskRequest supports toggling enabled status.
type UpdateScheduledTaskRequest struct {
	Enabled *bool `json:"enabled"`
}

// ListScheduledTasks returns all scheduled tasks for the current user.
func (a *Assistant) ListScheduledTasks(ctx *gin.Context) {
	userID, ok := getContextUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var tasks []table.ScheduledTask
	if err := a.db.Where("user_id = ?", userID).Order("created_at DESC").Find(&tasks).Error; err != nil {
		slog.Error("db.Find() error", "err", err, "user_id", userID)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load scheduled tasks"})
		return
	}

	response := make([]ScheduledTaskInfo, 0, len(tasks))
	for _, t := range tasks {
		response = append(response, newScheduledTaskInfo(t))
	}

	ctx.JSON(http.StatusOK, ListScheduledTasksResponse{
		Tasks: response,
		Total: int64(len(tasks)),
	})
}

// DeleteScheduledTask deletes a scheduled task owned by the current user.
func (a *Assistant) DeleteScheduledTask(ctx *gin.Context) {
	userID, ok := getContextUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	taskID, err := strconv.Atoi(ctx.Param("taskId"))
	if err != nil || taskID <= 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id"})
		return
	}

	result := a.db.Where("id = ? AND user_id = ?", taskID, userID).Delete(&table.ScheduledTask{})
	if result.Error != nil {
		slog.Error("db.Delete() error", "err", result.Error, "user_id", userID, "task_id", taskID)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete scheduled task"})
		return
	}
	if result.RowsAffected == 0 {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "scheduled task not found"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"id": taskID})
}

// UpdateScheduledTask updates the enabled status of a scheduled task.
func (a *Assistant) UpdateScheduledTask(ctx *gin.Context) {
	userID, ok := getContextUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	taskID, err := strconv.Atoi(ctx.Param("taskId"))
	if err != nil || taskID <= 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id"})
		return
	}

	var req UpdateScheduledTaskRequest
	if err := ctx.BindJSON(&req); err != nil {
		slog.Error("ctx.BindJSON() error", "err", err)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	if req.Enabled == nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "no fields to update"})
		return
	}

	var task table.ScheduledTask
	if err := a.db.Where("id = ? AND user_id = ?", taskID, userID).First(&task).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "scheduled task not found"})
			return
		}
		slog.Error("db.First() error", "err", err, "user_id", userID, "task_id", taskID)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load scheduled task"})
		return
	}

	task.Enabled = *req.Enabled
	if err := a.db.Save(&task).Error; err != nil {
		slog.Error("db.Save() error", "err", err, "user_id", userID, "task_id", taskID)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update scheduled task"})
		return
	}

	ctx.JSON(http.StatusOK, newScheduledTaskInfo(task))
}

func newScheduledTaskInfo(t table.ScheduledTask) ScheduledTaskInfo {
	return ScheduledTaskInfo{
		ID:           t.ID,
		Title:        t.Title,
		Action:       t.Action,
		ScheduleType: t.ScheduleType,
		RunAt:        t.RunAt,
		Weekday:      t.Weekday,
		Timezone:     t.Timezone,
		TargetEmail:  t.TargetEmail,
		Enabled:      t.Enabled,
		LastRunAt:    t.LastRunAt,
		NextRunAt:    t.NextRunAt,
		CreatedAt:    t.CreatedAt,
		UpdatedAt:    t.UpdatedAt,
	}
}

// Copyright 2025 AIGuides
// Task management tools for the Planner Agent

package tools

import (
	"encoding/json"
	"fmt"
	"log/slog"

	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
	"gorm.io/gorm"

	"aiguide/internal/app/aiguide/table"
	"aiguide/internal/pkg/constant"
)

// TaskCreate - 创建新任务
type TaskCreateInput struct {
	Title       string `json:"title" jsonschema:"Clear action-oriented task title"`
	Description string `json:"description" jsonschema:"Detailed description with acceptance criteria"`
	DependsOn   []int  `json:"depends_on,omitempty" jsonschema:"List of task IDs that must complete first"`
	Priority    int    `json:"priority,omitempty" jsonschema:"Task priority 0=low 1=medium 2=high default 0"`
}

type TaskCreateOutput struct {
	TaskID  int    `json:"task_id"`
	Message string `json:"message"`
}

func NewTaskCreateTool(db *gorm.DB) (tool.Tool, error) {
	config := functiontool.Config{
		Name:        "task_create",
		Description: "Create a new subtask in the current plan. Use this to break down complex work into manageable pieces.",
	}

	handler := func(ctx tool.Context, input TaskCreateInput) (*TaskCreateOutput, error) {
		// 从上下文获取 session_id
		sessionID, ok := ctx.Value("session_id").(string)
		if !ok || sessionID == "" {
			slog.Error("session_id not found in context")
			return nil, fmt.Errorf("session_id not found in context")
		}

		// 验证优先级
		priority := constant.TaskPriority(input.Priority)
		if !priority.Valid() {
			return nil, fmt.Errorf("invalid priority: %d (must be: %d=%s, %d=%s, %d=%s)",
				input.Priority,
				constant.TaskPriorityLow, constant.TaskPriorityLow.String(),
				constant.TaskPriorityMedium, constant.TaskPriorityMedium.String(),
				constant.TaskPriorityHigh, constant.TaskPriorityHigh.String())
		}

		dependsOnJSON, _ := json.Marshal(input.DependsOn)

		task := &table.Task{
			SessionID:   sessionID,
			Title:       input.Title,
			Description: input.Description,
			Status:      constant.TaskStatusPending,
			DependsOn:   string(dependsOnJSON),
			Priority:    priority,
		}

		if err := db.Create(task).Error; err != nil {
			slog.Error("task_create failed", "err", err)
			return nil, fmt.Errorf("failed to create task: %w", err)
		}

		slog.Info("task created", "task_id", task.ID, "title", task.Title)
		return &TaskCreateOutput{
			TaskID:  task.ID,
			Message: fmt.Sprintf("Task '%s' created successfully (ID: %d)", task.Title, task.ID),
		}, nil
	}

	return functiontool.New(config, handler)
}

// TaskUpdate - 更新任务状态
type TaskUpdateInput struct {
	TaskID int    `json:"task_id" jsonschema:"Task ID to update"`
	Status string `json:"status,omitempty" jsonschema:"New status: pending, in_progress, completed, failed"`
	Result string `json:"result,omitempty" jsonschema:"Task execution result or error message"`
}

type TaskUpdateOutput struct {
	Message string `json:"message"`
}

func NewTaskUpdateTool(db *gorm.DB) (tool.Tool, error) {
	config := functiontool.Config{
		Name:        "task_update",
		Description: "Update the status of a task. Use when starting (in_progress), completing (completed), or marking as failed (failed).",
	}

	handler := func(ctx tool.Context, input TaskUpdateInput) (*TaskUpdateOutput, error) {
		updates := map[string]any{}

		if input.Status != "" {
			// 验证状态
			status := constant.TaskStatus(input.Status)
			if !status.Valid() {
				return nil, fmt.Errorf("invalid status: %s (must be: %s, %s, %s, %s)",
					input.Status,
					constant.TaskStatusPending,
					constant.TaskStatusInProgress,
					constant.TaskStatusCompleted,
					constant.TaskStatusFailed)
			}
			updates["status"] = status
		}

		if input.Result != "" {
			updates["result"] = input.Result
		}

		if len(updates) == 0 {
			return nil, fmt.Errorf("no updates provided")
		}

		result := db.Model(&table.Task{}).Where("id = ?", input.TaskID).Updates(updates)
		if result.Error != nil {
			slog.Error("task_update failed", "err", result.Error)
			return nil, fmt.Errorf("failed to update task: %w", result.Error)
		}

		if result.RowsAffected == 0 {
			return nil, fmt.Errorf("task not found: %d", input.TaskID)
		}

		slog.Info("task updated", "task_id", input.TaskID, "status", input.Status)
		return &TaskUpdateOutput{
			Message: fmt.Sprintf("Task %d updated successfully", input.TaskID),
		}, nil
	}

	return functiontool.New(config, handler)
}

// TaskList - 列出任务
type TaskListInput struct {
	Status string `json:"status,omitempty" jsonschema:"Filter by status (optional): pending, in_progress, completed, failed"`
}

type TaskListOutput struct {
	Tasks []table.Task `json:"tasks"`
	Count int          `json:"count"`
}

func NewTaskListTool(db *gorm.DB) (tool.Tool, error) {
	config := functiontool.Config{
		Name:        "task_list",
		Description: "List all tasks in the current session. Optionally filter by status to see pending, in-progress, completed, or failed tasks.",
	}

	handler := func(ctx tool.Context, input TaskListInput) (*TaskListOutput, error) {
		sessionID, ok := ctx.Value("session_id").(string)
		if !ok || sessionID == "" {
			slog.Error("session_id not found in context")
			return nil, fmt.Errorf("session_id not found in context")
		}

		query := db.Where("session_id = ?", sessionID)
		if input.Status != "" {
			query = query.Where("status = ?", input.Status)
		}

		var tasks []table.Task
		if err := query.Order("priority DESC, created_at ASC").Find(&tasks).Error; err != nil {
			slog.Error("task_list failed", "err", err)
			return nil, fmt.Errorf("failed to list tasks: %w", err)
		}

		return &TaskListOutput{
			Tasks: tasks,
			Count: len(tasks),
		}, nil
	}

	return functiontool.New(config, handler)
}

// TaskGet - 获取单个任务详情
type TaskGetInput struct {
	TaskID int `json:"task_id" jsonschema:"Task ID to retrieve"`
}

type TaskGetOutput struct {
	Task table.Task `json:"task"`
}

func NewTaskGetTool(db *gorm.DB) (tool.Tool, error) {
	config := functiontool.Config{
		Name:        "task_get",
		Description: "Get detailed information about a specific task, including its description, status, dependencies, and results.",
	}

	handler := func(ctx tool.Context, input TaskGetInput) (*TaskGetOutput, error) {
		var task table.Task
		if err := db.Where("id = ?", input.TaskID).First(&task).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return nil, fmt.Errorf("task not found: %d", input.TaskID)
			}
			slog.Error("task_get failed", "err", err)
			return nil, fmt.Errorf("failed to get task: %w", err)
		}

		return &TaskGetOutput{Task: task}, nil
	}

	return functiontool.New(config, handler)
}

// FinishPlanning - Planner Agent 用于标记规划完成
type FinishPlanningInput struct {
	Summary   string `json:"summary" jsonschema:"Brief summary of the plan created"`
	TaskCount int    `json:"task_count" jsonschema:"Total number of tasks created"`
}

type FinishPlanningOutput struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

func NewFinishPlanningTool() (tool.Tool, error) {
	config := functiontool.Config{
		Name:        "finish_planning",
		Description: "Signal that planning is complete and return control to the root agent. Use this after you've created all necessary tasks.",
	}

	handler := func(ctx tool.Context, input FinishPlanningInput) (*FinishPlanningOutput, error) {
		slog.Info("planning finished", "summary", input.Summary, "task_count", input.TaskCount)
		return &FinishPlanningOutput{
			Status:  "completed",
			Message: fmt.Sprintf("Planning completed: %s (%d tasks created)", input.Summary, input.TaskCount),
		}, nil
	}

	return functiontool.New(config, handler)
}

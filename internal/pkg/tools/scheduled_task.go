package tools

import (
	"aiguide/internal/app/aiguide/table"
	"aiguide/internal/pkg/middleware"
	"fmt"
	"log/slog"
	"time"

	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
	"gorm.io/gorm"
)

type ScheduledTaskCreateInput struct {
	Title        string `json:"title" jsonschema:"定时任务标题"`
	Action       string `json:"action" jsonschema:"任务执行内容，例如：每天汇总市场新闻并发送到邮箱"`
	ScheduleType string `json:"schedule_type,omitempty" jsonschema:"调度类型：daily(默认) / weekly / once"`
	RunAt        string `json:"run_at" jsonschema:"执行时间：daily/weekly 使用 HH:MM，once 使用 RFC3339"`
	Weekday      int    `json:"weekday,omitempty" jsonschema:"仅 weekly 需要，0=周日,1=周一,...,6=周六"`
	Timezone     string `json:"timezone,omitempty" jsonschema:"时区，默认 Asia/Shanghai"`
	TargetEmail  string `json:"target_email,omitempty" jsonschema:"可选，目标邮箱地址"`
}

type ScheduledTaskCreateOutput struct {
	TaskID    int    `json:"task_id"`
	NextRunAt string `json:"next_run_at"`
	Message   string `json:"message"`
}

func NewScheduledTaskCreateTool(db *gorm.DB) (tool.Tool, error) {
	config := functiontool.Config{
		Name:        "scheduled_task_create",
		Description: "创建定时任务。支持每天/每周/一次性执行。适用于“每天早上8点发送市场快讯到邮箱”这类需求。",
	}

	handler := func(ctx tool.Context, input ScheduledTaskCreateInput) (*ScheduledTaskCreateOutput, error) {
		userID, ok := middleware.GetUserID(ctx)
		if !ok {
			slog.Error("user_id not found in context")
			return nil, fmt.Errorf("user_id not found in context")
		}

		sessionID, _ := ctx.Value("session_id").(string)

		normalizedInput, err := normalizeScheduledTaskInput(input)
		if err != nil {
			slog.Error("normalizeScheduledTaskInput() error", "err", err)
			return nil, err
		}

		nextRunAt, err := calculateNextRunAt(time.Now(), normalizedInput)
		if err != nil {
			slog.Error("calculateNextRunAt() error", "err", err)
			return nil, err
		}

		scheduledTask := &table.ScheduledTask{
			UserID:       userID,
			SessionID:    sessionID,
			Title:        normalizedInput.Title,
			Action:       normalizedInput.Action,
			ScheduleType: normalizedInput.ScheduleType,
			RunAt:        normalizedInput.RunAt,
			Weekday:      normalizedInput.Weekday,
			Timezone:     normalizedInput.Timezone,
			TargetEmail:  normalizedInput.TargetEmail,
			Enabled:      true,
			NextRunAt:    nextRunAt,
		}

		if err := db.Create(scheduledTask).Error; err != nil {
			slog.Error("db.Create() error", "err", err)
			return nil, fmt.Errorf("failed to create scheduled task: %w", err)
		}

		return &ScheduledTaskCreateOutput{
			TaskID:    scheduledTask.ID,
			NextRunAt: scheduledTask.NextRunAt.Format(time.RFC3339),
			Message:   fmt.Sprintf("定时任务已创建：%s", scheduledTask.Title),
		}, nil
	}

	return functiontool.New(config, handler)
}

type ScheduledTaskListInput struct {
	IncludeDisabled bool `json:"include_disabled,omitempty" jsonschema:"是否包含已禁用任务，默认 false"`
}

type ScheduledTaskListOutput struct {
	Tasks []table.ScheduledTask `json:"tasks"`
	Count int                   `json:"count"`
}

func NewScheduledTaskListTool(db *gorm.DB) (tool.Tool, error) {
	config := functiontool.Config{
		Name:        "scheduled_task_list",
		Description: "查看当前用户的定时任务列表。",
	}

	handler := func(ctx tool.Context, input ScheduledTaskListInput) (*ScheduledTaskListOutput, error) {
		userID, ok := middleware.GetUserID(ctx)
		if !ok {
			slog.Error("user_id not found in context")
			return nil, fmt.Errorf("user_id not found in context")
		}

		query := db.Where("user_id = ?", userID)
		if !input.IncludeDisabled {
			query = query.Where("enabled = ?", true)
		}

		var tasks []table.ScheduledTask
		if err := query.Order("next_run_at ASC").Find(&tasks).Error; err != nil {
			slog.Error("query.Find() error", "err", err)
			return nil, fmt.Errorf("failed to list scheduled tasks: %w", err)
		}

		return &ScheduledTaskListOutput{
			Tasks: tasks,
			Count: len(tasks),
		}, nil
	}

	return functiontool.New(config, handler)
}

func normalizeScheduledTaskInput(input ScheduledTaskCreateInput) (ScheduledTaskCreateInput, error) {
	if input.Title == "" {
		return input, fmt.Errorf("title is required")
	}
	if input.Action == "" {
		return input, fmt.Errorf("action is required")
	}
	if input.ScheduleType == "" {
		input.ScheduleType = "daily"
	}
	if input.Timezone == "" {
		input.Timezone = "Asia/Shanghai"
	}

	_, err := time.LoadLocation(input.Timezone)
	if err != nil {
		return input, fmt.Errorf("invalid timezone: %w", err)
	}

	switch input.ScheduleType {
	case "daily":
		if _, err := time.Parse("15:04", input.RunAt); err != nil {
			return input, fmt.Errorf("invalid run_at for daily schedule, expected HH:MM")
		}
	case "weekly":
		if _, err := time.Parse("15:04", input.RunAt); err != nil {
			return input, fmt.Errorf("invalid run_at for weekly schedule, expected HH:MM")
		}
		if input.Weekday < 0 || input.Weekday > 6 {
			return input, fmt.Errorf("invalid weekday for weekly schedule, expected 0-6")
		}
	case "once":
		if _, err := time.Parse(time.RFC3339, input.RunAt); err != nil {
			return input, fmt.Errorf("invalid run_at for once schedule, expected RFC3339")
		}
	default:
		return input, fmt.Errorf("invalid schedule_type: %s", input.ScheduleType)
	}

	return input, nil
}

func calculateNextRunAt(now time.Time, input ScheduledTaskCreateInput) (time.Time, error) {
	location, err := time.LoadLocation(input.Timezone)
	if err != nil {
		return time.Time{}, fmt.Errorf("time.LoadLocation() error, err = %w", err)
	}
	nowInLocation := now.In(location)

	switch input.ScheduleType {
	case "daily":
		runTime, _ := time.Parse("15:04", input.RunAt)
		candidate := time.Date(nowInLocation.Year(), nowInLocation.Month(), nowInLocation.Day(), runTime.Hour(), runTime.Minute(), 0, 0, location)
		if !candidate.After(nowInLocation) {
			candidate = candidate.Add(24 * time.Hour)
		}
		return candidate, nil
	case "weekly":
		runTime, _ := time.Parse("15:04", input.RunAt)
		delta := (input.Weekday - int(nowInLocation.Weekday()) + 7) % 7
		candidate := time.Date(nowInLocation.Year(), nowInLocation.Month(), nowInLocation.Day(), runTime.Hour(), runTime.Minute(), 0, 0, location).AddDate(0, 0, delta)
		if !candidate.After(nowInLocation) {
			candidate = candidate.AddDate(0, 0, 7)
		}
		return candidate, nil
	case "once":
		candidate, _ := time.Parse(time.RFC3339, input.RunAt)
		if !candidate.After(now) {
			return time.Time{}, fmt.Errorf("run_at must be in the future")
		}
		return candidate, nil
	default:
		return time.Time{}, fmt.Errorf("unsupported schedule_type: %s", input.ScheduleType)
	}
}

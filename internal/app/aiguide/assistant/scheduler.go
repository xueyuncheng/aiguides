package assistant

import (
	"aiguide/internal/app/aiguide/table"
	"aiguide/internal/pkg/constant"
	"aiguide/internal/pkg/tools"
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/runner"
	"google.golang.org/adk/session"
	"google.golang.org/genai"
	"gorm.io/gorm"
)

// schedulerPollInterval is how often the scheduler checks for due tasks.
const schedulerPollInterval = time.Minute

// Scheduler polls the database for due scheduled tasks and executes them
// via the assistant runner. Each task runs in a fresh session so it has
// full tool access (web search, email, etc.) just like an interactive chat.
type Scheduler struct {
	db      *gorm.DB
	runner  *runner.Runner
	session session.Service
}

func newScheduler(db *gorm.DB, r *runner.Runner, s session.Service) *Scheduler {
	return &Scheduler{db: db, runner: r, session: s}
}

// Start launches the scheduler loop as a background goroutine.
// It stops when ctx is cancelled.
func (s *Scheduler) Start(ctx context.Context) {
	go s.loop(ctx)
}

func (s *Scheduler) loop(ctx context.Context) {
	slog.Info("scheduler: started")
	ticker := time.NewTicker(schedulerPollInterval)
	defer ticker.Stop()

	// Fire immediately on startup to catch any tasks that became due while
	// the server was down (e.g. after a restart).
	s.tick(ctx)

	for {
		select {
		case <-ctx.Done():
			slog.Info("scheduler: stopped")
			return
		case <-ticker.C:
			s.tick(ctx)
		}
	}
}

// tick queries for all enabled tasks whose next_run_at is in the past and
// dispatches each one in its own goroutine.
func (s *Scheduler) tick(ctx context.Context) {
	var tasks []table.ScheduledTask
	now := time.Now()

	if err := s.db.Where("enabled = ? AND next_run_at <= ?", true, now).Find(&tasks).Error; err != nil {
		slog.Error("scheduler: failed to query due tasks", "err", err)
		return
	}

	for _, task := range tasks {
		task := task // capture loop variable

		// Pre-advance next_run_at (or disable for once-tasks) before
		// dispatching. This prevents double-dispatch if the task takes
		// longer than one poll interval to complete.
		if err := s.advanceNextRunAt(task, now); err != nil {
			slog.Error("scheduler: failed to advance next_run_at, skipping task",
				"err", err, "task_id", task.ID, "title", task.Title)
			continue
		}

		go func() {
			if err := s.dispatch(ctx, task); err != nil {
				slog.Error("scheduler: task dispatch failed",
					"err", err, "task_id", task.ID, "title", task.Title, "user_id", task.UserID)
			}
		}()
	}
}

// advanceNextRunAt moves the task's next_run_at forward so the scheduler
// does not pick it up again before it finishes. For once-tasks it sets
// enabled=false instead.
func (s *Scheduler) advanceNextRunAt(task table.ScheduledTask, now time.Time) error {
	if task.ScheduleType == "once" {
		return s.db.Model(&task).Update("enabled", false).Error
	}

	input := tools.ScheduledTaskCreateInput{
		ScheduleType: task.ScheduleType,
		RunAt:        task.RunAt,
		Weekday:      task.Weekday,
		Timezone:     task.Timezone,
	}
	nextRunAt, err := tools.CalculateNextRunAt(now, input)
	if err != nil {
		return fmt.Errorf("CalculateNextRunAt() error: %w", err)
	}

	return s.db.Model(&task).Update("next_run_at", nextRunAt).Error
}

// dispatch creates a dedicated session and runs the task action through the
// assistant runner, then records last_run_at on success.
func (s *Scheduler) dispatch(ctx context.Context, task table.ScheduledTask) error {
	if s.runner == nil || s.session == nil {
		return fmt.Errorf("scheduler: runner and session are not initialized")
	}

	slog.Info("scheduler: dispatching task",
		"task_id", task.ID, "title", task.Title, "user_id", task.UserID)

	// Inject the task owner's user_id and the shared db connection into the
	// context so tools that call middleware.GetUserID() and middleware.GetTx()
	// (e.g. send_email, manage_memory) resolve the correct values.
	taskCtx := context.WithValue(ctx, constant.ContextKeyUserID, task.UserID)
	taskCtx = context.WithValue(taskCtx, constant.ContextKeyTx, s.db)

	// Create a fresh session for this execution so the task has its own
	// conversation history and does not pollute an existing user session.
	sessionID := fmt.Sprintf("scheduled-%d-%d", task.ID, time.Now().UnixMilli())
	userIDStr := strconv.Itoa(task.UserID)

	createReq := &session.CreateRequest{
		AppName:   constant.AppNameScheduler.String(),
		UserID:    userIDStr,
		SessionID: sessionID,
		State:     map[string]any{},
	}
	if _, err := s.session.Create(taskCtx, createReq); err != nil {
		return fmt.Errorf("session.Create() error: %w", err)
	}

	// Make session_id available in the context so tools that read it
	// (e.g. scheduled_task_create) work correctly within this run.
	taskCtx = context.WithValue(taskCtx, constant.ContextKeySessionID, sessionID)

	// Build the prompt. If a target email was configured, append it so the
	// executor agent knows where to deliver the result.
	action := task.Action
	if task.TargetEmail != "" {
		action = fmt.Sprintf("%s\n请将结果发送到邮箱：%s", action, task.TargetEmail)
	}
	message := genai.NewContentFromText(action, genai.RoleUser)

	// Run the agent to completion, logging every event for observability.
	runConfig := agent.RunConfig{StreamingMode: agent.StreamingModeNone}
	var runErr error
	eventCount := 0
	for event, err := range s.runner.Run(taskCtx, userIDStr, sessionID, message, runConfig) {
		if err != nil {
			runErr = err
			slog.Error("scheduler: runner error",
				"err", err, "task_id", task.ID, "title", task.Title)
			break
		}
		if event == nil {
			continue
		}
		eventCount++

		// Log function calls (tool invocations).
		if event.Content != nil {
			for _, part := range event.Content.Parts {
				if part.FunctionCall != nil {
					slog.Info("scheduler: tool call",
						"task_id", task.ID, "author", event.Author,
						"tool", part.FunctionCall.Name, "args", part.FunctionCall.Args)
				}
				if part.FunctionResponse != nil {
					slog.Info("scheduler: tool result",
						"task_id", task.ID, "author", event.Author,
						"tool", part.FunctionResponse.Name, "response", part.FunctionResponse.Response)
				}
			}
		}
		// Log final text responses.
		if event.LLMResponse.Content != nil && !event.LLMResponse.Partial {
			for _, part := range event.LLMResponse.Content.Parts {
				if part.Text != "" && !part.Thought {
					slog.Info("scheduler: agent text",
						"task_id", task.ID, "author", event.Author, "text", part.Text)
				}
				if part.FunctionCall != nil {
					slog.Info("scheduler: llm tool call",
						"task_id", task.ID, "author", event.Author,
						"tool", part.FunctionCall.Name, "args", part.FunctionCall.Args)
				}
			}
		}
	}
	slog.Info("scheduler: run finished", "task_id", task.ID, "event_count", eventCount)

	// Record last_run_at regardless of whether the run succeeded.
	now := time.Now()
	if err := s.db.Model(&task).Update("last_run_at", now).Error; err != nil {
		slog.Error("scheduler: failed to update last_run_at",
			"err", err, "task_id", task.ID)
	}

	if runErr != nil {
		return fmt.Errorf("runner error: %w", runErr)
	}

	slog.Info("scheduler: task completed successfully",
		"task_id", task.ID, "title", task.Title, "user_id", task.UserID)
	return nil
}

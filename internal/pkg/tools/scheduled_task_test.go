package tools

import (
	"aiguide/internal/app/aiguide/table"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupScheduledTaskTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open() error: %v", err)
	}

	if err := db.AutoMigrate(&table.ScheduledTask{}); err != nil {
		t.Fatalf("AutoMigrate() error: %v", err)
	}

	return db
}

func TestNewScheduledTaskTools(t *testing.T) {
	db := setupScheduledTaskTestDB(t)

	createTool, err := NewScheduledTaskCreateTool(db)
	if err != nil {
		t.Fatalf("NewScheduledTaskCreateTool() error: %v", err)
	}
	if createTool == nil {
		t.Fatal("NewScheduledTaskCreateTool() returned nil")
	}

	listTool, err := NewScheduledTaskListTool(db)
	if err != nil {
		t.Fatalf("NewScheduledTaskListTool() error: %v", err)
	}
	if listTool == nil {
		t.Fatal("NewScheduledTaskListTool() returned nil")
	}
}

func TestNormalizeScheduledTaskInput_Daily(t *testing.T) {
	input := ScheduledTaskCreateInput{
		Title:  "市场简报",
		Action: "汇总市场消息并发邮件",
		RunAt:  "08:00",
	}

	output, err := normalizeScheduledTaskInput(input)
	if err != nil {
		t.Fatalf("normalizeScheduledTaskInput() error: %v", err)
	}
	if output.ScheduleType != "daily" {
		t.Fatalf("expected default schedule_type=daily, got %s", output.ScheduleType)
	}
	if output.Timezone == "" {
		t.Fatal("expected default timezone to be set")
	}
}

func TestCalculateNextRunAt_Daily(t *testing.T) {
	now := time.Date(2026, 2, 25, 9, 0, 0, 0, time.UTC)
	input := ScheduledTaskCreateInput{
		Title:        "市场简报",
		Action:       "汇总市场消息并发邮件",
		ScheduleType: "daily",
		RunAt:        "08:00",
		Timezone:     "UTC",
	}

	nextRunAt, err := CalculateNextRunAt(now, input)
	if err != nil {
		t.Fatalf("calculateNextRunAt() error: %v", err)
	}
	expected := time.Date(2026, 2, 26, 8, 0, 0, 0, time.UTC)
	if !nextRunAt.Equal(expected) {
		t.Fatalf("calculateNextRunAt()=%s, expected %s", nextRunAt, expected)
	}
}

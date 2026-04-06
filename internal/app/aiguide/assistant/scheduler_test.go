package assistant

import (
	"aiguide/internal/app/aiguide/table"
	"context"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupSchedulerTestDB(t *testing.T) *gorm.DB {
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

func TestScheduler_AdvanceNextRunAt_Daily(t *testing.T) {
	db := setupSchedulerTestDB(t)
	s := newScheduler(db, nil, nil)

	now := time.Date(2026, 2, 25, 9, 0, 0, 0, time.UTC)
	task := table.ScheduledTask{
		UserID:       1,
		Title:        "daily report",
		Action:       "send daily report",
		ScheduleType: "daily",
		RunAt:        "08:00",
		Timezone:     "UTC",
		Enabled:      true,
		NextRunAt:    now.Add(-time.Hour), // overdue
	}
	if err := db.Create(&task).Error; err != nil {
		t.Fatalf("db.Create() error: %v", err)
	}

	if err := s.advanceNextRunAt(task, now); err != nil {
		t.Fatalf("advanceNextRunAt() error: %v", err)
	}

	var updated table.ScheduledTask
	if err := db.First(&updated, task.ID).Error; err != nil {
		t.Fatalf("db.First() error: %v", err)
	}

	// daily at 08:00 UTC after 09:00 → tomorrow 08:00
	expected := time.Date(2026, 2, 26, 8, 0, 0, 0, time.UTC)
	if !updated.NextRunAt.Equal(expected) {
		t.Errorf("NextRunAt = %v, want %v", updated.NextRunAt, expected)
	}
	if !updated.Enabled {
		t.Error("Enabled should remain true for a daily task")
	}
}

func TestScheduler_AdvanceNextRunAt_Once(t *testing.T) {
	db := setupSchedulerTestDB(t)
	s := newScheduler(db, nil, nil)

	now := time.Now()
	task := table.ScheduledTask{
		UserID:       1,
		Title:        "one-time report",
		Action:       "send report",
		ScheduleType: "once",
		RunAt:        now.Add(-time.Hour).Format(time.RFC3339),
		Timezone:     "UTC",
		Enabled:      true,
		NextRunAt:    now.Add(-time.Hour),
	}
	if err := db.Create(&task).Error; err != nil {
		t.Fatalf("db.Create() error: %v", err)
	}

	if err := s.advanceNextRunAt(task, now); err != nil {
		t.Fatalf("advanceNextRunAt() error: %v", err)
	}

	var updated table.ScheduledTask
	if err := db.First(&updated, task.ID).Error; err != nil {
		t.Fatalf("db.First() error: %v", err)
	}

	if updated.Enabled {
		t.Error("once task should be disabled after advanceNextRunAt")
	}
}

func TestScheduler_AdvanceNextRunAt_Weekly(t *testing.T) {
	db := setupSchedulerTestDB(t)
	s := newScheduler(db, nil, nil)

	// 2026-02-25 is a Wednesday (weekday=3). Schedule weekly on Wednesday at 08:00.
	// Now is 09:00 Wednesday → next should be next Wednesday.
	now := time.Date(2026, 2, 25, 9, 0, 0, 0, time.UTC)
	task := table.ScheduledTask{
		UserID:       1,
		Title:        "weekly report",
		Action:       "send weekly report",
		ScheduleType: "weekly",
		RunAt:        "08:00",
		Weekday:      3, // Wednesday
		Timezone:     "UTC",
		Enabled:      true,
		NextRunAt:    now.Add(-time.Hour),
	}
	if err := db.Create(&task).Error; err != nil {
		t.Fatalf("db.Create() error: %v", err)
	}

	if err := s.advanceNextRunAt(task, now); err != nil {
		t.Fatalf("advanceNextRunAt() error: %v", err)
	}

	var updated table.ScheduledTask
	if err := db.First(&updated, task.ID).Error; err != nil {
		t.Fatalf("db.First() error: %v", err)
	}

	expected := time.Date(2026, 3, 4, 8, 0, 0, 0, time.UTC) // next Wednesday
	if !updated.NextRunAt.Equal(expected) {
		t.Errorf("NextRunAt = %v, want %v", updated.NextRunAt, expected)
	}
}

// TestScheduler_Tick_PicksUpDueTasks verifies that tick() queries for tasks
// whose next_run_at is in the past (and advances them) without actually
// invoking the runner (which is nil in this test).
func TestScheduler_Tick_PicksUpDueTasks(t *testing.T) {
	db := setupSchedulerTestDB(t)
	// runner and session are nil – tick() must advance next_run_at via
	// advanceNextRunAt() before spawning dispatch goroutines; if it errors
	// it skips dispatch. Here once-tasks are disabled synchronously before
	// any goroutine is spawned, so we can assert on DB state directly.

	s := newScheduler(db, nil, nil)

	now := time.Now()

	// Due daily task
	dueTask := table.ScheduledTask{
		UserID:       1,
		Title:        "due task",
		Action:       "do something",
		ScheduleType: "daily",
		RunAt:        "08:00",
		Timezone:     "UTC",
		Enabled:      true,
		NextRunAt:    now.Add(-time.Minute),
	}
	// Future task – must not be dispatched
	futureTask := table.ScheduledTask{
		UserID:       1,
		Title:        "future task",
		Action:       "do something later",
		ScheduleType: "daily",
		RunAt:        "08:00",
		Timezone:     "UTC",
		Enabled:      true,
		NextRunAt:    now.Add(time.Hour),
	}

	if err := db.Create(&dueTask).Error; err != nil {
		t.Fatalf("db.Create(dueTask) error: %v", err)
	}
	if err := db.Create(&futureTask).Error; err != nil {
		t.Fatalf("db.Create(futureTask) error: %v", err)
	}

	originalNextRunAt := dueTask.NextRunAt
	originalFutureNextRunAt := futureTask.NextRunAt

	s.tick(context.Background())

	// Give the goroutine spawned by tick a moment to start (dispatch will
	// immediately fail because runner is nil, but that's fine – we care about
	// the DB state after advanceNextRunAt, which runs synchronously in tick).
	time.Sleep(50 * time.Millisecond)

	var updatedDue, updatedFuture table.ScheduledTask
	if err := db.First(&updatedDue, dueTask.ID).Error; err != nil {
		t.Fatalf("db.First(dueTask) error: %v", err)
	}
	if err := db.First(&updatedFuture, futureTask.ID).Error; err != nil {
		t.Fatalf("db.First(futureTask) error: %v", err)
	}

	// Due task's next_run_at must have been advanced
	if !updatedDue.NextRunAt.After(originalNextRunAt) {
		t.Errorf("due task NextRunAt was not advanced: got %v, original %v",
			updatedDue.NextRunAt, originalNextRunAt)
	}

	// Future task must be untouched
	if !updatedFuture.NextRunAt.Equal(originalFutureNextRunAt) {
		t.Errorf("future task NextRunAt was unexpectedly changed: got %v, original %v",
			updatedFuture.NextRunAt, originalFutureNextRunAt)
	}
}

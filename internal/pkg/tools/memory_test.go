package tools

import (
	"aiguide/internal/app/aiguide/table"
	"aiguide/internal/pkg/constant"
	"context"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestDB 创建测试数据库
func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// 自动迁移
	if err := db.AutoMigrate(&table.UserMemory{}); err != nil {
		t.Fatalf("Failed to migrate test database: %v", err)
	}

	return db
}

func TestNewMemoryTool(t *testing.T) {
	db := setupTestDB(t)
	tool, err := NewMemoryTool(db)
	if err != nil {
		t.Fatalf("Failed to create memory tool: %v", err)
	}

	if tool == nil {
		t.Fatal("NewMemoryTool() returned nil tool")
	}
}

func TestNewMemoryTool_NilDB(t *testing.T) {
	_, err := NewMemoryTool(nil)
	if err == nil {
		t.Error("Expected error when passing nil database")
	}
}

func TestMemoryHandlerHandleMemoryRequiresUserIDInContext(t *testing.T) {
	db := setupTestDB(t)
	handler := &memoryHandler{db: db}

	output, err := handler.handleMemory(context.Background(), &MemoryInput{
		Action:  constant.MemoryActionRetrieve,
		Content: "ignored",
	})
	if err != nil {
		t.Fatalf("handleMemory() error = %v", err)
	}

	if output.Success {
		t.Fatal("handleMemory() succeeded without user_id in context")
	}

	if output.Error != "user_id not found in context" {
		t.Fatalf("handleMemory() error = %q, want %q", output.Error, "user_id not found in context")
	}
}

func TestMemoryHandlerUsesUserIDFromContext(t *testing.T) {
	db := setupTestDB(t)
	handler := &memoryHandler{db: db}

	ctxUser1 := context.WithValue(context.Background(), constant.ContextKeyUserID, 1)
	ctxUser2 := context.WithValue(context.Background(), constant.ContextKeyUserID, 2)

	saveOutput, err := handler.handleMemory(ctxUser1, &MemoryInput{
		Action:     constant.MemoryActionSave,
		Content:    "prefers tea",
		MemoryType: constant.MemoryTypePreference,
	})
	if err != nil {
		t.Fatalf("save handleMemory() error = %v", err)
	}
	if !saveOutput.Success {
		t.Fatalf("save handleMemory() failed: %+v", saveOutput)
	}

	var stored []table.UserMemory
	if err := db.Order("id ASC").Find(&stored).Error; err != nil {
		t.Fatalf("db.Find() error = %v", err)
	}
	if len(stored) != 1 {
		t.Fatalf("stored memory count = %d, want 1", len(stored))
	}
	if stored[0].UserID != 1 {
		t.Fatalf("stored memory user_id = %d, want 1", stored[0].UserID)
	}

	retrieveOutputUser1, err := handler.handleMemory(ctxUser1, &MemoryInput{
		Action: constant.MemoryActionRetrieve,
	})
	if err != nil {
		t.Fatalf("retrieve user1 handleMemory() error = %v", err)
	}
	if !retrieveOutputUser1.Success {
		t.Fatalf("retrieve user1 handleMemory() failed: %+v", retrieveOutputUser1)
	}
	if len(retrieveOutputUser1.Memories) != 1 {
		t.Fatalf("retrieve user1 memories = %d, want 1", len(retrieveOutputUser1.Memories))
	}

	retrieveOutputUser2, err := handler.handleMemory(ctxUser2, &MemoryInput{
		Action: constant.MemoryActionRetrieve,
	})
	if err != nil {
		t.Fatalf("retrieve user2 handleMemory() error = %v", err)
	}
	if !retrieveOutputUser2.Success {
		t.Fatalf("retrieve user2 handleMemory() failed: %+v", retrieveOutputUser2)
	}
	if len(retrieveOutputUser2.Memories) != 0 {
		t.Fatalf("retrieve user2 memories = %d, want 0", len(retrieveOutputUser2.Memories))
	}

	updateOutputUser2, err := handler.handleMemory(ctxUser2, &MemoryInput{
		Action:   constant.MemoryActionUpdate,
		MemoryID: stored[0].ID,
		Content:  "prefers coffee",
	})
	if err != nil {
		t.Fatalf("update user2 handleMemory() error = %v", err)
	}
	if updateOutputUser2.Success {
		t.Fatalf("update user2 unexpectedly succeeded: %+v", updateOutputUser2)
	}

	deleteOutputUser2, err := handler.handleMemory(ctxUser2, &MemoryInput{
		Action:   constant.MemoryActionDelete,
		MemoryID: stored[0].ID,
	})
	if err != nil {
		t.Fatalf("delete user2 handleMemory() error = %v", err)
	}
	if deleteOutputUser2.Success {
		t.Fatalf("delete user2 unexpectedly succeeded: %+v", deleteOutputUser2)
	}

	var persisted table.UserMemory
	if err := db.First(&persisted, stored[0].ID).Error; err != nil {
		t.Fatalf("db.First() error = %v", err)
	}
	if persisted.Content != "prefers tea" {
		t.Fatalf("persisted content = %q, want %q", persisted.Content, "prefers tea")
	}
}

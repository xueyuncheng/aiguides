package tools

import (
	"aiguide/internal/app/aiguide/table"
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

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

func TestGetUserMemoriesAsContext(t *testing.T) {
	db := setupTestDB(t)

	// 保存几条记忆
	memories := []table.UserMemory{
		{UserID: 1, MemoryType: "fact", Content: "用户是软件工程师", Importance: 8},
		{UserID: 1, MemoryType: "preference", Content: "喜欢简洁的代码", Importance: 7},
		{UserID: 1, MemoryType: "context", Content: "正在开发Go项目", Importance: 6},
	}

	for _, mem := range memories {
		if err := db.Create(&mem).Error; err != nil {
			t.Fatalf("Failed to create memory: %v", err)
		}
	}

	// 获取上下文
	context, err := GetUserMemoriesAsContext(db, 1)
	if err != nil {
		t.Fatalf("Failed to get context: %v", err)
	}

	if context == "" {
		t.Error("Expected non-empty context")
	}

	// 检查是否包含关键信息
	if !contains(context, "用户是软件工程师") {
		t.Error("Context should contain fact memory")
	}

	if !contains(context, "喜欢简洁的代码") {
		t.Error("Context should contain preference memory")
	}

	if !contains(context, "正在开发Go项目") {
		t.Error("Context should contain context memory")
	}
}

func TestGetUserMemoriesAsContext_NoMemories(t *testing.T) {
	db := setupTestDB(t)

	// 获取不存在用户的记忆
	context, err := GetUserMemoriesAsContext(db, 999)
	if err != nil {
		t.Fatalf("Failed to get context: %v", err)
	}

	if context != "" {
		t.Errorf("Expected empty context for user with no memories, got: %s", context)
	}
}

// contains 检查字符串是否包含子串
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

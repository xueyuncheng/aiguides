package assistant

import (
	"aiguide/internal/app/aiguide/table"
	"aiguide/internal/pkg/tools"
	"testing"

	"google.golang.org/adk/model"
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

func TestNewSearchAgent(t *testing.T) {
	// This test verifies that the search agent can be created
	// without errors when a nil genaiClient is passed (basic structure test)
	db := setupTestDB(t)
	webSearchConfig := tools.WebSearchConfig{
		SearXNG: tools.SearXNGConfig{
			InstanceURL: "https://searx.be",
		},
	}

	agent, err := NewAssistantAgent(nil, nil, db, true, webSearchConfig)
	if err != nil {
		t.Fatalf("NewSearchAgent() error = %v", err)
	}

	if agent == nil {
		t.Fatal("NewSearchAgent() returned nil agent")
	}
}

func TestNewSearchAgentWithModel(t *testing.T) {
	// Test with a mock model to verify the structure
	// In a real scenario, you would use a proper mock model
	db := setupTestDB(t)
	webSearchConfig := tools.WebSearchConfig{
		SearXNG: tools.SearXNGConfig{
			InstanceURL: "https://searx.be",
		},
	}

	agent, err := NewAssistantAgent(model.LLM(nil), nil, db, true, webSearchConfig)
	if err != nil {
		t.Fatalf("NewSearchAgent() with model error = %v", err)
	}

	if agent == nil {
		t.Fatal("NewSearchAgent() returned nil agent")
	}
}

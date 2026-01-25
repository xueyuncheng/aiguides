package assistant

import (
	"aiguide/internal/app/aiguide/table"
	"aiguide/internal/pkg/tools"
	"context"
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
	if err := db.AutoMigrate(&table.UserMemory{}, &table.Task{}); err != nil {
		t.Fatalf("Failed to migrate test database: %v", err)
	}

	return db
}

func TestNewSearchAgent(t *testing.T) {
	// This test verifies that the search agent can be created
	// without errors when a nil genaiClient is passed (basic structure test)
	ctx := context.Background()
	db := setupTestDB(t)
	webSearchConfig := tools.WebSearchConfig{
		SearXNG: tools.SearXNGConfig{
			InstanceURL: "https://searx.be",
		},
	}

	var _ context.Context = ctx

	agentConfig := &AssistantAgentConfig{
		Model:             nil,
		GenaiClient:       nil,
		DB:                db,
		MockImageGen:      true,
		MockEmailIMAPConn: true,
		WebSearchConfig:   webSearchConfig,
	}
	agent, err := NewAssistantAgent(agentConfig)
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
	ctx := context.Background()
	db := setupTestDB(t)
	webSearchConfig := tools.WebSearchConfig{
		SearXNG: tools.SearXNGConfig{
			InstanceURL: "https://searx.be",
		},
	}

	var _ context.Context = ctx

	agentConfig := &AssistantAgentConfig{
		Model:             model.LLM(nil),
		GenaiClient:       nil,
		DB:                db,
		MockImageGen:      true,
		MockEmailIMAPConn: true,
		WebSearchConfig:   webSearchConfig,
	}
	agent, err := NewAssistantAgent(agentConfig)
	if err != nil {
		t.Fatalf("NewSearchAgent() with model error = %v", err)
	}

	if agent == nil {
		t.Fatal("NewSearchAgent() returned nil agent")
	}
}

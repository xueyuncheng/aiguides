package assistant

import (
	"aiguide/internal/app/aiguide/table"
	"aiguide/internal/pkg/storage"
	"aiguide/internal/pkg/tools"
	"context"
	"strings"
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
	if err := db.AutoMigrate(&table.UserMemory{}, &table.Task{}, &table.SharedConversation{}); err != nil {
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
		FileStore:         mustTestFileStore(t),
		PDFWorkDir:        t.TempDir(),
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
		FileStore:         mustTestFileStore(t),
		PDFWorkDir:        t.TempDir(),
	}
	agent, err := NewAssistantAgent(agentConfig)
	if err != nil {
		t.Fatalf("NewSearchAgent() with model error = %v", err)
	}

	if agent == nil {
		t.Fatal("NewSearchAgent() returned nil agent")
	}
}

func TestAssistantAgentInstructionOnlyMentionsRootTools(t *testing.T) {
	requiredTools := []string{
		"`task_list`",
		"`task_get`",
		"`manage_memory`",
	}

	for _, toolName := range requiredTools {
		if !strings.Contains(assistantAgentInstruction, toolName) {
			t.Errorf("assistantAgentInstruction missing root tool %s", toolName)
		}
	}

	executorOnlyTools := []string{
		"`current_time`",
		"`image_gen`",
		"`email_query`",
		"`send_email`",
		"`web_search`",
		"`exa_search`",
		"`web_fetch`",
	}

	for _, toolName := range executorOnlyTools {
		if strings.Contains(assistantAgentInstruction, toolName) {
			t.Errorf("assistantAgentInstruction should not advertise executor-only tool %s", toolName)
		}
	}

	plannerOnlyPhrases := []string{
		"规划子流程",
		"先规划再执行",
		"独立的规划子流程",
	}

	for _, phrase := range plannerOnlyPhrases {
		if strings.Contains(assistantAgentInstruction, phrase) {
			t.Errorf("assistantAgentInstruction should not reference planner-only flow %q", phrase)
		}
	}
}

func TestExecutorAgentInstructionMentionsMemoryTool(t *testing.T) {
	if !strings.Contains(executorAgentInstruction, "`manage_memory`") {
		t.Fatal("executorAgentInstruction missing executor tool `manage_memory`")
	}
}

func TestNewAssistantAgentUsesOnlyExecutorSubAgent(t *testing.T) {
	db := setupTestDB(t)
	webSearchConfig := tools.WebSearchConfig{
		SearXNG: tools.SearXNGConfig{
			InstanceURL: "https://searx.be",
		},
	}

	assistantAgent, err := NewAssistantAgent(&AssistantAgentConfig{
		Model:             nil,
		GenaiClient:       nil,
		DB:                db,
		MockImageGen:      true,
		MockEmailIMAPConn: true,
		WebSearchConfig:   webSearchConfig,
		FileStore:         mustTestFileStore(t),
		PDFWorkDir:        t.TempDir(),
	})
	if err != nil {
		t.Fatalf("NewAssistantAgent() error = %v", err)
	}

	subAgents := assistantAgent.SubAgents()
	if got, want := len(subAgents), 1; got != want {
		t.Fatalf("len(SubAgents()) = %d, want %d", got, want)
	}

	subAgentNames := make([]string, 0, len(subAgents))
	for _, subAgent := range subAgents {
		subAgentNames = append(subAgentNames, subAgent.Name())
	}

	if len(subAgentNames) != 1 || subAgentNames[0] != "executor" {
		t.Fatalf("SubAgents() names = %v, want [executor]", subAgentNames)
	}
}

func mustTestFileStore(t *testing.T) *storage.LocalFileStore {
	t.Helper()

	store, err := storage.NewLocalFileStore(t.TempDir())
	if err != nil {
		t.Fatalf("storage.NewLocalFileStore() error = %v", err)
	}

	return store
}

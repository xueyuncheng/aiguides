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

func TestAssistantAgentInstructionMentionsMemoryTool(t *testing.T) {
	if !strings.Contains(assistantAgentInstruction, "`manage_memory`") {
		t.Fatal("assistantAgentInstruction missing tool `manage_memory`")
	}
}

func TestAssistantAgentInstructionMentionsAudioTranscribeTool(t *testing.T) {
	if !strings.Contains(assistantAgentInstruction, "`audio_transcribe`") {
		t.Fatal("assistantAgentInstruction missing tool `audio_transcribe`")
	}
}

func TestAssistantAgentInstructionMentionsFileDownloadTool(t *testing.T) {
	if !strings.Contains(assistantAgentInstruction, "`file_download`") {
		t.Fatal("assistantAgentInstruction missing tool `file_download`")
	}
}

func TestNewAssistantAgentHasNoSubAgents(t *testing.T) {
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

	if got := len(assistantAgent.SubAgents()); got != 0 {
		t.Fatalf("len(SubAgents()) = %d, want 0 (single agent, no sub-agents)", got)
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

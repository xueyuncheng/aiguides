package assistant

import (
	_ "embed"
	"fmt"
	"log/slog"

	"aiguide/internal/pkg/storage"
	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/model"
	"google.golang.org/genai"
	"gorm.io/gorm"

	"aiguide/internal/pkg/tools"
)

//go:embed assistant_agent_prompt.md
var assistantAgentInstruction string

// AssistantAgentConfig contains configuration for the root agent and its subagents.
type AssistantAgentConfig struct {
	Model             model.LLM
	GenaiClient       *genai.Client
	DB                *gorm.DB
	MockImageGen      bool
	MockEmailIMAPConn bool
	WebSearchConfig   tools.WebSearchConfig
	ExaConfig         tools.ExaConfig
	FileStore         storage.FileStore
	PDFWorkDir        string
}

// NewAssistantAgent creates the root agent with the executor as its subagent.
// The root agent is a pure orchestrator with no tools of its own — all tool
// execution is delegated to the executor subagent.
func NewAssistantAgent(config *AssistantAgentConfig) (agent.Agent, error) {
	if config == nil {
		slog.Error("config parameter is nil")
		return nil, fmt.Errorf("config cannot be nil")
	}

	executorConfig := &ExecutorAgentConfig{
		Model:           config.Model,
		GenaiClient:     config.GenaiClient,
		DB:              config.DB,
		MockImageGen:    config.MockImageGen,
		WebSearchConfig: config.WebSearchConfig,
		ExaConfig:       config.ExaConfig,
		FileStore:       config.FileStore,
		PDFWorkDir:      config.PDFWorkDir,
	}
	executorAgent, err := NewExecutorAgent(executorConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create executor agent: %w", err)
	}

	rootAgentConfig := llmagent.Config{
		Name:        "root_agent",
		Model:       config.Model,
		Description: "Main conversational agent that answers directly when no tools are needed and delegates all tool-based work to the executor subagent",
		Instruction: assistantAgentInstruction,
		GenerateContentConfig: &genai.GenerateContentConfig{
			ThinkingConfig: &genai.ThinkingConfig{
				IncludeThoughts: true,
			},
		},
		// No tools on root — executor owns all tool calls.
		SubAgents: []agent.Agent{
			executorAgent,
		},
	}

	rootAgent, err := llmagent.New(rootAgentConfig)
	if err != nil {
		slog.Error("failed to create root agent", "err", err)
		return nil, fmt.Errorf("failed to create root agent: %w", err)
	}

	slog.Info("root agent created successfully with executor subagent")
	return rootAgent, nil
}

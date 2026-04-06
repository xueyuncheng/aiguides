package assistant

import (
	_ "embed"
	"fmt"
	"log/slog"

	"aiguide/internal/pkg/storage"
	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/model"
	"google.golang.org/adk/tool"
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
func NewAssistantAgent(config *AssistantAgentConfig) (agent.Agent, error) {
	if config == nil {
		slog.Error("config parameter is nil")
		return nil, fmt.Errorf("config cannot be nil")
	}
	// 创建 Executor Agent（所有需要工具的执行都交给它）
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

	// Root Agent 的工具：任务查询 + 记忆管理
	taskListTool, err := tools.NewTaskListTool(config.DB)
	if err != nil {
		return nil, fmt.Errorf("failed to create task_list tool: %w", err)
	}

	taskGetTool, err := tools.NewTaskGetTool(config.DB)
	if err != nil {
		return nil, fmt.Errorf("failed to create task_get tool: %w", err)
	}

	scheduledTaskCreateTool, err := tools.NewScheduledTaskCreateTool(config.DB)
	if err != nil {
		return nil, fmt.Errorf("failed to create scheduled_task_create tool: %w", err)
	}

	scheduledTaskListTool, err := tools.NewScheduledTaskListTool(config.DB)
	if err != nil {
		return nil, fmt.Errorf("failed to create scheduled_task_list tool: %w", err)
	}

	// 创建记忆管理工具
	memoryTool, err := tools.NewMemoryTool(config.DB)
	if err != nil {
		return nil, fmt.Errorf("tools.NewMemoryTool() error, err = %w", err)
	}

	// 创建 Root Agent 配置
	rootAgentConfig := llmagent.Config{
		Name:        "root_agent",
		Model:       config.Model,
		Description: "Main conversational agent that answers directly when no tools are needed and delegates tool-based work to the executor",
		Instruction: assistantAgentInstruction,
		GenerateContentConfig: &genai.GenerateContentConfig{
			ThinkingConfig: &genai.ThinkingConfig{
				IncludeThoughts: true,
			},
		},
		// Root Agent 的工具：任务查询 + 记忆管理
		Tools: []tool.Tool{
			taskListTool,
			taskGetTool,
			scheduledTaskCreateTool,
			scheduledTaskListTool,
			memoryTool,
		},
		// Root 仅保留一个执行子代理，避免规划/执行边界混乱。
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

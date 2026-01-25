// Copyright 2025 AIGuides
// Executor Agent - specialized in executing tasks using available tools

package assistant

import (
	_ "embed"
	"fmt"
	"log/slog"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/model"
	"google.golang.org/adk/tool"
	"google.golang.org/genai"
	"gorm.io/gorm"

	"aiguide/internal/pkg/tools"
)

//go:embed executor_agent_prompt.md
var executorAgentInstruction string

// ExecutorAgentConfig configures the Executor Agent
type ExecutorAgentConfig struct {
	Model           model.LLM
	GenaiClient     *genai.Client
	DB              *gorm.DB
	MockImageGen    bool
	WebSearchConfig tools.WebSearchConfig
}

// NewExecutorAgent creates a specialized execution agent with all functional tools
func NewExecutorAgent(config *ExecutorAgentConfig) (agent.Agent, error) {
	if config == nil {
		slog.Error("config parameter is nil")
		return nil, fmt.Errorf("config cannot be nil")
	}
	// 创建功能工具
	imageGenTool, err := tools.NewImageGenTool(config.GenaiClient, config.MockImageGen)
	if err != nil {
		return nil, fmt.Errorf("failed to create image gen tool: %w", err)
	}

	emailQueryTool, err := tools.NewEmailQueryTool()
	if err != nil {
		return nil, fmt.Errorf("failed to create email query tool: %w", err)
	}

	webSearchTool, err := tools.NewWebSearchTool(config.WebSearchConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create web search tool: %w", err)
	}

	webFetchTool, err := tools.NewWebFetchTool()
	if err != nil {
		return nil, fmt.Errorf("failed to create web fetch tool: %w", err)
	}

	currentTimeTool, err := tools.NewCurrentTimeTool()
	if err != nil {
		return nil, fmt.Errorf("failed to create current time tool: %w", err)
	}

	// 任务查询和更新工具
	taskListTool, err := tools.NewTaskListTool(config.DB)
	if err != nil {
		return nil, fmt.Errorf("failed to create task_list tool: %w", err)
	}

	taskGetTool, err := tools.NewTaskGetTool(config.DB)
	if err != nil {
		return nil, fmt.Errorf("failed to create task_get tool: %w", err)
	}

	taskUpdateTool, err := tools.NewTaskUpdateTool(config.DB)
	if err != nil {
		return nil, fmt.Errorf("failed to create task_update tool: %w", err)
	}

	agentConfig := llmagent.Config{
		Name:        "executor",
		Description: "Specialized agent for executing tasks using tools like image generation, email queries, web search, and web fetching",
		Model:       config.Model,
		Tools: []tool.Tool{
			// 功能工具
			currentTimeTool, // Get current date/time - use before web_search for time-sensitive queries
			imageGenTool,
			emailQueryTool,
			webSearchTool,
			webFetchTool,
			// 任务管理工具（用于更新执行状态）
			taskListTool,
			taskGetTool,
			taskUpdateTool,
		},
		Instruction: executorAgentInstruction,
	}
	agent, err := llmagent.New(agentConfig)

	if err != nil {
		slog.Error("failed to create executor agent", "err", err)
		return nil, fmt.Errorf("failed to create executor agent: %w", err)
	}

	slog.Info("executor agent created successfully")
	return agent, nil
}

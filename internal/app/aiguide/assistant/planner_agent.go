// Copyright 2025 AIGuides
// Planner Agent - specialized in task decomposition and planning

package assistant

import (
	_ "embed"
	"fmt"
	"log/slog"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/model"
	"google.golang.org/adk/tool"
	"gorm.io/gorm"

	"aiguide/internal/pkg/tools"
)

//go:embed planner_agent_prompt.md
var plannerAgentInstruction string

// PlannerAgentConfig configures the Planner Agent
type PlannerAgentConfig struct {
	Model model.LLM
	DB    *gorm.DB
}

// NewPlannerAgent creates a specialized planning agent
func NewPlannerAgent(config *PlannerAgentConfig) (agent.Agent, error) {
	if config == nil {
		slog.Error("config parameter is nil")
		return nil, fmt.Errorf("config cannot be nil")
	}
	// 创建任务管理工具
	taskCreateTool, err := tools.NewTaskCreateTool(config.DB)
	if err != nil {
		return nil, fmt.Errorf("failed to create task_create tool: %w", err)
	}

	taskUpdateTool, err := tools.NewTaskUpdateTool(config.DB)
	if err != nil {
		return nil, fmt.Errorf("failed to create task_update tool: %w", err)
	}

	taskListTool, err := tools.NewTaskListTool(config.DB)
	if err != nil {
		return nil, fmt.Errorf("failed to create task_list tool: %w", err)
	}

	taskGetTool, err := tools.NewTaskGetTool(config.DB)
	if err != nil {
		return nil, fmt.Errorf("failed to create task_get tool: %w", err)
	}

	finishPlanningTool, err := tools.NewFinishPlanningTool()
	if err != nil {
		return nil, fmt.Errorf("failed to create finish_planning tool: %w", err)
	}

	agentConfig := llmagent.Config{
		Name:        "planner",
		Description: "Specialized agent for breaking down complex tasks into structured plans with subtasks, dependencies, and priorities",
		Model:       config.Model,
		Tools: []tool.Tool{
			taskCreateTool,
			taskUpdateTool,
			taskListTool,
			taskGetTool,
			finishPlanningTool,
		},
		Instruction: plannerAgentInstruction,
	}
	agent, err := llmagent.New(agentConfig)

	if err != nil {
		slog.Error("failed to create planner agent", "err", err)
		return nil, fmt.Errorf("failed to create planner agent: %w", err)
	}

	slog.Info("planner agent created successfully")
	return agent, nil
}

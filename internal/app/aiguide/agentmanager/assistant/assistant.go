package assistant

import (
	"fmt"
	"log/slog"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/agent/workflowagents/sequentialagent"
	"google.golang.org/adk/model"
	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/geminitool"
)

const searchAgentInstruction = `你是一个专业的信息检索助手。使用 GoogleSearch 工具查找信息，并以简洁、直接的方式回答。

**核心要求：**
1. 使用搜索工具获取准确信息
2. 回答简洁明了，直击要点
3. 只提供关键信息，避免冗长解释
4. 附上重要来源链接

**风格：**
- 简洁：避免啰嗦，每个要点不超过 2-3 句话
- 直接：直接回答问题，不要过度铺垫
- 结构化：使用简短的分点列表
- 务实：只提供用户需要的核心信息
`

func NewAssistantAgent(model model.LLM) (agent.Agent, error) {
	searchAgent, err := NewSearchAgent(model)
	if err != nil {
		return nil, fmt.Errorf("NewSearchAgent() error, err = %w", err)
	}

	cfg := sequentialagent.Config{
		AgentConfig: agent.Config{
			Name:        "AI assistant",
			Description: "一个 AI 助手，专门用于信息检索和事实核查",
			SubAgents:   []agent.Agent{searchAgent},
		},
	}
	assistent, err := sequentialagent.New(cfg)
	if err != nil {
		slog.Error("sequentialagent.New() error", "err", err)
		return nil, fmt.Errorf("sequentialagent.New() error, err = %w", err)
	}

	return assistent, nil
}

func NewSearchAgent(model model.LLM) (agent.Agent, error) {
	searchAgentConfig := llmagent.Config{
		Name:        "SearchAgent",
		Model:       model,
		Description: "专业的信息检索助手，擅长通过搜索获取准确、全面的信息并提供详细解答",
		Instruction: searchAgentInstruction,
		OutputKey:   "search_agent_output",
		Tools: []tool.Tool{
			geminitool.GoogleSearch{},
		},
	}
	agent, err := llmagent.New(searchAgentConfig)
	if err != nil {
		slog.Error("llmagent.New() error", "err", err)
		return nil, fmt.Errorf("llmagent.New() error, err = %w", err)
	}

	return agent, nil
}

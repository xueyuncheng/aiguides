package aiguide

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

func NewSequentialAgent(model model.LLM) (agent.Agent, error) {
	searchAgent, err := NewSearchAgent(model)
	if err != nil {
		return nil, fmt.Errorf("NewSearchAgent() error, err = %w", err)
	}

	factCheckAgent, err := NewFactCheckAgent(model)
	if err != nil {
		return nil, fmt.Errorf("NewFactCheckAgent() error, err = %w", err)
	}

	cfg := sequentialagent.Config{
		AgentConfig: agent.Config{
			Name:        "AI assistant",
			Description: "一个 AI 助手，会一次调用多个子 Agent 来完成任务",
			SubAgents:   []agent.Agent{searchAgent, factCheckAgent},
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
		Name:  "SearchAgent",
		Model: model,
		Instruction: `你是一个 AI 助手，回答用户的问题。

关键要求：
1. 使用 GoogleSearch 工具来搜索查询用户的问题；
2. 回答问题时，尽可能给出解决思路；
3. 如果可以的话，请提供相关链接以供参考；
`,
		Description: "这是一个 AI 助手",
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

func NewFactCheckAgent(model model.LLM) (agent.Agent, error) {
	factCheckAgent := llmagent.Config{
		Name:  "FactCheckAgent",
		Model: model,
		Instruction: `你是一个事实核查机器人。请检查 SearchAgent 提供的信息是否准确。

**SearchAgent 的输出：**
{search_agent_output}

**你的任务：**
- 使用 GoogleSearch 来验证关键信息
- 如果发现不准确的信息，提出修正建议
- 给出你的核查结论和最终的准确答案`,
		Description: "核查问题中的信息是否准确",
		OutputKey:   "fact_check_result",
		Tools: []tool.Tool{
			geminitool.GoogleSearch{},
		},
	}
	agent, err := llmagent.New(factCheckAgent)
	if err != nil {
		slog.Error("llmagent.New() error", "err", err)
		return nil, fmt.Errorf("llmagent.New() error, err = %w", err)
	}

	return agent, nil
}

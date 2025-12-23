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
			Name:        "SequentialAgent",
			Description: "调用搜索和总结子 Agent 以处理用户查询",
			SubAgents:   []agent.Agent{searchAgent, factCheckAgent},
		},
	}
	agent, err := sequentialagent.New(cfg)
	if err != nil {
		slog.Error("sequentialagent.New() error", "err", err)
		return nil, fmt.Errorf("sequentialagent.New() error, err = %w", err)
	}

	return agent, nil
}

func NewSearchAgent(model model.LLM) (agent.Agent, error) {
	searchAgentConfig := llmagent.Config{
		Name:  "SearchAgent",
		Model: model,
		Instruction: `你是一个搜索机器人。你的任务是通过搜索公开演讲和文章来介绍某个人的观点。
使用 Google 搜索工具来查找这些信息。

关键要求：
1. 仅提供搜索结果中明确找到的链接和信息。
2. 严禁幻觉或猜测任何 URL，尤其是 YouTube 链接。如果搜索结果中没有该链接，请不要包含它。
3. 对于每个来源，提供标题、简要描述和准确的 URL。
4. 优先考虑链接的准确性和相关性，而不是结果的数量。
`,
		Description: "该 Agent 可以根据用户查询在网络上搜索一个人的所有演讲和文章。",
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

func NewSummaryAgent(model model.LLM) (agent.Agent, error) {
	summaryAgent := llmagent.Config{
		Name:        "SummaryAgent",
		Model:       model,
		Instruction: `你是一个总结机器人。你会根据用户提供的内容生成简洁明了的总结。请确保你的总结涵盖所有重要信息，并且易于理解。`,
		Description: "该 Agent 可以根据用户提供的内容生成简洁明了的总结。",
		OutputKey:   "summary_agent_output",
	}
	agent, err := llmagent.New(summaryAgent)
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
		Instruction: `你是一个事实核查机器人。你会核实用户提供的内容中的事实是否准确。请确保你的核实涵盖所有重要信息，并且易于理解。
如果内容中包含链接，请务必检查链接是否真实存在，以及内容是否与用户提供的信息一致。
`,
		Description: "该 Agent 可以核实用户提供的内容中的事实是否准确。",
		OutputKey:   "fact_check_agent_output",
	}
	agent, err := llmagent.New(factCheckAgent)
	if err != nil {
		slog.Error("llmagent.New() error", "err", err)
		return nil, fmt.Errorf("llmagent.New() error, err = %w", err)
	}

	return agent, nil
}

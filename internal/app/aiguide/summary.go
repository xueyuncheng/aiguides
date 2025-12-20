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

	summaryAgent, err := NewSummaryAgent(model)
	if err != nil {
		return nil, fmt.Errorf("NewSummaryAgent() error, err = %w", err)
	}

	cfg := sequentialagent.Config{
		AgentConfig: agent.Config{
			Name:        "SequentialAgent",
			Description: "调用搜索和总结子 agent 以处理用户查询",
			SubAgents:   []agent.Agent{searchAgent, summaryAgent},
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
		Instruction: `你是一个搜索机器人，用户会让你介绍某个人的观点。
你需要去网络上搜索这个人的所有公开演讲和文章。
根据用户提供的名字，你需要在网络上搜索相关的信息，并返回给用户。
`,
		Description: "这个 agent 可以根据用户查询的内容去网络上查找这个人所有的演讲和文章",
		OutputKey:   "search_agent_output",
		Tools:       []tool.Tool{geminitool.GoogleSearch{}},
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
		Instruction: `你是一个总结机器人，你会根据用户提供的内容，生成一个简洁明了的总结。请确保你的总结涵盖所有重要信息，并且易于理解。`,
		Description: "这个 agent 可以根据用户提供的内容，生成一个简洁明了的总结",
		OutputKey:   "summary_agent_output",
	}
	agent, err := llmagent.New(summaryAgent)
	if err != nil {
		slog.Error("llmagent.New() error", "err", err)
		return nil, fmt.Errorf("llmagent.New() error, err = %w", err)
	}

	return agent, nil
}

package aiguide

import (
	"fmt"
	"log/slog"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/agent/workflowagents/sequentialagent"
	"google.golang.org/adk/model"
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
		Name:        "SearchAgent",
		Model:       model,
		Instruction: `你是一个搜索机器人，你会根据用户的查询内容，去网络上搜索相关的信息，并将搜索结果返回给用户。请确保你的回答简洁明了，直接切中要点。`,
		Description: "",
		OutputKey:   "search_agent_output",
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
		Description: "",
		OutputKey:   "summary_agent_output",
	}
	agent, err := llmagent.New(summaryAgent)
	if err != nil {
		slog.Error("llmagent.New() error", "err", err)
		return nil, fmt.Errorf("llmagent.New() error, err = %w", err)
	}

	return agent, nil
}

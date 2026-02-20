package assistant

import (
	"aiguide/internal/pkg/constant"
	"fmt"
	"log/slog"

	"google.golang.org/adk/runner"
)

func (a *Assistant) createRunner() (*runner.Runner, error) {
	// 创建 Root Agent 及其 SubAgents
	assistantConfig := &AssistantAgentConfig{
		Model:             a.model,
		GenaiClient:       a.genaiClient,
		DB:                a.db,
		MockImageGen:      a.mockImageGeneration,
		MockEmailIMAPConn: false,
		WebSearchConfig:   a.webSearchConfig,
		ExaConfig:         a.exaConfig,
	}
	assistantAgent, err := NewAssistantAgent(assistantConfig)
	if err != nil {
		return nil, fmt.Errorf("NewAssistantAgent() error, err = %w", err)
	}

	runnerConfig := runner.Config{
		AppName:        constant.AppNameAssistant.String(),
		Agent:          assistantAgent,
		SessionService: a.session,
	}
	runner, err := runner.New(runnerConfig)
	if err != nil {
		slog.Error("runner.New() error", "err", err)
		return nil, fmt.Errorf("runner.New() error, err = %w", err)
	}

	return runner, nil
}

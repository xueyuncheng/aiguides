package assistant

import (
	"aiguide/internal/pkg/constant"
	"fmt"
	"log/slog"

	"google.golang.org/adk/runner"
)

func (a *Assistant) createRunner() (*runner.Runner, error) {
	// 创建信息检索和事实核查的 Agent
	searchAgent, err := NewSearchAgent(a.model, a.genaiClient, a.mockImageGeneration)
	if err != nil {
		return nil, fmt.Errorf("NewSearchAgent() error, err = %w", err)
	}

	runnerConfig := runner.Config{
		AppName:        constant.AppNameAssistant.String(),
		Agent:          searchAgent,
		SessionService: a.session,
	}
	searchRunner, err := runner.New(runnerConfig)
	if err != nil {
		slog.Error("runner.New() error", "err", err)
		return nil, fmt.Errorf("runner.New() error, err = %w", err)
	}

	return searchRunner, nil
}

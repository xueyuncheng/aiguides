package assistant

import (
	"aiguide/internal/pkg/constant"
	"fmt"
	"log/slog"

	"google.golang.org/adk/runner"
)

func (a *Assistant) createSearchRunner() (*runner.Runner, error) {
	// 创建信息检索和事实核查的 Agent
	searchAgent, err := NewSearchAgent(a.model, a.genaiClient, a.mockImageGeneration)
	if err != nil {
		return nil, fmt.Errorf("NewSearchAgent() error, err = %w", err)
	}

	searchRunnerConfig := runner.Config{
		AppName:        constant.AppNameSearch.String(),
		Agent:          searchAgent,
		SessionService: a.session,
	}
	searchRunner, err := runner.New(searchRunnerConfig)
	if err != nil {
		slog.Error("runner.New() error", "err", err)
		return nil, fmt.Errorf("runner.New() error, err = %w", err)
	}

	return searchRunner, nil
}

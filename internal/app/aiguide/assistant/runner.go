package assistant

import (
	"aiguide/internal/pkg/constant"
	"fmt"
	"log/slog"

	"google.golang.org/adk/runner"
)

func (a *Assistant) createRunner() (*runner.Runner, error) {
	// 创建 Root Agent 及其执行子代理
	assistantConfig := &AssistantAgentConfig{
		Model:             a.model,
		GenaiClient:       a.genaiClient,
		DB:                a.db,
		MockImageGen:      a.mockImageGeneration,
		MockEmailIMAPConn: false,
		WebSearchConfig:   a.webSearchConfig,
		ExaConfig:         a.exaConfig,
		FileStore:         a.fileStore,
		PDFWorkDir:        a.pdfWorkDir,
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

// createExecutorRunner creates a runner that uses the executor agent directly as
// its root. This is used by the scheduler so that scheduled tasks bypass the
// orchestrator and go straight to the agent that has all the execution tools
// (send_email, web_search, etc.).
func (a *Assistant) createExecutorRunner() (*runner.Runner, error) {
	executorConfig := &ExecutorAgentConfig{
		Model:           a.model,
		GenaiClient:     a.genaiClient,
		DB:              a.db,
		MockImageGen:    a.mockImageGeneration,
		WebSearchConfig: a.webSearchConfig,
		ExaConfig:       a.exaConfig,
		FileStore:       a.fileStore,
		PDFWorkDir:      a.pdfWorkDir,
	}
	executorAgent, err := NewExecutorAgent(executorConfig)
	if err != nil {
		return nil, fmt.Errorf("NewExecutorAgent() error, err = %w", err)
	}

	runnerConfig := runner.Config{
		AppName:        constant.AppNameScheduler.String(),
		Agent:          executorAgent,
		SessionService: a.session,
	}
	r, err := runner.New(runnerConfig)
	if err != nil {
		slog.Error("runner.New() error for executor runner", "err", err)
		return nil, fmt.Errorf("runner.New() error, err = %w", err)
	}

	return r, nil
}

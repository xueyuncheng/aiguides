package assistant

import (
	"aiguide/internal/pkg/constant"
	"fmt"
	"log/slog"

	"google.golang.org/adk/runner"
)

// baseAgentConfig builds an AssistantAgentConfig from the assistant's fields.
// Both createRunner and createExecutorRunner use this to avoid repeating the
// same field assignments.
func (a *Assistant) baseAgentConfig() *AssistantAgentConfig {
	return &AssistantAgentConfig{
		Model:           a.model,
		GenaiClient:     a.genaiClient,
		DB:              a.db,
		MockImageGen:    a.mockImageGeneration,
		MockVideoGen:    a.mockVideoGeneration,
		WebSearchConfig: a.webSearchConfig,
		ExaConfig:       a.exaConfig,
		FileStore:       a.fileStore,
		PDFWorkDir:      a.pdfWorkDir,
		OAuthConfig:     a.oauthConfig,
		HTTPClient:      a.httpClient,
	}
}

func (a *Assistant) createRunner() (*runner.Runner, error) {
	cfg := a.baseAgentConfig()
	cfg.MockEmailIMAPConn = false

	assistantAgent, err := NewAssistantAgent(cfg)
	if err != nil {
		return nil, fmt.Errorf("NewAssistantAgent() error, err = %w", err)
	}

	runnerConfig := runner.Config{
		AppName:        constant.AppNameAssistant.String(),
		Agent:          assistantAgent,
		SessionService: a.session,
	}
	r, err := runner.New(runnerConfig)
	if err != nil {
		slog.Error("runner.New() error", "err", err)
		return nil, fmt.Errorf("runner.New() error, err = %w", err)
	}

	return r, nil
}

// createExecutorRunner creates a runner used by the scheduler. It uses the same
// assistant agent so that scheduled tasks have access to all tools.
func (a *Assistant) createExecutorRunner() (*runner.Runner, error) {
	assistantAgent, err := NewAssistantAgent(a.baseAgentConfig())
	if err != nil {
		return nil, fmt.Errorf("NewAssistantAgent() error, err = %w", err)
	}

	runnerConfig := runner.Config{
		AppName:        constant.AppNameScheduler.String(),
		Agent:          assistantAgent,
		SessionService: a.session,
	}
	r, err := runner.New(runnerConfig)
	if err != nil {
		slog.Error("runner.New() error for executor runner", "err", err)
		return nil, fmt.Errorf("runner.New() error, err = %w", err)
	}

	return r, nil
}

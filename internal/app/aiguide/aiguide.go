package aiguide

import (
	"context"
	"fmt"
	"log/slog"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/cmd/launcher"
	"google.golang.org/adk/cmd/launcher/full"
	"google.golang.org/adk/model/gemini"
	"google.golang.org/genai"
)

type Config struct {
	APIKey string `yaml:"api_key"`
}

type AIGuide struct {
	Config *Config

	launcher       launcher.Launcher
	launcherConfig *launcher.Config
}

func New(ctx context.Context, config *Config) (*AIGuide, error) {
	guide := &AIGuide{
		Config: config,
	}

	modelName := "gemini-2.0-flash"
	genaiConfig := &genai.ClientConfig{
		APIKey: config.APIKey,
	}
	model, err := gemini.NewModel(ctx, modelName, genaiConfig)
	if err != nil {
		slog.Error("gemini.NewModel() error", "err", err)
		return nil, fmt.Errorf("gemini.NewModel() error, err = %w", err)
	}

	sequentialAgent, err := NewSequentialAgent(model)
	if err != nil {
		return nil, fmt.Errorf("NewSequentialAgent() error, err = %w", err)
	}

	agentLoader := agent.NewSingleLoader(sequentialAgent)

	launcherConfig := &launcher.Config{
		AgentLoader: agentLoader,
	}
	guide.launcherConfig = launcherConfig

	launcher := full.NewLauncher()
	guide.launcher = launcher

	return guide, nil
}

func (a *AIGuide) Start(ctx context.Context) error {
	args := []string{"web", "api", "webui"}
	if err := a.launcher.Execute(ctx, a.launcherConfig, args); err != nil {
		slog.Error("a.launcher.Execute() error", "err", err)
		return fmt.Errorf("a.launcher.Execute() error, err = %w", err)
	}

	return nil
}

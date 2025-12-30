package aiguide

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/cmd/launcher"
	"google.golang.org/adk/cmd/launcher/full"
	"google.golang.org/adk/model/gemini"
	"google.golang.org/genai"
)

type Config struct {
	APIKey    string `yaml:"api_key"`
	ModelName string `yaml:"model_name"`
	Proxy     string `yaml:"proxy"`
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

	var httpClient *http.Client
	if config.Proxy != "" {
		parsedProxyURL, err := url.Parse(config.Proxy)
		if err != nil {
			slog.Error("url.Parse() error", "err", err)
			return nil, fmt.Errorf("url.Parse() error, err = %w", err)
		}
		parsedProxy := http.ProxyURL(parsedProxyURL)
		httpClient = &http.Client{
			Transport: &http.Transport{
				Proxy: parsedProxy,
			},
		}
	}

	genaiConfig := &genai.ClientConfig{
		APIKey:     config.APIKey,
		HTTPClient: httpClient,
	}
	model, err := gemini.NewModel(ctx, config.ModelName, genaiConfig)
	if err != nil {
		slog.Error("gemini.NewModel() error", "err", err)
		return nil, fmt.Errorf("gemini.NewModel() error, err = %w", err)
	}

	// 创建信息检索和事实核查的 Agent
	assistant, err := NewSequentialAgent(model)
	if err != nil {
		return nil, fmt.Errorf("NewSequentialAgent() error, err = %w", err)
	}

	// 创建网页总结 Agent
	webSummaryAgent, err := NewWebSummaryAgent(model)
	if err != nil {
		return nil, fmt.Errorf("NewWebSummaryAgent() error, err = %w", err)
	}

	// 创建旅游推荐 Agent
	travelAgent, err := NewTravelAgent(model)
	if err != nil {
		return nil, fmt.Errorf("NewTravelAgent() error, err = %w", err)
	}

	// 使用 MultiLoader 注册三个顶层 Agent
	agentLoader, err := agent.NewMultiLoader(assistant, webSummaryAgent, travelAgent)
	if err != nil {
		return nil, fmt.Errorf("agent.NewMultiLoader() error, err = %w", err)
	}

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

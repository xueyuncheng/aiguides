package aiguide

import (
	"aiguide/internal/app/aiguide/agentmanager"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/glebarez/sqlite"
	"google.golang.org/adk/model/gemini"
	"google.golang.org/genai"
)

type Config struct {
	APIKey    string `yaml:"api_key"`
	ModelName string `yaml:"model_name"`
	Proxy     string `yaml:"proxy"`
	UseGin    bool   `yaml:"use_gin"`
	GinPort   int    `yaml:"gin_port"`
}

type AIGuide struct {
	Config *Config

	agentManager *agentmanager.AgentManager
}

func New(ctx context.Context, config *Config) (*AIGuide, error) {
	httpClient, err := getHTTPClient(config.Proxy)
	if err != nil {
		return nil, fmt.Errorf("getHTTPClient() error, err = %w", err)
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

	dialector := sqlite.Open("file:aiguide_sessions.db")

	agentManager, err := agentmanager.New(model, dialector)
	if err != nil {
		return nil, fmt.Errorf("agentmanager.New() error, err = %w", err)
	}

	guide := &AIGuide{
		Config:       config,
		agentManager: agentManager,
	}

	return guide, nil
}

func getHTTPClient(proxy string) (*http.Client, error) {
	if proxy == "" {
		return http.DefaultClient, nil
	}

	parsedProxyURL, err := url.Parse(proxy)
	if err != nil {
		slog.Error("url.Parse() error", "err", err)
		return nil, fmt.Errorf("url.Parse() error, err = %w", err)
	}
	parsedProxy := http.ProxyURL(parsedProxyURL)
	httpClient := &http.Client{
		Transport: &http.Transport{
			Proxy: parsedProxy,
		},
	}
	return httpClient, nil
}

func (a *AIGuide) Run(ctx context.Context) error {
	// 这是一个阻塞操作
	if err := a.agentManager.Run(ctx); err != nil {
		return fmt.Errorf("a.agentManager.Run() error, err = %w", err)
	}

	return nil
}

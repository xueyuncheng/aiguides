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
	APIKey               string `yaml:"api_key"`
	ModelName            string `yaml:"model_name"`
	Proxy                string `yaml:"proxy"`
	UseGin               bool   `yaml:"use_gin"`
	GinPort              int    `yaml:"gin_port"`
	GoogleClientID       string `yaml:"google_client_id"`
	GoogleClientSecret   string `yaml:"google_client_secret"`
	GoogleRedirectURL    string `yaml:"google_redirect_url"`
	JWTSecret            string `yaml:"jwt_secret"`
	EnableAuthentication bool   `yaml:"enable_authentication"`
	FrontendURL          string `yaml:"frontend_url"`
}

// GetGoogleClientID returns the Google Client ID
func (c *Config) GetGoogleClientID() string {
	return c.GoogleClientID
}

// GetGoogleClientSecret returns the Google Client Secret
func (c *Config) GetGoogleClientSecret() string {
	return c.GoogleClientSecret
}

// GetGoogleRedirectURL returns the Google Redirect URL
func (c *Config) GetGoogleRedirectURL() string {
	return c.GoogleRedirectURL
}

// GetJWTSecret returns the JWT Secret
func (c *Config) GetJWTSecret() string {
	return c.JWTSecret
}

// GetEnableAuthentication returns whether authentication is enabled
func (c *Config) GetEnableAuthentication() bool {
	return c.EnableAuthentication
}

// GetFrontendURL returns the frontend URL
func (c *Config) GetFrontendURL() string {
	if c.FrontendURL == "" {
		return "http://localhost:3000"
	}
	return c.FrontendURL
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

	agentManager, err := agentmanager.New(model, dialector, config)
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

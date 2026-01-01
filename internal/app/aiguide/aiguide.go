package aiguide

import (
	"aiguide/internal/app/aiguide/agentmanager"
	"aiguide/internal/pkg/auth"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"google.golang.org/adk/model/gemini"
	"google.golang.org/genai"
)

type Config struct {
	APIKey             string `yaml:"api_key"`
	ModelName          string `yaml:"model_name"`
	Proxy              string `yaml:"proxy"`
	UseGin             bool   `yaml:"use_gin"`
	GinPort            int    `yaml:"gin_port"`
	GoogleClientID     string `yaml:"google_client_id"`
	GoogleClientSecret string `yaml:"google_client_secret"`
	GoogleRedirectURL  string `yaml:"google_redirect_url"`
	JWTSecret          string `yaml:"jwt_secret"`
	FrontendURL        string `yaml:"frontend_url"`
}

type AIGuide struct {
	config *Config

	agentManager *agentmanager.AgentManager
	authService  *auth.AuthService
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

	authConfig := &auth.Config{
		ClientID:     config.GoogleClientID,
		ClientSecret: config.GoogleClientSecret,
		RedirectURL:  config.GoogleRedirectURL,
		JWTSecret:    config.JWTSecret,
	}
	authService := auth.NewAuthService(authConfig)

	guide := &AIGuide{
		config:       config,
		agentManager: agentManager,
		authService:  authService,
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
	if err := a.agentManager.Run(ctx); err != nil {
		return fmt.Errorf("a.agentManager.Run() error, err = %w", err)
	}

	engine := gin.Default()
	if err := a.initRouter(engine); err != nil {
		return fmt.Errorf("a.initRouter() error, err = %w", err)
	}

	slog.Info("http listen", "port", "18080")
	if err := engine.Run(":18080"); err != nil {
		slog.Error("engine.Run() error", "err", err)
	}

	return nil
}

package aiguide

import (
	"aiguide/internal/app/aiguide/agentmanager"
	"aiguide/internal/app/aiguide/migration"
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
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Config struct {
	DBFile             string   `yaml:"db_file"`
	APIKey             string   `yaml:"api_key"`
	ModelName          string   `yaml:"model_name"`
	BaseURL            string   `yaml:"base_url"`
	Proxy              string   `yaml:"proxy"`
	UseGin             bool     `yaml:"use_gin"`
	GinPort            string   `yaml:"gin_port"`
	GoogleClientID     string   `yaml:"google_client_id"`
	GoogleClientSecret string   `yaml:"google_client_secret"`
	GoogleRedirectURL  string   `yaml:"google_redirect_url"`
	JWTSecret          string   `yaml:"jwt_secret"`
	FrontendURL        string   `yaml:"frontend_url"`
	AllowedEmails      []string `yaml:"allowed_emails"`
}

type AIGuide struct {
	config *Config

	migrator     *migration.Migrator
	db           *gorm.DB
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
	if config.BaseURL != "" {
		genaiConfig.HTTPOptions.BaseURL = config.BaseURL
	}
	model, err := gemini.NewModel(ctx, config.ModelName, genaiConfig)
	if err != nil {
		slog.Error("gemini.NewModel() error", "err", err)
		return nil, fmt.Errorf("gemini.NewModel() error, err = %w", err)
	}

	dialector := sqlite.Open(config.DBFile)
	dbConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Error),
	}

	db, err := gorm.Open(dialector, dbConfig)
	if err != nil {
		slog.Error("gorm.Open() error", "err", err)
		return nil, fmt.Errorf("gorm.Open() error, err = %w", err)
	}

	migrator := migration.New(db)

	agentManager, err := agentmanager.New(model, db)
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
		db:           db,
		migrator:     migrator,
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
	if err := a.migrator.Run(); err != nil {
		return fmt.Errorf("a.migrator.Run() error, err = %w", err)
	}

	if err := a.agentManager.Run(ctx); err != nil {
		return fmt.Errorf("a.agentManager.Run() error, err = %w", err)
	}

	engine := gin.Default()
	if err := a.initRouter(engine); err != nil {
		return fmt.Errorf("a.initRouter() error, err = %w", err)
	}

	slog.Info("http listen", "port", a.config.GinPort)
	if err := engine.Run(":" + a.config.GinPort); err != nil {
		slog.Error("engine.Run() error", "err", err)
	}

	return nil
}

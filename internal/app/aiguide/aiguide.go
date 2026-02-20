package aiguide

import (
	"aiguide/internal/app/aiguide/assistant"
	"aiguide/internal/app/aiguide/migration"
	"aiguide/internal/pkg/auth"
	"aiguide/internal/pkg/tools"
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
	"gorm.io/gorm/schema"
)

type Config struct {
	DBFile              string    `yaml:"db_file"`
	APIKey              string    `yaml:"api_key"`
	ModelName           string    `yaml:"model_name"`
	BaseURL             string    `yaml:"base_url"`
	Proxy               string    `yaml:"proxy"`
	UseGin              bool      `yaml:"use_gin"`
	GinPort             string    `yaml:"gin_port"`
	GoogleClientID      string    `yaml:"google_client_id"`
	GoogleClientSecret  string    `yaml:"google_client_secret"`
	GoogleRedirectURL   string    `yaml:"google_redirect_url"`
	JWTSecret           string    `yaml:"jwt_secret"`
	FrontendURL         string    `yaml:"frontend_url"`
	AllowedEmails       []string  `yaml:"allowed_emails"`
	SecureCookie        *bool     `yaml:"secure_cookie"` // 默认 true（生产环境），本地开发设置为 false
	MockImageGeneration bool      `yaml:"mock_image_generation"`
	WebSearch           WebSearch `yaml:"web_search"` // Web 搜索配置
	ExaSearch           ExaSearch `yaml:"exa_search"` // Exa 搜索配置
}

// WebSearch Web 搜索 YAML 配置（用于解析配置文件）
// 用户只需配置 instance_url，其他参数使用默认值
type WebSearch struct {
	InstanceURL string `yaml:"instance_url"`
}

// ExaSearch Exa 搜索 YAML 配置
type ExaSearch struct {
	APIKey string `yaml:"api_key"`
}

type AIGuide struct {
	config *Config

	migrator    *migration.Migrator
	db          *gorm.DB
	assistant   *assistant.Assistant
	authService *auth.AuthService
}

// secureCookie 返回 cookie 的 secure 标志值，默认为 true（生产环境）
func (a *AIGuide) secureCookie() bool {
	// 如果配置中未设置，默认返回 true（安全生产环境）
	// 只有在本地开发时才应该显式设置为 false
	if a.config.SecureCookie == nil {
		return true // 默认为安全模式
	}
	return *a.config.SecureCookie
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

	// 创建 genai client，用于图片生成等功能
	genaiClient, err := genai.NewClient(ctx, genaiConfig)
	if err != nil {
		slog.Error("genai.NewClient() error", "err", err)
		return nil, fmt.Errorf("genai.NewClient() error, err = %w", err)
	}

	model, err := gemini.NewModel(ctx, config.ModelName, genaiConfig)
	if err != nil {
		slog.Error("gemini.NewModel() error", "err", err)
		return nil, fmt.Errorf("gemini.NewModel() error, err = %w", err)
	}

	dialector := sqlite.Open(config.DBFile)
	dbConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Error),
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true,
		},
	}

	db, err := gorm.Open(dialector, dbConfig)
	if err != nil {
		slog.Error("gorm.Open() error", "err", err)
		return nil, fmt.Errorf("gorm.Open() error, err = %w", err)
	}

	migrator := migration.New(db)

	// 转换 WebSearch 配置为 tools.WebSearchConfig
	// 默认值（语言、超时等）已在 websearch.go 中硬编码
	webSearchConfig := tools.WebSearchConfig{
		SearXNG: tools.SearXNGConfig{
			InstanceURL: config.WebSearch.InstanceURL,
		},
	}

	assistant, err := assistant.New(model, db, genaiClient, config.MockImageGeneration, config.FrontendURL, webSearchConfig, tools.ExaConfig{APIKey: config.ExaSearch.APIKey})
	if err != nil {
		return nil, fmt.Errorf("assistant.New() error, err = %w", err)
	}

	authConfig := &auth.Config{
		ClientID:     config.GoogleClientID,
		ClientSecret: config.GoogleClientSecret,
		RedirectURL:  config.GoogleRedirectURL,
		JWTSecret:    config.JWTSecret,
	}
	authService := auth.NewAuthService(authConfig)

	guide := &AIGuide{
		config:      config,
		db:          db,
		migrator:    migrator,
		assistant:   assistant,
		authService: authService,
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

	if err := a.assistant.Run(ctx); err != nil {
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

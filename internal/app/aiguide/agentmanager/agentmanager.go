package agentmanager

import (
	"aiguide/internal/app/aiguide/agentmanager/assistant"
	"aiguide/internal/app/aiguide/agentmanager/emailsummary"
	"aiguide/internal/app/aiguide/agentmanager/travelagent"
	"aiguide/internal/app/aiguide/agentmanager/websummary"
	"aiguide/internal/pkg/auth"
	"aiguide/internal/pkg/constant"
	"context"
	"fmt"
	"log/slog"

	"github.com/gin-gonic/gin"
	"google.golang.org/adk/model"
	"google.golang.org/adk/runner"
	"google.golang.org/adk/session"
	"google.golang.org/adk/session/database"
	"gorm.io/gorm"
)

type Config struct {
	GoogleClientID       string
	GoogleClientSecret   string
	GoogleRedirectURL    string
	JWTSecret            string
	EnableAuthentication bool
	FrontendURL          string
}

type AgentManager struct {
	model   model.LLM
	session session.Service
	config  *Config

	runnerMap   map[constant.AppName]*runner.Runner
	authService *auth.AuthService
}

func New(model model.LLM, dialector gorm.Dialector, appConfig interface{}) (*AgentManager, error) {
	session, err := database.NewSessionService(dialector)
	if err != nil {
		slog.Error("database.NewSessionService() error", "err", err)
		return nil, fmt.Errorf("database.NewSessionService() error, err = %w", err)
	}

	if err := database.AutoMigrate(session); err != nil {
		slog.Error("database.AutoMigrate() error", "err", err)
		return nil, fmt.Errorf("database.AutoMigrate() error, err = %w", err)
	}

	// 转换配置
	var config *Config
	if cfg, ok := appConfig.(interface {
		GetGoogleClientID() string
		GetGoogleClientSecret() string
		GetGoogleRedirectURL() string
		GetJWTSecret() string
		GetEnableAuthentication() bool
		GetFrontendURL() string
	}); ok {
		config = &Config{
			GoogleClientID:       cfg.GetGoogleClientID(),
			GoogleClientSecret:   cfg.GetGoogleClientSecret(),
			GoogleRedirectURL:    cfg.GetGoogleRedirectURL(),
			JWTSecret:            cfg.GetJWTSecret(),
			EnableAuthentication: cfg.GetEnableAuthentication(),
			FrontendURL:          cfg.GetFrontendURL(),
		}
	} else {
		// 默认配置（不启用认证）
		config = &Config{
			EnableAuthentication: false,
			FrontendURL:          "http://localhost:3000",
		}
	}

	agentManager := &AgentManager{
		model:     model,
		session:   session,
		config:    config,
		runnerMap: make(map[constant.AppName]*runner.Runner, 16),
	}

	// 如果启用了认证，初始化认证服务
	if config.EnableAuthentication && config.GoogleClientID != "" {
		authConfig := &auth.Config{
			ClientID:     config.GoogleClientID,
			ClientSecret: config.GoogleClientSecret,
			RedirectURL:  config.GoogleRedirectURL,
			JWTSecret:    config.JWTSecret,
		}
		agentManager.authService = auth.NewAuthService(authConfig)
	}

	if err := agentManager.addTravelAgentRunner(); err != nil {
		return nil, fmt.Errorf("agentManager.addTravelAgentRunner() error, err = %w", err)
	}

	if err := agentManager.addWebSummaryRunner(); err != nil {
		return nil, fmt.Errorf("agentManager.addWebSummaryRunner() error, err = %w", err)
	}

	if err := agentManager.addAssistantRunner(); err != nil {
		return nil, fmt.Errorf("agentManager.addAssistantRunner() error, err = %w", err)
	}

	if err := agentManager.addEmailSummaryRunner(); err != nil {
		return nil, fmt.Errorf("agentManager.addEmailSummaryRunner() error, err = %w", err)
	}

	return agentManager, nil
}

func (a *AgentManager) Run(ctx context.Context) error {
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

func (a *AgentManager) addAssistantRunner() error {
	// 创建信息检索和事实核查的 Agent
	assistantAgent, err := assistant.NewAssistantAgent(a.model)
	if err != nil {
		return fmt.Errorf("NewAssistantAgent() error, err = %w", err)
	}

	assistantRunnerConfig := &runner.Config{
		AppName:        constant.AppNameAssistant.String(),
		Agent:          assistantAgent,
		SessionService: a.session,
	}
	assistantRunner, err := runner.New(*assistantRunnerConfig)
	if err != nil {
		slog.Error("runner.New() error", "err", err)
		return fmt.Errorf("runner.New() error, err = %w", err)
	}

	a.runnerMap[constant.AppNameAssistant] = assistantRunner
	return nil
}

func (a *AgentManager) addWebSummaryRunner() error {
	// 创建网页总结 Agent
	webSummaryAgent, err := websummary.NewWebSummaryAgent(a.model)
	if err != nil {
		return fmt.Errorf("NewWebSummaryAgent() error, err = %w", err)
	}

	webSummaryConfig := runner.Config{
		AppName:        constant.AppNameWebSummary.String(),
		Agent:          webSummaryAgent,
		SessionService: a.session,
	}
	webSummaryRunner, err := runner.New(webSummaryConfig)
	if err != nil {
		slog.Error("runner.New() error", "err", err)
		return fmt.Errorf("runner.New() error, err = %w", err)
	}

	a.runnerMap[constant.AppNameWebSummary] = webSummaryRunner
	return nil
}

func (a *AgentManager) addEmailSummaryRunner() error {
	// 创建邮件总结 Agent
	emailSummaryAgent, err := emailsummary.NewEmailSummaryAgent(a.model)
	if err != nil {
		return fmt.Errorf("NewEmailSummaryAgent() error, err = %w", err)
	}

	emailSummaryRunnerConfig := &runner.Config{
		AppName:        constant.AppNameEmailSummary.String(),
		Agent:          emailSummaryAgent,
		SessionService: a.session,
	}
	emailSummaryRunner, err := runner.New(*emailSummaryRunnerConfig)
	if err != nil {
		slog.Error("runner.New() error", "err", err)
		return fmt.Errorf("runner.New() error, err = %w", err)
	}

	a.runnerMap[constant.AppNameEmailSummary] = emailSummaryRunner
	return nil
}

func (a *AgentManager) addTravelAgentRunner() error {
	// 创建旅游推荐 Agent
	travelAgent, err := travelagent.NewTravelAgent(a.model)
	if err != nil {
		return fmt.Errorf("NewTravelAgent() error, err = %w", err)
	}

	travelAgentRunnerConfig := runner.Config{
		AppName:        constant.AppNameTravel.String(),
		Agent:          travelAgent,
		SessionService: a.session,
	}
	travelAgentRunner, err := runner.New(travelAgentRunnerConfig)
	if err != nil {
		slog.Error("runner.New() error", "err", err)
		return fmt.Errorf("runner.New() error, err = %w", err)
	}

	a.runnerMap[constant.AppNameTravel] = travelAgentRunner
	return nil
}

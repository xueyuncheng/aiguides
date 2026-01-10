package agentmanager

import (
	"aiguide/internal/app/aiguide/agentmanager/assistant"
	"aiguide/internal/app/aiguide/agentmanager/emailsummary"
	"aiguide/internal/app/aiguide/agentmanager/imagegen"
	"aiguide/internal/app/aiguide/agentmanager/travelagent"
	"aiguide/internal/app/aiguide/agentmanager/websummary"
	"aiguide/internal/pkg/auth"
	"aiguide/internal/pkg/constant"
	"context"
	"fmt"
	"log/slog"

	"google.golang.org/adk/model"
	"google.golang.org/adk/runner"
	"google.golang.org/adk/session"
	"google.golang.org/adk/session/database"
	"google.golang.org/genai"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

type AgentManager struct {
	mockImageGeneration bool
	model               model.LLM
	session             session.Service
	db                  *gorm.DB
	genaiClient         *genai.Client

	runnerMap   map[constant.AppName]*runner.Runner
	authService *auth.AuthService
}

func New(model model.LLM, db *gorm.DB, genaiClient *genai.Client, mockImageGeneration bool) (*AgentManager, error) {
	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true,
		},
	}
	session, err := database.NewSessionService(db.Dialector, gormConfig)
	if err != nil {
		slog.Error("database.NewSessionService() error", "err", err)
		return nil, fmt.Errorf("database.NewSessionService() error, err = %w", err)
	}

	if err := database.AutoMigrate(session); err != nil {
		slog.Error("database.AutoMigrate() error", "err", err)
		return nil, fmt.Errorf("database.AutoMigrate() error, err = %w", err)
	}

	agentManager := &AgentManager{
		mockImageGeneration: mockImageGeneration,
		model:               model,
		session:             session,
		db:                  db,
		genaiClient:         genaiClient,
		runnerMap:           make(map[constant.AppName]*runner.Runner, 16),
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

	if err := agentManager.addImageGenRunner(); err != nil {
		return nil, fmt.Errorf("agentManager.addImageGenRunner() error, err = %w", err)
	}

	return agentManager, nil
}

func (a *AgentManager) Run(ctx context.Context) error {
	return nil
}

func (a *AgentManager) addAssistantRunner() error {
	// 创建信息检索和事实核查的 Agent
	assistantAgent, err := assistant.NewAssistantAgent(a.model, a.genaiClient, a.mockImageGeneration)
	if err != nil {
		return fmt.Errorf("NewAssistantAgent() error, err = %w", err)
	}

	assistantRunnerConfig := runner.Config{
		AppName:        constant.AppNameAssistant.String(),
		Agent:          assistantAgent,
		SessionService: a.session,
	}
	assistantRunner, err := runner.New(assistantRunnerConfig)
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

func (a *AgentManager) addImageGenRunner() error {
	// 创建图片生成 Agent，传入模拟模式标志
	imageGenAgent, err := imagegen.NewImageGenAgent(a.model, a.genaiClient)
	if err != nil {
		return fmt.Errorf("NewImageGenAgent() error, err = %w", err)
	}

	imageGenRunnerConfig := runner.Config{
		AppName:        constant.AppNameImageGen.String(),
		Agent:          imageGenAgent,
		SessionService: a.session,
	}
	imageGenRunner, err := runner.New(imageGenRunnerConfig)
	if err != nil {
		slog.Error("runner.New() error", "err", err)
		return fmt.Errorf("runner.New() error, err = %w", err)
	}

	a.runnerMap[constant.AppNameImageGen] = imageGenRunner
	return nil
}

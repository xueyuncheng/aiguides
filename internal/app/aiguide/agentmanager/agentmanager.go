package agentmanager

import (
	"aiguide/internal/app/aiguide/agentmanager/assistant"
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

	if err := agentManager.addAssistantRunner(); err != nil {
		return nil, fmt.Errorf("agentManager.addAssistantRunner() error, err = %w", err)
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

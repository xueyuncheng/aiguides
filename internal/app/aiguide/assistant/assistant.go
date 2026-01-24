package assistant

import (
	"aiguide/internal/pkg/auth"
	"aiguide/internal/pkg/tools"
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

type Assistant struct {
	mockImageGeneration bool
	model               model.LLM
	session             session.Service
	db                  *gorm.DB
	genaiClient         *genai.Client
	frontendURL         string
	webSearchConfig     tools.WebSearchConfig

	runner      *runner.Runner
	authService *auth.AuthService
}

func New(model model.LLM, db *gorm.DB, genaiClient *genai.Client, mockImageGeneration bool, frontendURL string, webSearchConfig tools.WebSearchConfig) (*Assistant, error) {
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

	assistant := &Assistant{
		mockImageGeneration: mockImageGeneration,
		model:               model,
		session:             session,
		db:                  db,
		genaiClient:         genaiClient,
		frontendURL:         frontendURL,
		webSearchConfig:     webSearchConfig,
	}

	runner, err := assistant.createRunner()
	if err != nil {
		return nil, fmt.Errorf("assistant.createRunner() error, err = %w", err)
	}
	assistant.runner = runner

	return assistant, nil
}

func (a *Assistant) Run(ctx context.Context) error {
	return nil
}

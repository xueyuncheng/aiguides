package assistant

import (
	"aiguide/internal/pkg/auth"
	"aiguide/internal/pkg/storage"
	"aiguide/internal/pkg/tools"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync"

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
	mockVideoGeneration bool
	model               model.LLM
	session             session.Service
	db                  *gorm.DB
	genaiClient         *genai.Client
	frontendURL         string
	webSearchConfig     tools.WebSearchConfig
	exaConfig           tools.ExaConfig
	fileStore           storage.FileStore
	pdfWorkDir          string

	runner         *runner.Runner
	executorRunner *runner.Runner
	scheduler      *Scheduler

	authService *auth.AuthService

	liveClient     *genai.Client
	liveClientOnce sync.Once
	liveClientErr  error
	liveModel      string
	apiKey         string
	baseURL        string
	httpClient     *http.Client
}

type Config struct {
	Model               model.LLM
	DB                  *gorm.DB
	GenaiClient         *genai.Client
	MockImageGeneration bool
	MockVideoGeneration bool
	FrontendURL         string
	WebSearchConfig     tools.WebSearchConfig
	ExaConfig           tools.ExaConfig
	FileStore           storage.FileStore
	PDFWorkDir          string
	APIKey              string
	BaseURL             string
	HTTPClient          *http.Client
	LiveModel           string
}

func New(config *Config) (*Assistant, error) {
	if config == nil {
		slog.Error("config parameter is nil")
		return nil, fmt.Errorf("config cannot be nil")
	}
	if config.DB == nil {
		slog.Error("config.DB is nil")
		return nil, fmt.Errorf("config.DB cannot be nil")
	}
	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true,
		},
	}
	session, err := database.NewSessionService(config.DB.Dialector, gormConfig)
	if err != nil {
		slog.Error("database.NewSessionService() error", "err", err)
		return nil, fmt.Errorf("database.NewSessionService() error, err = %w", err)
	}

	if err := database.AutoMigrate(session); err != nil {
		slog.Error("database.AutoMigrate() error", "err", err)
		return nil, fmt.Errorf("database.AutoMigrate() error, err = %w", err)
	}

	assistant := &Assistant{
		mockImageGeneration: config.MockImageGeneration,
		mockVideoGeneration: config.MockVideoGeneration,
		model:               config.Model,
		session:             session,
		db:                  config.DB,
		genaiClient:         config.GenaiClient,
		frontendURL:         config.FrontendURL,
		webSearchConfig:     config.WebSearchConfig,
		exaConfig:           config.ExaConfig,
		fileStore:           config.FileStore,
		pdfWorkDir:          config.PDFWorkDir,
		liveModel:           config.LiveModel,
		apiKey:              config.APIKey,
		baseURL:             config.BaseURL,
		httpClient:          config.HTTPClient,
	}

	runner, err := assistant.createRunner()
	if err != nil {
		return nil, fmt.Errorf("assistant.createRunner() error, err = %w", err)
	}
	assistant.runner = runner

	executorRunner, err := assistant.createExecutorRunner()
	if err != nil {
		return nil, fmt.Errorf("assistant.createExecutorRunner() error, err = %w", err)
	}
	assistant.executorRunner = executorRunner
	assistant.scheduler = newScheduler(config.DB, executorRunner, session)

	return assistant, nil
}

func (a *Assistant) Run(ctx context.Context) error {
	a.scheduler.Start(ctx)
	return nil
}

func (a *Assistant) getLiveClient(ctx context.Context) (*genai.Client, error) {
	a.liveClientOnce.Do(func() {
		a.liveClient, a.liveClientErr = newLiveClient(ctx, a.apiKey, a.baseURL, a.httpClient)
	})
	return a.liveClient, a.liveClientErr
}

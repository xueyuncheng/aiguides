// Copyright 2025 AIGuides
// Executor Agent - specialized in executing tasks using available tools

package assistant

import (
	_ "embed"
	"fmt"
	"log/slog"

	"aiguide/internal/pkg/storage"
	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/model"
	"google.golang.org/adk/tool"
	"google.golang.org/genai"
	"gorm.io/gorm"

	"aiguide/internal/pkg/tools"
)

//go:embed executor_agent_prompt.md
var executorAgentInstruction string

// ExecutorAgentConfig configures the Executor Agent
type ExecutorAgentConfig struct {
	Model           model.LLM
	GenaiClient     *genai.Client
	DB              *gorm.DB
	MockImageGen    bool
	WebSearchConfig tools.WebSearchConfig
	ExaConfig       tools.ExaConfig
	FileStore       storage.FileStore
	PDFWorkDir      string
}

// NewExecutorAgent creates a specialized execution agent with all functional tools
func NewExecutorAgent(config *ExecutorAgentConfig) (agent.Agent, error) {
	if config == nil {
		slog.Error("config parameter is nil")
		return nil, fmt.Errorf("config cannot be nil")
	}
	// 创建功能工具
	imageGenTool, err := tools.NewImageGenTool(config.GenaiClient, config.MockImageGen)
	if err != nil {
		return nil, fmt.Errorf("failed to create image gen tool: %w", err)
	}

	emailQueryTool, err := tools.NewEmailQueryTool()
	if err != nil {
		return nil, fmt.Errorf("failed to create email query tool: %w", err)
	}

	sendEmailTool, err := tools.NewSendEmailTool()
	if err != nil {
		return nil, fmt.Errorf("failed to create send email tool: %w", err)
	}

	webSearchTool, err := tools.NewWebSearchTool(config.WebSearchConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create web search tool: %w", err)
	}

	webFetchTool, err := tools.NewWebFetchTool()
	if err != nil {
		return nil, fmt.Errorf("failed to create web fetch tool: %w", err)
	}

	exaSearchTool, err := tools.NewExaSearchTool(config.ExaConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create exa search tool: %w", err)
	}

	currentTimeTool, err := tools.NewCurrentTimeTool()
	if err != nil {
		return nil, fmt.Errorf("failed to create current time tool: %w", err)
	}

	memoryTool, err := tools.NewMemoryTool(config.DB)
	if err != nil {
		return nil, fmt.Errorf("failed to create manage_memory tool: %w", err)
	}

	pdfExtractTextTool, err := tools.NewPDFExtractTextTool(config.DB, config.FileStore, config.PDFWorkDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create pdf_extract_text tool: %w", err)
	}

	pdfGenerateDocumentTool, err := tools.NewPDFGenerateDocumentTool(config.DB, config.FileStore, config.PDFWorkDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create pdf_generate_document tool: %w", err)
	}

	audioTranscribeTool, err := tools.NewAudioTranscribeTool(config.DB, config.FileStore, config.GenaiClient, config.PDFWorkDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create audio_transcribe tool: %w", err)
	}

	fileListTool, err := tools.NewFileListTool(config.DB)
	if err != nil {
		return nil, fmt.Errorf("failed to create file_list tool: %w", err)
	}

	fileGetTool, err := tools.NewFileGetTool(config.DB)
	if err != nil {
		return nil, fmt.Errorf("failed to create file_get tool: %w", err)
	}

	fileDownloadTool, err := tools.NewFileDownloadTool(config.DB, config.FileStore)
	if err != nil {
		return nil, fmt.Errorf("failed to create file_download tool: %w", err)
	}

	// 任务查询和更新工具
	taskListTool, err := tools.NewTaskListTool(config.DB)
	if err != nil {
		return nil, fmt.Errorf("failed to create task_list tool: %w", err)
	}

	taskGetTool, err := tools.NewTaskGetTool(config.DB)
	if err != nil {
		return nil, fmt.Errorf("failed to create task_get tool: %w", err)
	}

	taskUpdateTool, err := tools.NewTaskUpdateTool(config.DB)
	if err != nil {
		return nil, fmt.Errorf("failed to create task_update tool: %w", err)
	}

	agentConfig := llmagent.Config{
		Name:        "executor",
		Description: "Specialized agent for executing tasks using tools like image generation, email queries, email sending, web search, and web fetching",
		Model:       config.Model,
		GenerateContentConfig: &genai.GenerateContentConfig{
			ThinkingConfig: &genai.ThinkingConfig{
				IncludeThoughts: true,
			},
		},
		Tools: []tool.Tool{
			// 功能工具
			currentTimeTool, // Get current date/time - use before web_search for time-sensitive queries
			memoryTool,
			imageGenTool,
			emailQueryTool,
			sendEmailTool,
			webSearchTool,
			exaSearchTool,
			webFetchTool,
			fileDownloadTool,
			fileListTool,
			fileGetTool,
			pdfExtractTextTool,
			pdfGenerateDocumentTool,
			audioTranscribeTool,
			// 任务管理工具（用于更新执行状态）
			taskListTool,
			taskGetTool,
			taskUpdateTool,
		},
		Instruction: executorAgentInstruction,
	}
	agent, err := llmagent.New(agentConfig)

	if err != nil {
		slog.Error("failed to create executor agent", "err", err)
		return nil, fmt.Errorf("failed to create executor agent: %w", err)
	}

	slog.Info("executor agent created successfully")
	return agent, nil
}

package assistant

import (
	_ "embed"
	"fmt"
	"log/slog"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/tool"
	"google.golang.org/genai"

	"aiguide/internal/pkg/tools"
)

//go:embed assistant_agent_prompt.md
var assistantAgentInstruction string

// NewAssistantAgent creates the root assistant agent with domain-specific sub-agents.
func NewAssistantAgent(config *Config) (agent.Agent, error) {
	if config == nil {
		slog.Error("config parameter is nil")
		return nil, fmt.Errorf("config cannot be nil")
	}

	allTools, err := createAssistantTools(config)
	if err != nil {
		return nil, err
	}

	partition := partitionTools(allTools)

	subAgents, err := buildSubAgents(partition, config.Model)
	if err != nil {
		return nil, fmt.Errorf("failed to build sub-agents: %w", err)
	}

	agentConfig := llmagent.Config{
		Name:        "assistant",
		Model:       config.Model,
		Description: "AI assistant that answers questions and delegates to specialized sub-agents",
		Instruction: assistantAgentInstruction,
		GenerateContentConfig: &genai.GenerateContentConfig{
			ThinkingConfig: &genai.ThinkingConfig{
				IncludeThoughts: true,
			},
		},
		Tools:     partition.Common,
		SubAgents: subAgents,
	}

	a, err := llmagent.New(agentConfig)
	if err != nil {
		slog.Error("failed to create assistant agent", "err", err)
		return nil, fmt.Errorf("failed to create assistant agent: %w", err)
	}

	slog.Info("assistant agent created with sub-agents", "sub_agent_count", len(subAgents))
	return a, nil
}

func createAssistantTools(config *Config) ([]tool.Tool, error) {
	// Context
	currentTimeTool, err := tools.NewCurrentTimeTool()
	if err != nil {
		return nil, fmt.Errorf("failed to create current time tool: %w", err)
	}

	memoryTool, err := tools.NewMemoryTool(config.DB)
	if err != nil {
		return nil, fmt.Errorf("failed to create manage_memory tool: %w", err)
	}

	// Web
	webSearchTool, err := tools.NewWebSearchTool(config.WebSearchConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create web search tool: %w", err)
	}

	exaSearchTool, err := tools.NewExaSearchTool(config.ExaConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create exa search tool: %w", err)
	}

	webFetchTool, err := tools.NewWebFetchTool()
	if err != nil {
		return nil, fmt.Errorf("failed to create web fetch tool: %w", err)
	}

	// Email
	emailQueryTool, err := tools.NewEmailQueryTool()
	if err != nil {
		return nil, fmt.Errorf("failed to create email query tool: %w", err)
	}

	sendEmailTool, err := tools.NewSendEmailTool()
	if err != nil {
		return nil, fmt.Errorf("failed to create send email tool: %w", err)
	}

	// Files
	fileDownloadTool, err := tools.NewFileDownloadTool(config.DB, config.FileStore)
	if err != nil {
		return nil, fmt.Errorf("failed to create file_download tool: %w", err)
	}

	fileListTool, err := tools.NewFileListTool(config.DB)
	if err != nil {
		return nil, fmt.Errorf("failed to create file_list tool: %w", err)
	}

	fileGetTool, err := tools.NewFileGetTool(config.DB)
	if err != nil {
		return nil, fmt.Errorf("failed to create file_get tool: %w", err)
	}

	// Documents & media
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

	imageGenTool, err := tools.NewImageGenTool(config.GenaiClient, config.MockImageGen)
	if err != nil {
		return nil, fmt.Errorf("failed to create image gen tool: %w", err)
	}

	videoGenTool, err := tools.NewVideoGenTool(config.GenaiClient, config.DB, config.FileStore, config.MockVideoGen)
	if err != nil {
		return nil, fmt.Errorf("failed to create video gen tool: %w", err)
	}

	// Tasks
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

	scheduledTaskCreateTool, err := tools.NewScheduledTaskCreateTool(config.DB)
	if err != nil {
		return nil, fmt.Errorf("failed to create scheduled_task_create tool: %w", err)
	}

	scheduledTaskListTool, err := tools.NewScheduledTaskListTool(config.DB)
	if err != nil {
		return nil, fmt.Errorf("failed to create scheduled_task_list tool: %w", err)
	}

	// System
	sshListServersTool, err := tools.NewSSHListServersTool()
	if err != nil {
		return nil, fmt.Errorf("failed to create ssh_list_servers tool: %w", err)
	}

	sshExecuteTool, err := tools.NewSSHExecuteTool()
	if err != nil {
		return nil, fmt.Errorf("failed to create ssh_execute tool: %w", err)
	}

	toolList := []tool.Tool{
		// Context
		currentTimeTool,
		memoryTool,
		// Web
		webSearchTool,
		exaSearchTool,
		webFetchTool,
		// Email
		emailQueryTool,
		sendEmailTool,
		// Files
		fileDownloadTool,
		fileListTool,
		fileGetTool,
		// Documents & media
		pdfExtractTextTool,
		pdfGenerateDocumentTool,
		audioTranscribeTool,
		imageGenTool,
		videoGenTool,
		// Tasks
		taskListTool,
		taskGetTool,
		taskUpdateTool,
		scheduledTaskCreateTool,
		scheduledTaskListTool,
		// System
		sshListServersTool,
		sshExecuteTool,
	}

	// Calendar tool requires OAuth to be configured.
	if config.OAuthConfig != nil {
		calendarTool, err := tools.NewCalendarTool(config.DB, config.OAuthConfig, config.HTTPClient)
		if err != nil {
			return nil, fmt.Errorf("failed to create manage_calendar tool: %w", err)
		}
		toolList = append(toolList, calendarTool)
	}

	return toolList, nil
}

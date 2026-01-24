package assistant

import (
	"aiguide/internal/pkg/tools"
	_ "embed"
	"fmt"
	"log/slog"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/model"
	"google.golang.org/adk/tool"
	"google.golang.org/genai"
)

//go:embed assistant_agent_prompt.md
var assistantAgentInstruction string

func NewAssistantAgent(model model.LLM, genaiClient *genai.Client, mockImageGeneration bool, webSearchConfig tools.WebSearchConfig) (agent.Agent, error) {
	// 创建图片生成工具
	imageGenTool, err := tools.NewImageGenTool(genaiClient, mockImageGeneration)
	if err != nil {
		slog.Error("tools.NewImageGenTool() error", "err", err)
		return nil, fmt.Errorf("tools.NewImageGenTool() error, err = %w", err)
	}

	// 创建邮件查询工具
	emailQueryTool, err := tools.NewEmailQueryTool()
	if err != nil {
		slog.Error("tools.NewEmailQueryTool() error", "err", err)
		return nil, fmt.Errorf("tools.NewEmailQueryTool() error, err = %w", err)
	}

	// 创建网页搜索工具
	webSearchTool, err := tools.NewWebSearchTool(webSearchConfig)
	if err != nil {
		slog.Error("tools.NewWebSearchTool() error", "err", err)
		return nil, fmt.Errorf("tools.NewWebSearchTool() error, err = %w", err)
	}

	// 创建网页抓取工具
	webFetchTool, err := tools.NewWebFetchTool()
	if err != nil {
		slog.Error("tools.NewWebFetchTool() error", "err", err)
		return nil, fmt.Errorf("tools.NewWebFetchTool() error, err = %w", err)
	}

	searchAgentConfig := llmagent.Config{
		Name:        "root_agent",
		Model:       model,
		Description: "专业的信息检索助手，擅长通过搜索获取准确、全面的信息并提供详细解答，也可以生成图片和查询邮件",
		Instruction: assistantAgentInstruction,
		GenerateContentConfig: &genai.GenerateContentConfig{
			ThinkingConfig: &genai.ThinkingConfig{
				IncludeThoughts: true,
			},
		},
		Tools: []tool.Tool{
			imageGenTool,
			emailQueryTool,
			webSearchTool,
			webFetchTool,
		},
	}
	agent, err := llmagent.New(searchAgentConfig)
	if err != nil {
		slog.Error("llmagent.New() error", "err", err)
		return nil, fmt.Errorf("llmagent.New() error, err = %w", err)
	}

	return agent, nil
}

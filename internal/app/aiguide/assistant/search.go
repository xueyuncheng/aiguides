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
	"google.golang.org/adk/tool/geminitool"
	"google.golang.org/genai"
)

//go:embed search_prompt.md
var searchAgentInstruction string

func NewSearchAgent(model model.LLM, genaiClient *genai.Client, mockImageGeneration bool) (agent.Agent, error) {
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

	searchAgentConfig := llmagent.Config{
		Name:        "SearchAgent",
		Model:       model,
		Description: "专业的信息检索助手，擅长通过搜索获取准确、全面的信息并提供详细解答，也可以生成图片和查询邮件",
		Instruction: searchAgentInstruction,
		Tools: []tool.Tool{
			geminitool.GoogleSearch{},
			imageGenTool,
			emailQueryTool,
		},
	}
	agent, err := llmagent.New(searchAgentConfig)
	if err != nil {
		slog.Error("llmagent.New() error", "err", err)
		return nil, fmt.Errorf("llmagent.New() error, err = %w", err)
	}

	return agent, nil
}

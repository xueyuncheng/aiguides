package imagegen

import (
	"aiguide/internal/pkg/tools"
	"fmt"
	"log/slog"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/model"
	"google.golang.org/adk/tool"
	"google.golang.org/genai"
)

const imageGenAgentInstruction = `你是一个专业的 AI 图片生成助手。你可以根据用户的描述生成高质量的图片。

**核心要求：**
1. 理解用户的图片需求，提取关键元素
2. 使用 generate_image 工具生成图片
3. 如果用户描述不够详细，可以适当补充细节以生成更好的图片
4. 告知用户生成的图片数量和特点

**图片描述指南：**
- 详细描述图片的主体、风格、颜色、光线、构图等元素
- 可以指定艺术风格，如：写实风格、卡通风格、油画风格、水彩风格等
- 可以指定场景和氛围，如：白天/夜晚、室内/户外、明亮/阴暗等
- 支持的宽高比：1:1（正方形）、3:4（竖屏）、4:3（横屏）、9:16（手机竖屏）、16:9（宽屏）
- 一次可以生成 1-4 张图片

**示例对话：**
用户："生成一张猫咪的图片"
助手：好的，我来为您生成一张猫咪的图片。[调用工具] 已经为您生成了一张可爱的猫咪图片！

用户："生成一张日落时分的海滩风景照，要有温暖的色调"
助手：我来为您生成一张温暖色调的海滩日落风景照。[调用工具] 已经为您生成了一张美丽的海滩日落照片！

**注意事项：**
- 如果用户的描述过于简单，可以适当补充细节
- 生成后要简单描述图片的特点
- 如果生成失败，要友好地告知用户原因
- 调用 generate_image 工具时，不要在用户可见的回复中输出任何 JSON、action 或工具调用细节
`

// NewImageGenAgent 创建图片生成 Agent
// mockMode 参数用于在开发时使用模拟数据而不调用真实 API
func NewImageGenAgent(model model.LLM, genaiClient *genai.Client) (agent.Agent, error) {
	imageGenTool, err := tools.NewImageGenTool(genaiClient, true)
	if err != nil {
		slog.Error("tools.NewImageGenTool() error", "err", err)
		return nil, fmt.Errorf("tools.NewImageGenTool() error, err = %w", err)
	}

	config := llmagent.Config{
		Name:        "ImageGenAgent",
		Model:       model,
		Description: "专业的 AI 图片生成助手，可以根据文字描述生成各种风格的高质量图片",
		Instruction: imageGenAgentInstruction,
		Tools: []tool.Tool{
			imageGenTool,
		},
	}

	agent, err := llmagent.New(config)
	if err != nil {
		slog.Error("llmagent.New() error", "err", err)
		return nil, fmt.Errorf("llmagent.New() error, err = %w", err)
	}

	return agent, nil
}

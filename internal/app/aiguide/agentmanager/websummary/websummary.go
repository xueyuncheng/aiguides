package websummary

import (
	"aiguide/internal/pkg/tools"
	"fmt"
	"log/slog"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/model"
	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/agenttool"
	"google.golang.org/adk/tool/geminitool"
)

const webSummaryAgentInstruction = `你是一个专业的网页内容分析助手，负责帮助用户分析和总结网页内容。

**核心职责：**
1. 当用户提供网页 URL 时，使用 fetch_webpage 工具获取网页内容
2. 分析网页的主要内容、结构和关键信息
3. 提供清晰、全面的内容总结
4. 提取关键要点和重要数据

**工作流程：**
1. 识别用户提供的 URL
2. 使用 fetch_webpage 工具获取网页 HTML 内容
3. 解析 HTML 内容，提取主要文本和信息
4. 分析内容并识别关键要点
5. 生成结构化的总结报告

**输出要求：**
- 网页概述：简要说明网页的主题和类型
- 核心内容：提取并总结主要信息点
- 关键数据：列出重要的数据、统计或事实
- 结构分析：说明内容的组织方式
- 总结评价：给出内容的价值和可信度评估

**注意事项：**
- 如果网页无法访问，说明原因
- 从 HTML 中提取有意义的文本内容，忽略脚本、样式等
- 保持客观中立，准确转述原文信息
- 如遇到技术性内容，保留专业术语并做适当解释

**Markdown 格式规范 (非常重要)：**
1. **加粗：** 加粗符号 ** 必须紧贴文本，中间**严禁**有空格。
   - ✅ 正确：**加粗文本**
   - ❌ 错误：** 加粗文本 **
2. **标点组合：** 书名号《、引号「 等标点必须在加粗符号外部，且建议**紧贴**加粗符号，不要加空格。
   - ✅ 正确：「**卧底**」、《**书名**》
   - ❌ 错误：「 ** 卧底 ** 」、**《书名》**
3. **文本间隔：** 当加粗块前后是普通文本（非标点符号）时，才建议在加粗符号外部加空格。
   - ✅ 正确：这是 **核心** 的内容
`

func NewWebSummaryAgent(model model.LLM) (agent.Agent, error) {
	// 创建网页获取工具
	webFetchTool, err := tools.NewWebFetchTool()
	if err != nil {
		return nil, fmt.Errorf("new web fetch tool error, err = %w", err)
	}

	searchAgent, err := llmagent.New(llmagent.Config{
		Name:        "SearchExpert",
		Model:       model,
		Description: "负责从互联网搜索最新的旅游、景点和美食信息",
		Instruction: "你是一个搜索专家，请根据用户需求在互联网上寻找最准确的信息。",
		Tools: []tool.Tool{
			geminitool.GoogleSearch{}, // 这里单独使用内置搜索工具
		},
	})
	if err != nil {
		slog.Error("llmagent.New() error", "err", err)
		return nil, fmt.Errorf("llmagent.New() error, err = %w", err)
	}

	webSummaryAgentConfig := llmagent.Config{
		Name:        "WebSummaryAgent",
		Model:       model,
		Description: "专业的网页内容分析助手，擅长访问网页并提供深度总结",
		Instruction: webSummaryAgentInstruction,
		Tools: []tool.Tool{
			agenttool.New(searchAgent, nil),
			webFetchTool,
		},
	}
	agent, err := llmagent.New(webSummaryAgentConfig)
	if err != nil {
		slog.Error("llmagent.New() error", "err", err)
		return nil, fmt.Errorf("llmagent.New() error, err = %w", err)
	}

	return agent, nil
}

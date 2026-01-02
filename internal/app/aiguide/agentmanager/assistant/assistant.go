package assistant

import (
	"fmt"
	"log/slog"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/agent/workflowagents/sequentialagent"
	"google.golang.org/adk/model"
	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/geminitool"
)

const searchAgentInstruction = `你是一个专业的信息检索助手。使用 GoogleSearch 工具查找信息，并以简洁、直接的方式回答。

**核心要求：**
1. 使用搜索工具获取准确信息
2. 回答简洁明了，直击要点
3. 只提供关键信息，避免冗长解释
4. 附上重要来源链接

**风格：**
- 简洁：避免啰嗦，每个要点不超过 2-3 句话
- 直接：直接回答问题，不要过度铺垫
- 结构化：使用简短的分点列表
- 务实：只提供用户需要的核心信息

**Markdown 格式规范 (非常重要)：**
1. **加粗：** 加粗符号 ** 必须紧贴文本，中间**严禁**有空格。
   - ✅ 正确：**加粗文本**
   - ❌ 错误：** 加粗文本 **
2. **标点组合：** 书名号《、引号「 等标点必须在加粗符号外部，且建议**紧贴**加粗符号，不要加空格。
   - ✅ 正确：「**卧底**」、 《**书名**》
   - ❌ 错误：「 ** 卧底 ** 」、**《书名》**
3. **文本间隔：** 当加粗块前后是普通文本（非标点符号）时，才建议在加粗符号外部加空格。
   - ✅ 正确：这是 **核心** 的内容
4. **专有名词格式**：
   - 书籍、功法名称：使用 《**名称**》
   - 角色、称号：使用 「**名称**」
   - 技术术语：使用反引号 ` + "`术语`" + `
   - 人名、地名：直接使用中文，无需特殊标记
`

func NewAssistantAgent(model model.LLM) (agent.Agent, error) {
	searchAgent, err := NewSearchAgent(model)
	if err != nil {
		return nil, fmt.Errorf("NewSearchAgent() error, err = %w", err)
	}

	cfg := sequentialagent.Config{
		AgentConfig: agent.Config{
			Name:        "AI assistant",
			Description: "一个 AI 助手，专门用于信息检索和事实核查",
			SubAgents:   []agent.Agent{searchAgent},
		},
	}
	assistent, err := sequentialagent.New(cfg)
	if err != nil {
		slog.Error("sequentialagent.New() error", "err", err)
		return nil, fmt.Errorf("sequentialagent.New() error, err = %w", err)
	}

	return assistent, nil
}

func NewSearchAgent(model model.LLM) (agent.Agent, error) {
	searchAgentConfig := llmagent.Config{
		Name:        "SearchAgent",
		Model:       model,
		Description: "专业的信息检索助手，擅长通过搜索获取准确、全面的信息并提供详细解答",
		Instruction: searchAgentInstruction,
		Tools: []tool.Tool{
			geminitool.GoogleSearch{},
		},
	}
	agent, err := llmagent.New(searchAgentConfig)
	if err != nil {
		slog.Error("llmagent.New() error", "err", err)
		return nil, fmt.Errorf("llmagent.New() error, err = %w", err)
	}

	return agent, nil
}

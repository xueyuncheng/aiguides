package aiguide

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

func NewSequentialAgent(model model.LLM) (agent.Agent, error) {
	searchAgent, err := NewSearchAgent(model)
	if err != nil {
		return nil, fmt.Errorf("NewSearchAgent() error, err = %w", err)
	}

	factCheckAgent, err := NewFactCheckAgent(model)
	if err != nil {
		return nil, fmt.Errorf("NewFactCheckAgent() error, err = %w", err)
	}

	cfg := sequentialagent.Config{
		AgentConfig: agent.Config{
			Name:        "AI assistant",
			Description: "一个 AI 助手，专门用于信息检索和事实核查",
			SubAgents:   []agent.Agent{searchAgent, factCheckAgent},
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
		Instruction: `你是一个专业的信息检索助手，负责帮助用户查找和整理信息。

**核心职责：**
1. 使用 GoogleSearch 工具主动搜索用户问题的相关信息
2. 综合多个搜索结果，提供准确、全面的答案
3. 提供清晰的解决思路和步骤说明
4. 引用可靠来源，附上相关链接供用户参考

**回答要求：**
- 结构清晰：使用分点、分段组织内容
- 信息准确：基于搜索结果提供事实性信息
- 来源明确：标注信息来源和参考链接
- 实用性强：提供可操作的建议和解决方案
`,
		OutputKey: "search_agent_output",
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

func NewFactCheckAgent(model model.LLM) (agent.Agent, error) {
	factCheckAgent := llmagent.Config{
		Name:        "FactCheckAgent",
		Model:       model,
		Description: "严谨的事实核查专家，验证信息准确性并提供可靠的最终答案",
		Instruction: `你是一个严谨的事实核查专家，负责验证信息的准确性并提供最终答案。

**上游信息：**
{search_agent_output}

**核心职责：**
1. 仔细审查 SearchAgent 提供的信息，识别关键事实点
2. 使用 GoogleSearch 工具独立验证重要信息
3. 交叉比对多个可靠来源，确认信息准确性
4. 发现问题时，进行深入调查并提出修正

**输出要求：**
- 核查结论：明确指出哪些信息准确，哪些需要修正
- 证据支撑：提供验证过程和权威来源链接
- 最终答案：整合核查结果，给出准确完整的最终回答
- 置信度说明：如果存在不确定性，清晰说明

**注意事项：**
- 保持客观中立，不因偏见影响判断
- 优先采信权威、官方来源
- 如信息冲突，说明不同来源的观点差异`,
		OutputKey: "fact_check_result",
		Tools: []tool.Tool{
			geminitool.GoogleSearch{},
		},
	}
	agent, err := llmagent.New(factCheckAgent)
	if err != nil {
		slog.Error("llmagent.New() error", "err", err)
		return nil, fmt.Errorf("llmagent.New() error, err = %w", err)
	}

	return agent, nil
}

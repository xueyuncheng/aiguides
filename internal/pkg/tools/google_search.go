package tools

import (
	"fmt"

	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/model"
	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/agenttool"
	"google.golang.org/adk/tool/geminitool"
)

// NewGoogleSearchTool 创建 Google 搜索工具
// 该工具通过 agenttool 封装了 geminitool.GoogleSearch{}。
// 这样做的目的是为了解决在同一个 Agent 中混用 Gemini 原生工具（如 GoogleSearch）和自定义工具（如 functiontool）时可能出现的兼容性问题。
// 通过封装，GoogleSearch 对主 Agent 来说表现为一个普通的工具调用。
func NewGoogleSearchTool(model model.LLM) (tool.Tool, error) {
	// 创建一个专门负责搜索的子 Agent
	searchAgent, err := llmagent.New(llmagent.Config{
		Name:        "google_search",
		Description: "使用 Google 搜索获取互联网上的实时信息、事实、新闻等。当你需要准确、最新的信息来回答问题时，请使用此工具。",
		Model:       model,
		Instruction: "你是一个专业的搜索助手。请直接使用 Google 搜索工具来查找用户请求的信息，并返回最相关、最准确的搜索结果。不需要过多的解释，直接提供搜索到的事实即可。",
		Tools: []tool.Tool{
			geminitool.GoogleSearch{},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("llmagent.New() for search agent error: %w", err)
	}

	// 使用 agenttool 将该 Agent 封装成一个工具
	// 这样主 Agent 看到的将是一个名为 google_search 的普通工具
	googleSearchTool := agenttool.New(searchAgent, &agenttool.Config{
		SkipSummarization: true, // 搜索结果直接返回，不需要子 Agent 特外总结
	})

	return googleSearchTool, nil
}

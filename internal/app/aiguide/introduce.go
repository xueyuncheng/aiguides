package aiguide

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/agent/workflowagents/sequentialagent"
	"google.golang.org/adk/model"
	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
	"google.golang.org/adk/tool/geminitool"
)

func NewSequentialAgent(model model.LLM) (agent.Agent, error) {
	// Create WebFetch tool once to be shared across all agents
	webFetchTool, err := functiontool.New(functiontool.Config{
		Name:        "WebFetch",
		Description: "访问指定的 URL 并获取网页内容，用于核实信息的真实性。",
	}, webFetchHandler)
	if err != nil {
		return nil, fmt.Errorf("functiontool.New() error, err = %w", err)
	}

	searchAgent, err := NewSearchAgent(model, webFetchTool)
	if err != nil {
		return nil, fmt.Errorf("NewSearchAgent() error, err = %w", err)
	}

	factCheckAgent, err := NewFactCheckAgent(model, webFetchTool)
	if err != nil {
		return nil, fmt.Errorf("NewFactCheckAgent() error, err = %w", err)
	}

	cfg := sequentialagent.Config{
		AgentConfig: agent.Config{
			Name:        "SequentialAgent",
			Description: "调用搜索和总结子 Agent 以处理用户查询",
			SubAgents:   []agent.Agent{searchAgent, factCheckAgent},
		},
	}
	agent, err := sequentialagent.New(cfg)
	if err != nil {
		slog.Error("sequentialagent.New() error", "err", err)
		return nil, fmt.Errorf("sequentialagent.New() error, err = %w", err)
	}

	return agent, nil
}

func NewSearchAgent(model model.LLM, webFetchTool tool.Tool) (agent.Agent, error) {
	searchAgentConfig := llmagent.Config{
		Name:  "SearchAgent",
		Model: model,
		Instruction: `你是一个搜索机器人。当用户想你来询问问题时，你可以通过调用 GoogleSearch 工具来在网络上查找相关信息。
对于 Google 搜索的至少前 3 个结果，你都需要调用 webFetchTool 工具来获取网页内容，以确保你获得的信息是准确和最新的。

关键要求：
1. 仅提供搜索结果中明确找到的链接和信息。
2. 严禁幻觉或猜测任何 URL，尤其是 YouTube 链接。如果搜索结果中没有该链接，请不要包含它。
3. 对于每个来源，提供标题、简要描述和准确的 URL。
4. 优先考虑链接的准确性和相关性，而不是结果的数量。
`,
		Description: "该 Agent 可以根据用户查询在网络上搜索一个人的所有演讲和文章。",
		OutputKey:   "search_agent_output",
		Tools: []tool.Tool{
			geminitool.GoogleSearch{},
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

func NewFactCheckAgent(model model.LLM, webFetchTool tool.Tool) (agent.Agent, error) {
	factCheckAgent := llmagent.Config{
		Name:  "FactCheckAgent",
		Model: model,
		Instruction: `你是一个事实核查机器人。你的任务是核实用户提供的内容（特别是 SearchAgent 的输出）中的事实是否准确。

关键要求：
1. 如果内容中包含链接，请务必使用 WebFetch 工具访问这些链接。
2. 检查链接是否真实存在，以及网页的实际内容是否与 SearchAgent 描述的内容一致。
3. 如果发现信息不实、链接失效或内容对不上，请明确指出。
4. 请确保你的核查报告涵盖所有重要信息，条理清晰，易于理解。
`,
		Description: "该 Agent 可以通过访问网页来核实用户提供的内容中的事实是否准确。",
		OutputKey:   "fact_check_agent_output",
		Tools:       []tool.Tool{webFetchTool},
	}
	agent, err := llmagent.New(factCheckAgent)
	if err != nil {
		slog.Error("llmagent.New() error", "err", err)
		return nil, fmt.Errorf("llmagent.New() error, err = %w", err)
	}

	return agent, nil
}

type WebFetchArgs struct {
	URL string `json:"url"`
}

type WebFetchResults struct {
	Content string `json:"content"`
}

func webFetchHandler(ctx tool.Context, args WebFetchArgs) (WebFetchResults, error) {
	slog.Info("Fetching web page", "url", args.URL)

	// Configure HTTP client with proxy
	proxyURL, err := url.Parse("http://localhost:7890")
	if err != nil {
		slog.Error("url.Parse() error", "err", err)
		return WebFetchResults{}, fmt.Errorf("url.Parse() error, err = %w", err)
	}

	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		},
	}

	resp, err := client.Get(args.URL)
	if err != nil {
		slog.Error("client.Get() error", "err", err)
		return WebFetchResults{}, fmt.Errorf("client.Get() error, err = %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return WebFetchResults{
			Content: fmt.Sprintf("Error: received status code %d", resp.StatusCode),
		}, nil
	}

	// Limit to 50KB to avoid context overflow
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1024*50))
	if err != nil {
		slog.Error("io.ReadAll() error", "err", err)
		return WebFetchResults{}, fmt.Errorf("io.ReadAll() error, err = %w", err)
	}

	return WebFetchResults{Content: string(body)}, nil
}

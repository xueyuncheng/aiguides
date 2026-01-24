package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
)

// WebSearchInput 定义网页搜索的输入参数
type WebSearchInput struct {
	Query      string `json:"query" jsonschema:"必填，搜索关键词或问题"`
	NumResults int    `json:"num_results,omitempty" jsonschema:"返回结果数量，范围1-20，默认5"`
	Language   string `json:"language,omitempty" jsonschema:"搜索语言，如'zh-CN'中文、'en'英文，默认'zh-CN'"`
}

// WebSearchOutput 定义网页搜索的输出结果
type WebSearchOutput struct {
	Success bool                  `json:"success"`
	Results []WebSearchResultItem `json:"results,omitempty"`
	Query   string                `json:"query,omitempty"`
	Message string                `json:"message,omitempty"`
	Error   string                `json:"error,omitempty"`
}

// WebSearchResultItem 单个搜索结果
type WebSearchResultItem struct {
	Title   string `json:"title"`
	Link    string `json:"link"`
	Snippet string `json:"snippet"`
	Engine  string `json:"engine,omitempty"` // 来源引擎（如 google, bing 等）
}

// SearXNGConfig SearXNG 配置
type SearXNGConfig struct {
	InstanceURL string // SearXNG 实例 URL
}

// WebSearchConfig 网页搜索配置
type WebSearchConfig struct {
	SearXNG SearXNGConfig
}

// SearXNGResponse SearXNG API 响应结构
type SearXNGResponse struct {
	Query           string          `json:"query"`
	NumberOfResults int             `json:"number_of_results"`
	Results         []SearXNGResult `json:"results"`
	Suggestions     []string        `json:"suggestions"`
}

// SearXNGResult SearXNG 单个搜索结果
type SearXNGResult struct {
	Title   string  `json:"title"`
	URL     string  `json:"url"`
	Content string  `json:"content"`
	Engine  string  `json:"engine"`
	Score   float64 `json:"score"`
}

// NewWebSearchTool 创建网页搜索工具
//
// 使用 SearXNG 开源搜索引擎聚合服务，支持：
// - 完全免费，无需 API Key
// - 搜索全网，无域名限制
// - 聚合 Google、Bing、DuckDuckGo 等多个搜索引擎
// - 自动故障转移到备用实例
func NewWebSearchTool(config WebSearchConfig) (tool.Tool, error) {
	// 设置默认值
	if config.SearXNG.InstanceURL == "" {
		config.SearXNG.InstanceURL = "https://searx.be" // 默认使用 searx.be 公共实例
		slog.Info("使用默认 SearXNG 实例", "instance", config.SearXNG.InstanceURL)
	}

	toolConfig := functiontool.Config{
		Name:        "web_search",
		Description: "搜索互联网上的实时信息。当用户询问需要最新信息、实时数据、当前事件、新闻、统计数据或任何你不确定答案的问题时使用此工具。",
	}

	handler := func(ctx tool.Context, input WebSearchInput) (*WebSearchOutput, error) {
		return executeWebSearch(ctx, input, config)
	}

	return functiontool.New(toolConfig, handler)
}

// executeWebSearch 执行网页搜索（带自动故障转移）
func executeWebSearch(ctx context.Context, input WebSearchInput, config WebSearchConfig) (*WebSearchOutput, error) {
	// 参数验证
	if input.Query == "" {
		return &WebSearchOutput{
			Success: false,
			Error:   "搜索查询不能为空",
		}, nil
	}

	// 设置默认值
	numResults := input.NumResults
	if numResults <= 0 {
		numResults = 5
	}
	if numResults > 20 {
		numResults = 20
	}

	language := input.Language
	if language == "" {
		language = "zh-CN" // 默认使用中文
	}

	slog.Info("执行网页搜索", "query", input.Query, "num_results", numResults, "language", language)

	// 直接使用配置的实例
	result, err := searchSearXNG(ctx, config.SearXNG.InstanceURL, input.Query, numResults, language)
	if err == nil {
		slog.Info("搜索成功", "instance", config.SearXNG.InstanceURL, "results_count", len(result.Results))
		return result, nil
	}

	// 搜索失败
	slog.Error("SearXNG 搜索失败", "instance", config.SearXNG.InstanceURL, "err", err)
	return &WebSearchOutput{
		Success: false,
		Error:   fmt.Sprintf("搜索失败，请稍后重试。错误: %v", err),
	}, nil
}

// searchSearXNG 调用单个 SearXNG 实例进行搜索
func searchSearXNG(ctx context.Context, instanceURL, query string, numResults int, language string) (*WebSearchOutput, error) {
	// 构建请求 URL
	apiURL := fmt.Sprintf("%s/search", instanceURL)
	params := url.Values{}
	params.Add("q", query)
	params.Add("format", "json")
	params.Add("language", language)
	params.Add("pageno", "1")

	fullURL := fmt.Sprintf("%s?%s", apiURL, params.Encode())

	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	// 创建 HTTP 请求
	req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	// 设置 User-Agent（某些实例可能需要）
	req.Header.Set("User-Agent", "AIGuides/1.0 (Web Search Bot)")

	// 发送请求
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	// 检查 HTTP 状态码
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP 错误: %d, body: %s", resp.StatusCode, string(body))
	}

	// 解析 JSON 响应
	var searxResp SearXNGResponse
	if err := json.Unmarshal(body, &searxResp); err != nil {
		return nil, fmt.Errorf("解析 JSON 失败: %w", err)
	}

	// 转换为输出格式
	results := make([]WebSearchResultItem, 0, numResults)
	for i, item := range searxResp.Results {
		if i >= numResults {
			break
		}
		results = append(results, WebSearchResultItem{
			Title:   item.Title,
			Link:    item.URL,
			Snippet: item.Content,
			Engine:  item.Engine,
		})
	}

	// 检查是否有结果
	if len(results) == 0 {
		return &WebSearchOutput{
			Success: false,
			Query:   query,
			Message: "未找到搜索结果",
		}, nil
	}

	return &WebSearchOutput{
		Success: true,
		Results: results,
		Query:   query,
		Message: fmt.Sprintf("找到 %d 个搜索结果", len(results)),
	}, nil
}

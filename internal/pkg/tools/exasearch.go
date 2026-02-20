package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
)

const exaAPIURL = "https://api.exa.ai/search"

// ExaConfig Exa API 配置
type ExaConfig struct {
	APIKey string // Exa API Key
}

// ExaSearchInput 定义 Exa 搜索的输入参数
type ExaSearchInput struct {
	Query           string `json:"query" jsonschema:"必填，搜索关键词或问题"`
	NumResults      int    `json:"num_results,omitempty" jsonschema:"返回结果数量，范围1-10，默认5"`
	IncludePageText bool   `json:"include_page_text,omitempty" jsonschema:"是否包含网页正文内容，默认false。当需要深度阅读网页内容时设为true"`
}

// ExaSearchOutput 定义 Exa 搜索的输出结果
type ExaSearchOutput struct {
	Success bool              `json:"success"`
	Results []ExaSearchResult `json:"results,omitempty"`
	Query   string            `json:"query,omitempty"`
	Message string            `json:"message,omitempty"`
	Error   string            `json:"error,omitempty"`
}

// ExaSearchResult 单个 Exa 搜索结果
type ExaSearchResult struct {
	Title   string  `json:"title"`
	URL     string  `json:"url"`
	Snippet string  `json:"snippet,omitempty"`
	Text    string  `json:"text,omitempty"` // 网页正文（当 IncludePageText=true 时返回）
	Score   float64 `json:"score,omitempty"`
}

// exaRequest Exa API 请求体
type exaRequest struct {
	Query      string      `json:"query"`
	NumResults int         `json:"numResults"`
	Type       string      `json:"type"` // "neural", "keyword", "auto"
	Contents   exaContents `json:"contents"`
}

// exaContents Exa 内容配置
type exaContents struct {
	Text       *exaTextOptions `json:"text,omitempty"`
	Highlights *exaHighlights  `json:"highlights,omitempty"`
}

// exaTextOptions 全文抓取选项
type exaTextOptions struct {
	MaxCharacters int  `json:"maxCharacters,omitempty"`
	IncludeHTML   bool `json:"includeHtml"`
}

// exaHighlights 摘要高亮选项
type exaHighlights struct {
	NumSentences     int `json:"numSentences,omitempty"`
	HighlightsPerURL int `json:"highlightsPerUrl,omitempty"`
}

// exaResponse Exa API 响应体
type exaResponse struct {
	Results []exaResult `json:"results"`
}

// exaResult Exa API 单个结果
type exaResult struct {
	ID         string   `json:"id"`
	URL        string   `json:"url"`
	Title      string   `json:"title"`
	Score      float64  `json:"score"`
	Text       string   `json:"text,omitempty"`
	Highlights []string `json:"highlights,omitempty"`
}

// NewExaSearchTool 创建 Exa 搜索工具
//
// Exa 是一个基于神经网络的搜索引擎，擅长：
// - 深度语义理解，找到高质量的参考资料
// - 直接抓取网页正文内容供 AI 阅读
// - 需要 API Key（https://exa.ai/）
func NewExaSearchTool(config ExaConfig) (tool.Tool, error) {
	toolConfig := functiontool.Config{
		Name:        "exa_search",
		Description: "使用 Exa 神经网络搜索引擎进行深度语义搜索。适合需要深度理解、寻找高质量参考资料、学术内容或需要直接获取网页正文供 AI 阅读的任务。与 web_search 互补：web_search 适合实时事实性查询，exa_search 适合深度语义理解。",
	}

	handler := func(ctx tool.Context, input ExaSearchInput) (*ExaSearchOutput, error) {
		return executeExaSearch(ctx, input, config)
	}

	return functiontool.New(toolConfig, handler)
}

// executeExaSearch 执行 Exa 搜索
func executeExaSearch(ctx context.Context, input ExaSearchInput, config ExaConfig) (*ExaSearchOutput, error) {
	// 参数验证
	if input.Query == "" {
		slog.Error("Exa 搜索查询不能为空")
		return &ExaSearchOutput{
			Success: false,
			Error:   "搜索查询不能为空",
		}, nil
	}

	if config.APIKey == "" {
		slog.Error("Exa API Key 未配置")
		return &ExaSearchOutput{
			Success: false,
			Error:   "Exa API Key 未配置，请在配置文件中设置 exa_search.api_key",
		}, nil
	}

	// 设置默认值
	numResults := input.NumResults
	if numResults <= 0 {
		numResults = 5
	}
	if numResults > 10 {
		numResults = 10
	}

	slog.Info("执行 Exa 搜索", "query", input.Query, "num_results", numResults, "include_page_text", input.IncludePageText)

	result, err := searchExa(ctx, input.Query, numResults, input.IncludePageText, config.APIKey)
	if err != nil {
		slog.Error("Exa 搜索失败", "err", err)
		return &ExaSearchOutput{
			Success: false,
			Error:   fmt.Sprintf("搜索失败，请稍后重试。错误: %v", err),
		}, nil
	}

	return result, nil
}

// searchExa 调用 Exa API 进行搜索
func searchExa(ctx context.Context, query string, numResults int, includePageText bool, apiKey string) (*ExaSearchOutput, error) {
	return searchExaWithURL(ctx, query, numResults, includePageText, apiKey, exaAPIURL)
}

// searchExaWithURL 调用指定 URL 的 Exa API 进行搜索（支持测试注入）
func searchExaWithURL(ctx context.Context, query string, numResults int, includePageText bool, apiKey, apiURL string) (*ExaSearchOutput, error) {
	// 构建请求体
	reqBody := exaRequest{
		Query:      query,
		NumResults: numResults,
		Type:       "auto",
	}

	if includePageText {
		reqBody.Contents = exaContents{
			Text: &exaTextOptions{
				MaxCharacters: 2000,
				IncludeHTML:   false,
			},
		}
	} else {
		reqBody.Contents = exaContents{
			Highlights: &exaHighlights{
				NumSentences:     3,
				HighlightsPerURL: 1,
			},
		}
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		slog.Error("json.Marshal() error", "err", err)
		return nil, fmt.Errorf("序列化请求体失败: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	// 创建 HTTP 请求
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(bodyBytes))
	if err != nil {
		slog.Error("http.NewRequestWithContext() error", "url", apiURL, "err", err)
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("User-Agent", "AIGuides/1.0")

	// 发送请求
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		slog.Error("http.DefaultClient.Do() error", "url", apiURL, "err", err)
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.Error("io.ReadAll() error", "err", err)
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	// 检查 HTTP 状态码
	if resp.StatusCode != http.StatusOK {
		slog.Error("HTTP status code error", "status", resp.StatusCode, "body", string(body))
		return nil, fmt.Errorf("HTTP 错误: %d, body: %s", resp.StatusCode, string(body))
	}

	// 解析 JSON 响应
	var exaResp exaResponse
	if err := json.Unmarshal(body, &exaResp); err != nil {
		slog.Error("json.Unmarshal() error", "err", err)
		return nil, fmt.Errorf("解析 JSON 失败: %w", err)
	}

	// 转换为输出格式
	results := make([]ExaSearchResult, 0, len(exaResp.Results))
	for _, item := range exaResp.Results {
		result := ExaSearchResult{
			Title: item.Title,
			URL:   item.URL,
			Score: item.Score,
			Text:  item.Text,
		}
		// 使用第一个 highlight 作为 snippet
		if len(item.Highlights) > 0 {
			result.Snippet = item.Highlights[0]
		}
		results = append(results, result)
	}

	// 检查是否有结果
	if len(results) == 0 {
		slog.Error("Exa 未找到搜索结果", "query", query)
		return &ExaSearchOutput{
			Success: false,
			Query:   query,
			Message: "未找到搜索结果",
		}, nil
	}

	return &ExaSearchOutput{
		Success: true,
		Results: results,
		Query:   query,
		Message: fmt.Sprintf("找到 %d 个搜索结果", len(results)),
	}, nil
}

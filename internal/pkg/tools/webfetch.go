package tools

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
)

// WebFetchInput 定义网页获取的输入参数
type WebFetchInput struct {
	URL string `json:"url" jsonschema:"要获取内容的网页 URL"`
}

// WebFetchOutput 定义网页获取的输出
type WebFetchOutput struct {
	Success bool   `json:"success"`
	Content string `json:"content,omitempty"`
	Error   string `json:"error,omitempty"`
}

// NewWebFetchTool 使用 functiontool.New 创建网页内容获取工具。
//
// 该工具通过 HTTP GET 请求抓取网页的 HTML 内容，并返回用于总结分析的数据。
func NewWebFetchTool() (tool.Tool, error) {
	client := &http.Client{Timeout: 5 * time.Second}

	config := functiontool.Config{
		Name:        "fetch_webpage",
		Description: "获取指定 URL 的网页内容。返回网页的 HTML 文本，适用于网页分析、内容提取和总结任务。",
	}

	handler := func(ctx tool.Context, input WebFetchInput) (*WebFetchOutput, error) {
		return fetchWebPage(ctx, client, input), nil
	}

	return functiontool.New(config, handler)
}

// fetchWebPage 执行实际的网页抓取逻辑。
func fetchWebPage(ctx context.Context, client *http.Client, input WebFetchInput) *WebFetchOutput {
	if input.URL == "" {
		return &WebFetchOutput{Success: false, Error: "URL 不能为空"}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, input.URL, nil)
	if err != nil {
		return &WebFetchOutput{Success: false, Error: fmt.Sprintf("创建请求失败: %v", err)}
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")

	resp, err := client.Do(req)
	if err != nil {
		return &WebFetchOutput{Success: false, Error: fmt.Sprintf("请求失败: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return &WebFetchOutput{Success: false, Error: fmt.Sprintf("HTTP 状态码错误: %d %s", resp.StatusCode, resp.Status)}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &WebFetchOutput{Success: false, Error: fmt.Sprintf("读取响应失败: %v", err)}
	}

	return &WebFetchOutput{
		Success: true,
		Content: string(body),
	}
}

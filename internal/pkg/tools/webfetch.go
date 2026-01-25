package tools

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	readability "codeberg.org/readeck/go-readability/v2"
	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
)

const (
	webFetchTimeout   = 30 * time.Second
	webFetchUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/143.0.0.0 Safari/537.36"
)

// WebFetchInput 定义网页抓取的输入参数
type WebFetchInput struct {
	URL string `json:"url" jsonschema:"必填，要获取内容的网页URL"`
}

// WebFetchOutput 定义网页抓取的输出结果
type WebFetchOutput struct {
	Success       bool   `json:"success"`
	URL           string `json:"url"`
	Title         string `json:"title"`
	TextContent   string `json:"text_content"`             // 纯文本内容（主要）
	Content       string `json:"content"`                  // HTML内容（备选）
	Byline        string `json:"byline"`                   // 作者
	Excerpt       string `json:"excerpt"`                  // 摘要
	Length        int    `json:"length"`                   // 字数
	SiteName      string `json:"site_name"`                // 站点名称
	PublishedTime string `json:"published_time,omitempty"` // 发布时间
	ModifiedTime  string `json:"modified_time,omitempty"`  // 修改时间
	Image         string `json:"image,omitempty"`          // 主图
	Favicon       string `json:"favicon,omitempty"`        // 图标
	Language      string `json:"language,omitempty"`       // 语言
	Message       string `json:"message,omitempty"`        // 成功消息
	Error         string `json:"error,omitempty"`          // 错误消息
}

// NewWebFetchTool 创建网页抓取工具
//
// 使用 go-readability 开源库提取网页可读内容，支持：
// - 自动提取文章正文（纯文本和HTML格式）
// - 提取元数据（标题、作者、发布时间等）
// - 过滤广告、导航栏等无关内容
// - 适用于新闻、博客、技术文档等内容型网页
func NewWebFetchTool() (tool.Tool, error) {
	toolConfig := functiontool.Config{
		Name:        "web_fetch",
		Description: "获取网页的完整内容和元数据。当 web_search 返回的链接需要详细阅读时使用此工具。可以提取文章标题、作者、正文、发布时间等信息。",
	}

	handler := func(ctx tool.Context, input WebFetchInput) (*WebFetchOutput, error) {
		return executeWebFetch(ctx, input)
	}

	return functiontool.New(toolConfig, handler)
}

// executeWebFetch 执行网页抓取
func executeWebFetch(ctx context.Context, input WebFetchInput) (*WebFetchOutput, error) {
	// 1. 参数验证
	if input.URL == "" {
		slog.Error("URL 不能为空")
		return &WebFetchOutput{
			Success: false,
			Error:   "URL 不能为空",
		}, nil
	}

	// 2. 解析 URL
	parsedURL, err := url.Parse(input.URL)
	if err != nil {
		return &WebFetchOutput{
			Success: false,
			Error:   fmt.Sprintf("无效的 URL: %v", err),
		}, nil
	}

	slog.Info("开始抓取网页", "url", input.URL)

	// 3. 发送 HTTP 请求
	ctx, cancel := context.WithTimeout(ctx, webFetchTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", input.URL, nil)
	if err != nil {
		slog.Error("http.NewRequestWithContext() error", "url", input.URL, "err", err)
		return &WebFetchOutput{
			Success: false,
			URL:     input.URL,
			Error:   fmt.Sprintf("创建请求失败: %v", err),
		}, nil
	}
	req.Header.Set("User-Agent", webFetchUserAgent)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		slog.Error("http.DefaultClient.Do() error", "url", input.URL, "err", err)
		return &WebFetchOutput{
			Success: false,
			URL:     input.URL,
			Error:   fmt.Sprintf("请求失败: %v", err),
		}, nil
	}
	defer resp.Body.Close()

	// 4. 检查状态码
	if resp.StatusCode != http.StatusOK {
		slog.Error("HTTP status code error", "status", resp.StatusCode, "url", input.URL)
		return &WebFetchOutput{
			Success: false,
			URL:     input.URL,
			Error:   fmt.Sprintf("HTTP 错误: %d %s", resp.StatusCode, resp.Status),
		}, nil
	}

	// 5. 使用 go-readability v2 解析
	article, err := readability.FromReader(resp.Body, parsedURL)
	if err != nil {
		slog.Error("readability.FromReader() error", "url", input.URL, "err", err)
		return &WebFetchOutput{
			Success: false,
			URL:     input.URL,
			Error:   fmt.Sprintf("解析网页失败: %v", err),
		}, nil
	}

	// 6. 检查是否有有效内容
	if article.Node == nil {
		slog.Error("网页内容为空或无法提取可读内容", "url", input.URL)
		return &WebFetchOutput{
			Success: false,
			URL:     input.URL,
			Error:   "网页内容为空或无法提取可读内容",
		}, nil
	}

	// 7. 渲染 HTML 内容
	var htmlBuf bytes.Buffer
	if err := article.RenderHTML(&htmlBuf); err != nil {
		slog.Warn("渲染 HTML 失败", "err", err)
	}

	// 8. 渲染文本内容（优先）
	var textBuf bytes.Buffer
	if err := article.RenderText(&textBuf); err != nil {
		slog.Warn("渲染文本失败", "err", err)
	}

	// 9. 处理时间字段
	publishedTime := ""
	if pt, err := article.PublishedTime(); err == nil {
		publishedTime = pt.Format(time.RFC3339)
	}

	modifiedTime := ""
	if mt, err := article.ModifiedTime(); err == nil {
		modifiedTime = mt.Format(time.RFC3339)
	}

	// 10. 计算字数（使用文本内容的长度）
	textContent := textBuf.String()
	length := len([]rune(textContent))

	slog.Info("网页抓取成功", "url", input.URL, "title", article.Title(), "length", length)

	// 11. 返回结果
	return &WebFetchOutput{
		Success:       true,
		URL:           input.URL,
		Title:         article.Title(),
		TextContent:   textContent,
		Content:       htmlBuf.String(),
		Byline:        article.Byline(),
		Excerpt:       article.Excerpt(),
		Length:        length,
		SiteName:      article.SiteName(),
		PublishedTime: publishedTime,
		ModifiedTime:  modifiedTime,
		Image:         article.ImageURL(),
		Favicon:       article.Favicon(),
		Language:      article.Language(),
		Message:       fmt.Sprintf("成功获取网页内容，共 %d 字", length),
	}, nil
}

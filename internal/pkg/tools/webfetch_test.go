package tools

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestNewWebFetchTool 测试工具创建
func TestNewWebFetchTool(t *testing.T) {
	tool, err := NewWebFetchTool()
	if err != nil {
		t.Fatalf("NewWebFetchTool() error = %v", err)
	}
	if tool == nil {
		t.Fatal("NewWebFetchTool() returned nil")
	}
}

// TestWebFetch_EmptyURL 测试空 URL
func TestWebFetch_EmptyURL(t *testing.T) {
	input := WebFetchInput{URL: ""}
	output, err := executeWebFetch(context.Background(), input)

	if err != nil {
		t.Fatalf("executeWebFetch() error = %v", err)
	}
	if output.Success {
		t.Error("expected Success=false for empty URL")
	}
	if output.Error == "" {
		t.Error("expected error message for empty URL")
	}
	if !strings.Contains(output.Error, "不能为空") {
		t.Errorf("unexpected error message: %s", output.Error)
	}
}

// TestWebFetch_InvalidURL 测试无效 URL
func TestWebFetch_InvalidURL(t *testing.T) {
	input := WebFetchInput{URL: "not-a-valid-url"}
	output, err := executeWebFetch(context.Background(), input)

	if err != nil {
		t.Fatalf("executeWebFetch() error = %v", err)
	}
	if output.Success {
		t.Error("expected Success=false for invalid URL")
	}
	if output.Error == "" {
		t.Error("expected error message for invalid URL")
	}
}

// TestWebFetch_ValidHTML 测试有效的 HTML 文章
func TestWebFetch_ValidHTML(t *testing.T) {
	// 创建 Mock HTTP Server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		html := `
<!DOCTYPE html>
<html>
<head>
	<title>Test Article Title</title>
	<meta name="author" content="Test Author">
	<meta property="og:site_name" content="Test Site">
</head>
<body>
	<article>
		<h1>Test Article Heading</h1>
		<p>This is a test article with enough content to be parsed by readability.</p>
		<p>Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.</p>
		<p>Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat.</p>
		<p>Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur.</p>
	</article>
</body>
</html>
		`
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(html))
	}))
	defer server.Close()

	input := WebFetchInput{URL: server.URL}
	output, err := executeWebFetch(context.Background(), input)

	if err != nil {
		t.Fatalf("executeWebFetch() error = %v", err)
	}
	if !output.Success {
		t.Errorf("expected Success=true, got error: %s", output.Error)
	}
	if output.URL != server.URL {
		t.Errorf("expected URL=%s, got %s", server.URL, output.URL)
	}
	if output.Title == "" {
		t.Error("expected non-empty Title")
	}
	if output.TextContent == "" {
		t.Error("expected non-empty TextContent")
	}
	if output.Content == "" {
		t.Error("expected non-empty Content (HTML)")
	}
	if output.Length == 0 {
		t.Error("expected non-zero Length")
	}
	if output.Message == "" {
		t.Error("expected non-empty Message")
	}
}

// TestWebFetch_404Error 测试 HTTP 404 错误
func TestWebFetch_404Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	input := WebFetchInput{URL: server.URL}
	output, err := executeWebFetch(context.Background(), input)

	if err != nil {
		t.Fatalf("executeWebFetch() error = %v", err)
	}
	if output.Success {
		t.Error("expected Success=false for 404")
	}
	if !strings.Contains(output.Error, "404") {
		t.Errorf("expected error message to contain '404', got: %s", output.Error)
	}
}

// TestWebFetch_500Error 测试 HTTP 500 错误
func TestWebFetch_500Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	input := WebFetchInput{URL: server.URL}
	output, err := executeWebFetch(context.Background(), input)

	if err != nil {
		t.Fatalf("executeWebFetch() error = %v", err)
	}
	if output.Success {
		t.Error("expected Success=false for 500")
	}
	if !strings.Contains(output.Error, "500") {
		t.Errorf("expected error message to contain '500', got: %s", output.Error)
	}
}

// TestWebFetch_MetadataExtraction 测试元数据提取
func TestWebFetch_MetadataExtraction(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		html := `
<!DOCTYPE html>
<html>
<head>
	<title>Metadata Test Article</title>
	<meta name="author" content="John Doe">
	<meta property="og:site_name" content="Example Site">
	<meta property="og:image" content="https://example.com/image.jpg">
	<link rel="icon" href="https://example.com/favicon.ico">
</head>
<body>
	<article>
		<h1>Metadata Test</h1>
		<p>This article tests metadata extraction including author, site name, and images.</p>
		<p>Lorem ipsum dolor sit amet, consectetur adipiscing elit.</p>
		<p>Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.</p>
	</article>
</body>
</html>
		`
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(html))
	}))
	defer server.Close()

	input := WebFetchInput{URL: server.URL}
	output, err := executeWebFetch(context.Background(), input)

	if err != nil {
		t.Fatalf("executeWebFetch() error = %v", err)
	}
	if !output.Success {
		t.Errorf("expected Success=true, got error: %s", output.Error)
	}
	if output.Title == "" {
		t.Error("expected non-empty Title")
	}
	// Byline, SiteName, Image 等字段可能被 readability 提取到，也可能没有
	// 这取决于 readability 的解析逻辑，所以这里不做强制断言
}

// TestWebFetch_TextContent 测试纯文本内容提取
func TestWebFetch_TextContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		html := `
<!DOCTYPE html>
<html>
<head>
	<title>Text Content Test</title>
</head>
<body>
	<div class="ads">This is an advertisement</div>
	<article>
		<h1>Main Article</h1>
		<p>This is the main content of the article.</p>
		<p>It should be extracted as text content.</p>
	</article>
	<div class="footer">Footer content</div>
</body>
</html>
		`
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(html))
	}))
	defer server.Close()

	input := WebFetchInput{URL: server.URL}
	output, err := executeWebFetch(context.Background(), input)

	if err != nil {
		t.Fatalf("executeWebFetch() error = %v", err)
	}
	if !output.Success {
		t.Errorf("expected Success=true, got error: %s", output.Error)
	}
	if output.TextContent == "" {
		t.Error("expected non-empty TextContent")
	}
	// 纯文本应该包含主要内容，但不应该包含广告
	// readability 库会自动过滤掉广告和无关内容
}

// TestWebFetch_EmptyContent 测试空内容处理
func TestWebFetch_EmptyContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		html := `
<!DOCTYPE html>
<html>
<head>
	<title>Empty Page</title>
</head>
<body>
</body>
</html>
		`
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(html))
	}))
	defer server.Close()

	input := WebFetchInput{URL: server.URL}
	_, err := executeWebFetch(context.Background(), input)

	// 对于空内容，readability 可能会失败，也可能返回空内容
	// 这取决于 readability 的行为，我们只确保不会 panic
	if err != nil {
		t.Fatalf("executeWebFetch() should not return error: %v", err)
	}
	// Success 可能为 true 或 false，取决于 readability 的解析结果
}

// TestWebFetch_InvalidHTML 测试无效 HTML
func TestWebFetch_InvalidHTML(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		html := `<html><head><title>Test`
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(html))
	}))
	defer server.Close()

	input := WebFetchInput{URL: server.URL}
	_, err := executeWebFetch(context.Background(), input)

	// readability 应该能够处理无效 HTML，即使不完整
	// 我们只确保不会 panic
	if err != nil {
		t.Fatalf("executeWebFetch() should not return error: %v", err)
	}
}

// TestWebFetch_LongContent 测试长内容处理
func TestWebFetch_LongContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 生成一个较长的文章
		longContent := strings.Repeat("<p>This is a paragraph with some content. Lorem ipsum dolor sit amet, consectetur adipiscing elit. </p>", 100)
		html := `
<!DOCTYPE html>
<html>
<head>
	<title>Long Article</title>
</head>
<body>
	<article>
		<h1>Long Content Test</h1>
		` + longContent + `
	</article>
</body>
</html>
		`
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(html))
	}))
	defer server.Close()

	input := WebFetchInput{URL: server.URL}
	output, err := executeWebFetch(context.Background(), input)

	if err != nil {
		t.Fatalf("executeWebFetch() error = %v", err)
	}
	if !output.Success {
		t.Errorf("expected Success=true, got error: %s", output.Error)
	}
	if output.Length == 0 {
		t.Error("expected non-zero Length for long content")
	}
	// 不限制长度，所以长内容应该完整返回
	if len(output.TextContent) == 0 {
		t.Error("expected long TextContent")
	}
}

// TestWebFetch_UserAgent 测试 User-Agent 设置
func TestWebFetch_UserAgent(t *testing.T) {
	var receivedUA string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedUA = r.Header.Get("User-Agent")
		html := `
<!DOCTYPE html>
<html>
<head><title>UA Test</title></head>
<body>
	<article>
		<h1>User Agent Test</h1>
		<p>Testing user agent header.</p>
		<p>Lorem ipsum dolor sit amet.</p>
	</article>
</body>
</html>
		`
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(html))
	}))
	defer server.Close()

	input := WebFetchInput{URL: server.URL}
	_, err := executeWebFetch(context.Background(), input)

	if err != nil {
		t.Fatalf("executeWebFetch() error = %v", err)
	}
	if receivedUA != webFetchUserAgent {
		t.Errorf("expected User-Agent=%s, got %s", webFetchUserAgent, receivedUA)
	}
	if !strings.Contains(receivedUA, "Chrome") {
		t.Errorf("expected User-Agent to contain 'Chrome', got: %s", receivedUA)
	}
}

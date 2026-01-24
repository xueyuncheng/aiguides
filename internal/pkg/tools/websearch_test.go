package tools

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Mock SearXNG 响应
const mockSearXNGResponse = `{
  "query": "golang",
  "number_of_results": 100,
  "results": [
    {
      "title": "The Go Programming Language",
      "url": "https://go.dev",
      "content": "Go is an open source programming language that makes it easy to build simple, reliable, and efficient software.",
      "engine": "google",
      "score": 1.0
    },
    {
      "title": "Go 中文文档",
      "url": "https://go-zh.org",
      "content": "Go 语言中文官方文档",
      "engine": "bing",
      "score": 0.9
    },
    {
      "title": "Golang Tutorial",
      "url": "https://go.dev/doc/tutorial",
      "content": "A tutorial introduction to Go",
      "engine": "duckduckgo",
      "score": 0.85
    }
  ],
  "suggestions": ["golang tutorial", "golang download"]
}`

const mockEmptyResponse = `{
  "query": "golang",
  "number_of_results": 0,
  "results": [],
  "suggestions": []
}`

// TestNewWebSearchTool 测试工具创建
func TestNewWebSearchTool(t *testing.T) {
	tests := []struct {
		name    string
		config  WebSearchConfig
		wantErr bool
	}{
		{
			name: "有效配置",
			config: WebSearchConfig{
				SearXNG: SearXNGConfig{
					InstanceURL: "https://searx.be",
				},
			},
			wantErr: false,
		},
		{
			name: "使用默认值",
			config: WebSearchConfig{
				SearXNG: SearXNGConfig{},
			},
			wantErr: false,
		},
		{
			name: "带备用实例",
			config: WebSearchConfig{
				SearXNG: SearXNGConfig{
					InstanceURL: "https://searx.be",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tool, err := NewWebSearchTool(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewWebSearchTool() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tool == nil {
				t.Error("NewWebSearchTool() returned nil tool")
			}
		})
	}
}

// TestWebSearchInputValidation 测试参数验证和默认值
func TestWebSearchInputValidation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockSearXNGResponse))
	}))
	defer server.Close()

	config := WebSearchConfig{
		SearXNG: SearXNGConfig{
			InstanceURL: server.URL,
		},
	}

	tests := []struct {
		name          string
		input         WebSearchInput
		expectSuccess bool
		expectError   string
	}{
		{
			name: "空查询",
			input: WebSearchInput{
				Query: "",
			},
			expectSuccess: false,
			expectError:   "搜索查询不能为空",
		},
		{
			name: "正常查询",
			input: WebSearchInput{
				Query: "golang",
			},
			expectSuccess: true,
		},
		{
			name: "带参数查询",
			input: WebSearchInput{
				Query:      "golang",
				NumResults: 3,
				Language:   "zh-CN",
			},
			expectSuccess: true,
		},
		{
			name: "超大结果数量（应该被限制）",
			input: WebSearchInput{
				Query:      "golang",
				NumResults: 100,
			},
			expectSuccess: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := executeWebSearch(context.Background(), tt.input, config)
			if err != nil {
				t.Fatalf("executeWebSearch() error = %v", err)
			}

			if result.Success != tt.expectSuccess {
				t.Errorf("Expected success=%v, got %v. Error: %s", tt.expectSuccess, result.Success, result.Error)
			}

			if !tt.expectSuccess && result.Error != tt.expectError {
				t.Errorf("Expected error '%s', got '%s'", tt.expectError, result.Error)
			}
		})
	}
}

// TestSearXNGSearch 测试 SearXNG 搜索（使用 Mock Server）
func TestSearXNGSearch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 验证请求参数
		query := r.URL.Query().Get("q")
		if query == "" {
			t.Error("Missing query parameter")
		}

		format := r.URL.Query().Get("format")
		if format != "json" {
			t.Errorf("Expected format='json', got '%s'", format)
		}

		// 验证 User-Agent
		userAgent := r.Header.Get("User-Agent")
		if userAgent == "" {
			t.Error("Missing User-Agent header")
		}

		// 返回 mock 响应
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockSearXNGResponse))
	}))
	defer server.Close()

	result, err := searchSearXNG(
		context.Background(),
		server.URL,
		"golang",
		5,
	)

	if err != nil {
		t.Fatalf("searchSearXNG() failed: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected success=true, got false. Error: %s", result.Error)
	}

	if len(result.Results) != 3 {
		t.Errorf("Expected 3 results, got %d", len(result.Results))
	}

	if result.Results[0].Title != "The Go Programming Language" {
		t.Errorf("Unexpected title: %s", result.Results[0].Title)
	}

	if result.Results[0].Engine != "google" {
		t.Errorf("Expected engine='google', got '%s'", result.Results[0].Engine)
	}
}

// TestEmptySearchResults 测试空结果处理
func TestEmptySearchResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockEmptyResponse))
	}))
	defer server.Close()

	result, err := searchSearXNG(
		context.Background(),
		server.URL,
		"nonexistent query",
		5,
	)

	if err != nil {
		t.Fatalf("searchSearXNG() failed: %v", err)
	}

	if result.Success {
		t.Error("Expected success=false for empty results")
	}

	if result.Message != "未找到搜索结果" {
		t.Errorf("Unexpected message: %s", result.Message)
	}
}

// TestFallbackInstances 测试实例失败时的错误处理
// 注意：fallback 功能已移除，单实例失败时应该返回错误
func TestFallbackInstances(t *testing.T) {
	// 失败的实例返回 500 错误
	failServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer failServer.Close()

	config := WebSearchConfig{
		SearXNG: SearXNGConfig{
			InstanceURL: failServer.URL,
		},
	}

	result, err := executeWebSearch(
		context.Background(),
		WebSearchInput{Query: "golang", NumResults: 5},
		config,
	)

	if err != nil {
		t.Fatalf("executeWebSearch() failed: %v", err)
	}

	// 移除 fallback 后，实例失败应该返回失败
	if result.Success {
		t.Error("Expected failure when instance returns 500")
	}

	if result.Error == "" {
		t.Error("Expected error message when instance fails")
	}
}

// TestAllInstancesFail 测试实例失败的情况
// 注意：移除 fallback 后，这个测试和 TestFallbackInstances 类似
func TestAllInstancesFail(t *testing.T) {
	// 创建失败的服务器
	failServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer failServer.Close()

	config := WebSearchConfig{
		SearXNG: SearXNGConfig{
			InstanceURL: failServer.URL,
		},
	}

	result, err := executeWebSearch(
		context.Background(),
		WebSearchInput{Query: "golang"},
		config,
	)

	if err != nil {
		t.Fatalf("executeWebSearch() should not return error: %v", err)
	}

	if result.Success {
		t.Error("Expected success=false when instance fails")
	}

	if result.Error == "" {
		t.Error("Expected error message when instance fails")
	}
}

// TestInvalidJSON 测试无效 JSON 响应
func TestInvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("invalid json"))
	}))
	defer server.Close()

	_, err := searchSearXNG(
		context.Background(),
		server.URL,
		"golang",
		5,
	)

	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

// TestHTTPError 测试 HTTP 错误状态码
func TestHTTPError(t *testing.T) {
	statusCodes := []int{400, 403, 404, 500, 502, 503}

	for _, code := range statusCodes {
		t.Run(fmt.Sprintf("Status%d", code), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(code)
				w.Write([]byte("Error"))
			}))
			defer server.Close()

			_, err := searchSearXNG(
				context.Background(),
				server.URL,
				"golang",
				5,
			)

			if err == nil {
				t.Errorf("Expected error for status code %d", code)
			}
		})
	}
}

// TestNumResultsLimit 测试结果数量限制
func TestNumResultsLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockSearXNGResponse))
	}))
	defer server.Close()

	// 请求 2 个结果
	result, err := searchSearXNG(
		context.Background(),
		server.URL,
		"golang",
		2, // 只要 2 个结果
	)

	if err != nil {
		t.Fatalf("searchSearXNG() failed: %v", err)
	}

	if len(result.Results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(result.Results))
	}
}

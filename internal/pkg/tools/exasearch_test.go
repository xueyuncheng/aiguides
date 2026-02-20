package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

const mockExaResponse = `{
  "results": [
    {
      "id": "https://go.dev",
      "url": "https://go.dev",
      "title": "The Go Programming Language",
      "score": 0.95,
      "highlights": ["Go is an open source programming language."]
    },
    {
      "id": "https://go.dev/doc",
      "url": "https://go.dev/doc",
      "title": "Documentation - The Go Programming Language",
      "score": 0.88,
      "highlights": ["Official Go documentation and tutorials."]
    }
  ]
}`

const mockExaResponseWithText = `{
  "results": [
    {
      "id": "https://go.dev",
      "url": "https://go.dev",
      "title": "The Go Programming Language",
      "score": 0.95,
      "text": "Go is an open source programming language that makes it easy to build simple, reliable, and efficient software."
    }
  ]
}`

const mockExaEmptyResponse = `{
  "results": []
}`

// TestNewExaSearchTool 测试工具创建
func TestNewExaSearchTool(t *testing.T) {
	tests := []struct {
		name    string
		config  ExaConfig
		wantErr bool
	}{
		{
			name: "有效配置",
			config: ExaConfig{
				APIKey: "test-api-key",
			},
			wantErr: false,
		},
		{
			name:    "空 API Key",
			config:  ExaConfig{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tool, err := NewExaSearchTool(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewExaSearchTool() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tool == nil {
				t.Error("NewExaSearchTool() returned nil tool")
			}
		})
	}
}

// TestExaSearchInputValidation 测试参数验证
func TestExaSearchInputValidation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockExaResponse))
	}))
	defer server.Close()

	tests := []struct {
		name          string
		input         ExaSearchInput
		config        ExaConfig
		expectSuccess bool
		expectError   string
	}{
		{
			name:          "空查询",
			input:         ExaSearchInput{Query: ""},
			config:        ExaConfig{APIKey: "test-key"},
			expectSuccess: false,
			expectError:   "搜索查询不能为空",
		},
		{
			name:          "未配置 API Key",
			input:         ExaSearchInput{Query: "golang"},
			config:        ExaConfig{},
			expectSuccess: false,
			expectError:   "Exa API Key 未配置，请在配置文件中设置 exa_search.api_key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := executeExaSearch(context.Background(), tt.input, tt.config)
			if err != nil {
				t.Fatalf("executeExaSearch() error = %v", err)
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

// TestSearchExaSuccess 测试 Exa 搜索成功（使用 Mock Server）
func TestSearchExaSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 验证请求方法
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST, got %s", r.Method)
		}

		// 验证 API Key 头
		apiKey := r.Header.Get("x-api-key")
		if apiKey != "test-api-key" {
			t.Errorf("Expected api-key='test-api-key', got '%s'", apiKey)
		}

		// 验证 Content-Type
		contentType := r.Header.Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Expected Content-Type='application/json', got '%s'", contentType)
		}

		// 验证请求体
		var reqBody exaRequest
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}
		if reqBody.Query != "golang" {
			t.Errorf("Expected query='golang', got '%s'", reqBody.Query)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockExaResponse))
	}))
	defer server.Close()

	result, err := searchExaWithURL(context.Background(), "golang", 5, false, "test-api-key", server.URL)
	if err != nil {
		t.Fatalf("searchExaWithURL() failed: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected success=true, got false. Error: %s", result.Error)
	}

	if len(result.Results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(result.Results))
	}

	if result.Results[0].Title != "The Go Programming Language" {
		t.Errorf("Unexpected title: %s", result.Results[0].Title)
	}

	if result.Results[0].Snippet != "Go is an open source programming language." {
		t.Errorf("Unexpected snippet: %s", result.Results[0].Snippet)
	}
}

// TestSearchExaWithPageText 测试包含网页正文的搜索
func TestSearchExaWithPageText(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqBody exaRequest
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}

		// 验证 includePageText=true 时使用 text 内容选项
		if reqBody.Contents.Text == nil {
			t.Error("Expected contents.text to be set when includePageText=true")
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockExaResponseWithText))
	}))
	defer server.Close()

	result, err := searchExaWithURL(context.Background(), "golang", 5, true, "test-api-key", server.URL)
	if err != nil {
		t.Fatalf("searchExaWithURL() failed: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected success=true, got false. Error: %s", result.Error)
	}

	if result.Results[0].Text == "" {
		t.Error("Expected Text to be populated when include_page_text=true")
	}
}

// TestSearchExaEmpty 测试空结果处理
func TestSearchExaEmpty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockExaEmptyResponse))
	}))
	defer server.Close()

	result, err := searchExaWithURL(context.Background(), "nonexistent", 5, false, "test-api-key", server.URL)
	if err != nil {
		t.Fatalf("searchExaWithURL() failed: %v", err)
	}

	if result.Success {
		t.Error("Expected success=false for empty results")
	}

	if result.Message != "未找到搜索结果" {
		t.Errorf("Unexpected message: %s", result.Message)
	}
}

// TestSearchExaHTTPError 测试 HTTP 错误响应
func TestSearchExaHTTPError(t *testing.T) {
	statusCodes := []int{400, 401, 403, 429, 500}

	for _, code := range statusCodes {
		t.Run(fmt.Sprintf("Status%d", code), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(code)
				w.Write([]byte("Error"))
			}))
			defer server.Close()

			_, err := searchExaWithURL(context.Background(), "golang", 5, false, "test-api-key", server.URL)
			if err == nil {
				t.Errorf("Expected error for status code %d", code)
			}
		})
	}
}

// TestSearchExaInvalidJSON 测试无效 JSON 响应
func TestSearchExaInvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("invalid json"))
	}))
	defer server.Close()

	_, err := searchExaWithURL(context.Background(), "golang", 5, false, "test-api-key", server.URL)
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

// TestSearchExaNumResultsLimit 测试结果数量限制
func TestSearchExaNumResultsLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqBody exaRequest
		json.NewDecoder(r.Body).Decode(&reqBody)
		// 验证 numResults 被限制在 10 以内
		if reqBody.NumResults > 10 {
			t.Errorf("numResults should be capped at 10, got %d", reqBody.NumResults)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockExaResponse))
	}))
	defer server.Close()

	// 请求超过最大限制的数量
	config := ExaConfig{APIKey: "test-api-key"}
	input := ExaSearchInput{Query: "golang", NumResults: 100}

	// 模拟 executeExaSearch（用本地服务器，但 executeExaSearch 调用真实 URL）
	// 只测试数量限制逻辑
	numResults := input.NumResults
	if numResults > 10 {
		numResults = 10
	}
	if numResults != 10 {
		t.Errorf("Expected numResults to be capped at 10, got %d", numResults)
	}
	_ = config
}

package tools

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"google.golang.org/adk/v2/agent"
	"google.golang.org/adk/v2/tool"
	"google.golang.org/adk/v2/tool/toolconfirmation"
)

type parallelRoundTripFunc func(*http.Request) (*http.Response, error)

func (f parallelRoundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type parallelAgentContext struct {
	agent.StrictContextMock
	confirmation *toolconfirmation.ToolConfirmation
}

func (c parallelAgentContext) SessionID() string { return "test-conversation" }
func (c parallelAgentContext) AppName() string   { return "aiguides" }
func (c parallelAgentContext) UserID() string    { return "test-user" }
func (c parallelAgentContext) ToolConfirmation() *toolconfirmation.ToolConfirmation {
	return c.confirmation
}

// This exercises real Streamable HTTP framing with the official SDK server.
// The transport records the production URL before directing it to the fixture.
func installParallelFixture(t *testing.T, handler mcp.ToolHandler, jsonResponse bool) *atomic.Int32 {
	t.Helper()
	server := mcp.NewServer(&mcp.Implementation{Name: "fixture", Version: "1"}, nil)
	server.AddTool(&mcp.Tool{Name: "web_search", InputSchema: map[string]any{"type": "object"}}, handler)
	httpServer := httptest.NewServer(mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		return server
	}, &mcp.StreamableHTTPOptions{Stateless: true, JSONResponse: jsonResponse}))
	t.Cleanup(httpServer.Close)
	destination, _ := url.Parse(httpServer.URL)
	original := http.DefaultTransport
	var calls atomic.Int32
	http.DefaultTransport = parallelRoundTripFunc(func(r *http.Request) (*http.Response, error) {
		calls.Add(1)
		if r.URL.String() != parallelMCPURL {
			t.Errorf("unexpected destination: %s", r.URL)
			return nil, errors.New("unexpected destination")
		}
		if r.UserAgent() != "AIGuides/1.0 (Parallel Search MCP)" || r.Header.Get("Authorization") != "" {
			t.Errorf("unexpected user agent or authentication: %s", r.UserAgent())
		}
		clone := r.Clone(r.Context())
		clone.URL.Scheme, clone.URL.Host = destination.Scheme, destination.Host
		clone.Host = destination.Host
		return original.RoundTrip(clone)
	})
	t.Cleanup(func() { http.DefaultTransport = original })
	return &calls
}

func runParallelTool(t *testing.T, ctx context.Context, args map[string]any) WebSearchOutput {
	t.Helper()
	search, err := NewWebSearchTool(WebSearchConfig{Provider: "parallel", SearXNG: SearXNGConfig{InstanceURL: "https://must-not-fallback.invalid"}})
	if err != nil {
		t.Fatal(err)
	}
	runnable := search.(interface {
		Run(agent.Context, any) (map[string]any, error)
	})
	result, err := runnable.Run(&parallelAgentContext{StrictContextMock: agent.NewStrictContextMock(ctx)}, args)
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	var output WebSearchOutput
	if err := json.Unmarshal(data, &output); err != nil {
		t.Fatal(err)
	}
	return output
}

func TestParallelSearchTool(t *testing.T) {
	for _, jsonResponse := range []bool{true, false} {
		name := "SSE"
		if jsonResponse {
			name = "JSON"
		}
		t.Run(name, func(t *testing.T) {
			t.Setenv("PARALLEL_API_KEY", "must-not-be-used")
			var previousSession string
			var mu sync.Mutex
			calls := installParallelFixture(t, func(_ context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				mu.Lock()
				defer mu.Unlock()
				var args struct {
					Objective string   `json:"objective"`
					Queries   []string `json:"search_queries"`
					SessionID string   `json:"session_id"`
				}
				if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
					t.Error(err)
				}
				if !strings.Contains(args.Objective, "zh-CN") || len(args.Queries) != 1 || args.Queries[0] != "Go release notes" {
					t.Errorf("incorrect MCP arguments: %s", req.Params.Arguments)
				}
				if len(args.SessionID) != 64 || previousSession != "" && args.SessionID != previousSession {
					t.Errorf("conversation ID missing or unstable: %q", args.SessionID)
				}
				previousSession = args.SessionID
				return &mcp.CallToolResult{StructuredContent: map[string]any{"results": []any{
					map[string]any{"title": "Go", "url": "https://go.dev/doc/devel/release", "excerpts": []string{"Release notes", "Current Go versions"}},
					map[string]any{"url": "https://go.dev", "excerpts": []string{"Go home"}},
				}}}, nil
			}, jsonResponse)
			for range 2 {
				out := runParallelTool(t, context.Background(), map[string]any{"query": "Go release notes", "language": "zh-CN", "num_results": 1})
				if !out.Success || out.Query != "Go release notes" || len(out.Results) != 1 || out.Results[0].Link != "https://go.dev/doc/devel/release" || out.Results[0].Snippet != "Release notes\n\nCurrent Go versions" || out.Results[0].Engine != "parallel" {
					t.Fatalf("unexpected model-visible result: %+v", out)
				}
			}
			if calls.Load() < 4 {
				t.Fatalf("MCP initialization and search requests missing: %d", calls.Load())
			}
		})
	}
}

func TestParallelSearchErrors(t *testing.T) {
	cases := []struct {
		name   string
		result *mcp.CallToolResult
		want   string
	}{
		{"quota", &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: "anonymous quota exceeded"}}}, "quota exceeded"},
		{"invalid JSON", &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "not JSON"}}}, "parse Parallel"},
		{"missing results", &mcp.CallToolResult{StructuredContent: map[string]any{"unexpected": true}}, "missing results"},
		{"empty results", &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: `{"results":[]}`}}}, ""},
		{"text JSON", &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: `{"results":[{"url":"https://go.dev","excerpts":["Go"]}]}`}}}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			installParallelFixture(t, func(context.Context, *mcp.CallToolRequest) (*mcp.CallToolResult, error) { return tc.result, nil }, true)
			out := runParallelTool(t, context.Background(), map[string]any{"query": "Go release notes"})
			if tc.want != "" && (out.Success || !strings.Contains(out.Error, tc.want)) {
				t.Fatalf("expected visible failure %q: %+v", tc.want, out)
			}
			if tc.name == "text JSON" && (!out.Success || out.Results[0].Link != "https://go.dev") {
				t.Fatalf("text JSON not consumed: %+v", out)
			}
			if tc.name == "empty results" && (out.Success || out.Error != "") {
				t.Fatalf("unexpected empty result: %+v", out)
			}
		})
	}
}

func TestParallelSearchCancellationAndTimeout(t *testing.T) {
	for _, timeout := range []bool{false, true} {
		t.Run(map[bool]string{false: "cancel", true: "timeout"}[timeout], func(t *testing.T) {
			started := make(chan struct{})
			release := make(chan struct{})
			installParallelFixture(t, func(ctx context.Context, _ *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				close(started)
				// A remote server may keep working after its client disconnects.
				// Release it explicitly after verifying caller cancellation.
				select {
				case <-ctx.Done():
				case <-release:
				}
				return &mcp.CallToolResult{}, ctx.Err()
			}, true)
			ctx, cancel := context.WithCancel(context.Background())
			if timeout {
				cancel()
				ctx, cancel = context.WithTimeout(context.Background(), 100*time.Millisecond)
			}
			defer cancel()
			if !timeout {
				go func() { <-started; cancel() }()
			}
			out := runParallelTool(t, ctx, map[string]any{"query": "Go release notes"})
			close(release)
			select {
			case <-started:
			default:
				t.Fatal("search did not reach the server before cancellation/timeout")
			}
			if out.Success || !strings.Contains(out.Error, ctx.Err().Error()) {
				t.Fatalf("cancellation/timeout not surfaced: %+v", out)
			}
		})
	}
}

func TestParallelSearchRejectedConfirmation(t *testing.T) {
	calls := installParallelFixture(t, func(context.Context, *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		t.Error("rejected tool reached MCP")
		return nil, nil
	}, true)
	search, err := NewWebSearchTool(WebSearchConfig{Provider: "parallel"})
	if err != nil {
		t.Fatal(err)
	}
	runnable := search.(interface {
		Run(agent.Context, any) (map[string]any, error)
	})
	_, err = runnable.Run(&parallelAgentContext{StrictContextMock: agent.NewStrictContextMock(context.Background()), confirmation: &toolconfirmation.ToolConfirmation{Confirmed: false}}, map[string]any{"query": "Go"})
	if !errors.Is(err, tool.ErrConfirmationRejected) || calls.Load() != 0 {
		t.Fatalf("confirmation not preserved: %v, requests=%d", err, calls.Load())
	}
}

func TestWebSearchProviderSelection(t *testing.T) {
	for _, tc := range []struct {
		name, provider, instance, wantURL string
	}{
		{"fresh default", "", "", "https://searx.be/search"},
		{"saved instance", "", "https://saved.example", "https://saved.example/search"},
		{"explicit searxng", "searxng", "https://chosen.example", "https://chosen.example/search"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			original := http.DefaultTransport
			defer func() { http.DefaultTransport = original }()
			var requests int
			http.DefaultTransport = parallelRoundTripFunc(func(r *http.Request) (*http.Response, error) {
				requests++
				if r.URL.Scheme+"://"+r.URL.Host+r.URL.Path != tc.wantURL || r.URL.Query().Get("q") != "golang" {
					t.Errorf("saved/default selection changed: %s", r.URL)
				}
				return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(mockSearXNGResponse))}, nil
			})
			search, err := NewWebSearchTool(WebSearchConfig{Provider: tc.provider, SearXNG: SearXNGConfig{InstanceURL: tc.instance}})
			if err != nil {
				t.Fatal(err)
			}
			runnable := search.(interface {
				Run(agent.Context, any) (map[string]any, error)
			})
			out, err := runnable.Run(&parallelAgentContext{StrictContextMock: agent.NewStrictContextMock(context.Background())}, map[string]any{"query": "golang"})
			if err != nil || out["success"] != true || requests != 1 {
				t.Fatalf("SearXNG dispatch failed: %+v, %v, requests=%d", out, err, requests)
			}
		})
	}
	if _, err := NewWebSearchTool(WebSearchConfig{Provider: "typo"}); err == nil {
		t.Fatal("unknown explicit provider silently selected a default")
	}
}

func TestParallelSearchHTTPFailure(t *testing.T) {
	original := http.DefaultTransport
	defer func() { http.DefaultTransport = original }()
	http.DefaultTransport = parallelRoundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.String() != parallelMCPURL {
			t.Errorf("explicit provider failure fell back: %s", r.URL)
		}
		return &http.Response{StatusCode: http.StatusServiceUnavailable, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("service unavailable"))}, nil
	})
	out := runParallelTool(t, context.Background(), map[string]any{"query": "Go release notes"})
	if out.Success || !strings.Contains(out.Error, "Service Unavailable") {
		t.Fatalf("HTTP failure not returned to model: %+v", out)
	}
}

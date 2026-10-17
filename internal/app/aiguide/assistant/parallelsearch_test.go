package assistant

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync/atomic"
	"testing"

	"aiguide/internal/pkg/tools"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"google.golang.org/genai"
)

type parallelDispatcherTransport func(*http.Request) (*http.Response, error)

func (f parallelDispatcherTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func parallelDispatcherConfig(t *testing.T) *Config {
	t.Helper()
	return &Config{
		DB: setupTestDB(t), FileStore: mustTestFileStore(t), PDFWorkDir: t.TempDir(),
		MockImageGen: true, MockVideoGen: true,
		WebSearchConfig: tools.WebSearchConfig{Provider: "parallel"},
		ExaConfig:       tools.ExaConfig{APIKey: "saved-exa-choice"},
	}
}

func dispatchParallelSearch(t *testing.T, config *Config) *genai.FunctionResponse {
	t.Helper()
	allTools, err := createAssistantTools(config)
	if err != nil {
		t.Fatal(err)
	}
	partition := partitionTools(allTools)
	registry := buildLiveToolRegistry(partition.Web)
	if registry["web_search"] == nil || registry["web_fetch"] == nil || registry["exa_search"] == nil {
		t.Fatal("web tools were not preserved in the assistant registry")
	}
	if buildLiveToolRegistry(partition.Research)["web_search"] == nil {
		t.Fatal("research agent did not receive web_search")
	}
	return executeLiveTool(context.Background(), registry, &genai.FunctionCall{
		ID: "search-call", Name: "web_search", Args: map[string]any{"query": "Go release notes", "num_results": 1},
	}, nil, "")
}

func TestParallelSearchAssistantDispatcher(t *testing.T) {
	server := mcp.NewServer(&mcp.Implementation{Name: "fixture", Version: "1"}, nil)
	server.AddTool(&mcp.Tool{Name: "web_search", InputSchema: map[string]any{"type": "object"}}, func(context.Context, *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return &mcp.CallToolResult{StructuredContent: map[string]any{"results": []any{
			map[string]any{"url": "https://go.dev/doc/devel/release", "title": "Go releases", "excerpts": []string{"Release notes"}},
		}}}, nil
	})
	httpServer := httptest.NewServer(mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return server }, &mcp.StreamableHTTPOptions{Stateless: true, JSONResponse: true}))
	defer httpServer.Close()
	destination, _ := url.Parse(httpServer.URL)
	original := http.DefaultTransport
	var requests atomic.Int32
	http.DefaultTransport = parallelDispatcherTransport(func(r *http.Request) (*http.Response, error) {
		requests.Add(1)
		if r.URL.String() != "https://search.parallel.ai/mcp" || r.UserAgent() != "AIGuides/1.0 (Parallel Search MCP)" || r.Header.Get("Authorization") != "" {
			return nil, fmt.Errorf("unexpected MCP destination, identity or authentication")
		}
		clone := r.Clone(r.Context())
		clone.URL.Scheme, clone.URL.Host, clone.Host = destination.Scheme, destination.Host, destination.Host
		return original.RoundTrip(clone)
	})
	defer func() { http.DefaultTransport = original }()
	response := dispatchParallelSearch(t, parallelDispatcherConfig(t))
	data, err := json.Marshal(response.Response)
	if err != nil {
		t.Fatal(err)
	}
	var output tools.WebSearchOutput
	if err := json.Unmarshal(data, &output); err != nil {
		t.Fatal(err)
	}
	if !output.Success || len(output.Results) != 1 || output.Results[0].Link != "https://go.dev/doc/devel/release" || output.Results[0].Snippet != "Release notes" || response.ID != "search-call" {
		t.Fatalf("tool result did not reach the model response: %s", data)
	}
	if requests.Load() < 2 {
		t.Fatal("MCP handshake/search requests missing")
	}

	// An unregistered tool cannot bypass the caller's tool allowlist.
	beforeDenied := requests.Load()
	denied := executeLiveTool(context.Background(), liveToolRegistry{}, &genai.FunctionCall{Name: "web_search"}, nil, "")
	if !strings.Contains(fmt.Sprint(denied.Response["error"]), "unknown tool") || requests.Load() != beforeDenied {
		t.Fatal("missing tool allowlist was bypassed")
	}
}

// Opt-in live check through the normal assistant tool factory and dispatcher.
func TestParallelSearchLive(t *testing.T) {
	if os.Getenv("AIGUIDES_PARALLEL_LIVE") != "1" {
		t.Skip("set AIGUIDES_PARALLEL_LIVE=1 to call the anonymous Parallel endpoint")
	}
	t.Setenv("PARALLEL_API_KEY", "")
	original := http.DefaultTransport
	var requests atomic.Int32
	http.DefaultTransport = parallelDispatcherTransport(func(r *http.Request) (*http.Response, error) {
		requests.Add(1)
		if r.URL.String() != "https://search.parallel.ai/mcp" || r.UserAgent() != "AIGuides/1.0 (Parallel Search MCP)" || r.Header.Get("Authorization") != "" {
			return nil, fmt.Errorf("unexpected MCP destination, identity or authentication")
		}
		return original.RoundTrip(r)
	})
	defer func() { http.DefaultTransport = original }()
	response := dispatchParallelSearch(t, parallelDispatcherConfig(t))
	data, err := json.Marshal(response.Response)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("anonymous MCP result: %s", data)
	if requests.Load() < 2 {
		t.Fatal("MCP handshake/search requests missing")
	}
	if response.Response["success"] != true {
		t.Fatalf("live search did not succeed: %s", data)
	}
}

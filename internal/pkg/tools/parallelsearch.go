package tools

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"google.golang.org/adk/v2/agent"
)

const parallelMCPURL = "https://search.parallel.ai/mcp"

// searchParallel keeps the existing web_search response contract while using
// the anonymous Parallel Search MCP endpoint over Streamable HTTP.
func searchParallel(ctx context.Context, query string, numResults int, language string) (*WebSearchOutput, error) {
	var sessionID string
	if ac, ok := ctx.(agent.Context); ok && ac.SessionID() != "" {
		sessionID = fmt.Sprintf("%x", sha256.Sum256([]byte(ac.AppName()+"\x00"+ac.UserID()+"\x00"+ac.SessionID())))
	}
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	client := mcp.NewClient(&mcp.Implementation{Name: "aiguides", Version: "1.0"}, nil)
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{
		Endpoint: parallelMCPURL,
		HTTPClient: &http.Client{
			Transport: parallelUserAgentTransport{base: http.DefaultTransport},
			Timeout:   time.Minute,
		},
	}, nil)
	if err != nil {
		slog.Error("failed to connect to parallel search", "err", err)
		return nil, fmt.Errorf("connect to Parallel Search MCP: %w", err)
	}
	defer session.Close()

	args := map[string]any{
		"objective":      query + "\nPreferred source language: " + language,
		"search_queries": []string{query},
	}
	// Hash runtime conversation identity rather than sending user/session IDs.
	// Some callers (including the voice bridge) do not provide a session ID.
	if sessionID != "" {
		args["session_id"] = sessionID
	}
	result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "web_search", Arguments: args})
	if err != nil {
		slog.Error("failed to call parallel search", "err", err)
		return nil, fmt.Errorf("Parallel search: %w", err)
	}
	if result.IsError {
		var messages []string
		for _, content := range result.Content {
			if text, ok := content.(*mcp.TextContent); ok {
				messages = append(messages, text.Text)
			}
		}
		err := fmt.Errorf("Parallel search failed: %s", strings.Join(messages, "\n"))
		slog.Error("parallel search returned an error", "err", err)
		return nil, err
	}

	// MCP specifies structuredContent; text JSON is also supported for servers
	// returning the backwards-compatible text representation of that response.
	var data []byte
	if result.StructuredContent != nil {
		data, err = json.Marshal(result.StructuredContent)
	} else {
		for _, content := range result.Content {
			if text, ok := content.(*mcp.TextContent); ok {
				data = []byte(text.Text)
				break
			}
		}
	}
	var response struct {
		Results *[]struct {
			URL      string   `json:"url"`
			Title    string   `json:"title"`
			Excerpts []string `json:"excerpts"`
		} `json:"results"`
	}
	if err == nil {
		err = json.Unmarshal(data, &response)
	}
	if err == nil && response.Results == nil {
		err = fmt.Errorf("missing results field")
	}
	if err != nil {
		slog.Error("failed to parse parallel search response", "err", err)
		return nil, fmt.Errorf("parse Parallel search response: %w", err)
	}

	items := make([]WebSearchResultItem, 0, numResults)
	for _, item := range *response.Results {
		if len(items) >= numResults {
			break
		}
		items = append(items, WebSearchResultItem{
			Title: item.Title, Link: item.URL,
			Snippet: strings.Join(item.Excerpts, "\n\n"), Engine: "parallel",
		})
	}
	return &WebSearchOutput{
		Success: len(items) > 0, Query: query, Results: items,
		Message: fmt.Sprintf("找到 %d 个搜索结果", len(items)),
	}, nil
}

type parallelUserAgentTransport struct {
	base http.RoundTripper
}

func (t parallelUserAgentTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.Header.Set("User-Agent", "AIGuides/1.0 (Parallel Search MCP)")
	return t.base.RoundTrip(req)
}

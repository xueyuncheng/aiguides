package tools

import (
	"context"
	"net/http"
	"testing"
	"time"
)

func TestNewWebFetchTool(t *testing.T) {
	fetchTool, err := NewWebFetchTool()
	if err != nil {
		t.Fatalf("NewWebFetchTool returned error: %v", err)
	}

	if fetchTool.Name() != "fetch_webpage" {
		t.Errorf("Expected name 'fetch_webpage', got '%s'", fetchTool.Name())
	}

	if fetchTool.IsLongRunning() {
		t.Error("Expected IsLongRunning to be false")
	}
}

func TestFetchWebPage(t *testing.T) {
	ctx := context.Background()
	client := &http.Client{Timeout: 10 * time.Second}
	input := WebFetchInput{URL: "https://example.com"}

	output := fetchWebPage(ctx, client, input)
	if !output.Success {
		t.Fatalf("Expected success, got error: %s", output.Error)
	}

	if output.Content == "" {
		t.Fatal("Expected content, got empty string")
	}

	t.Logf("Successfully fetched %d bytes from %s", len(output.Content), input.URL)
}

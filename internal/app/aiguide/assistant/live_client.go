package assistant

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"google.golang.org/genai"
)

func newLiveClient(ctx context.Context, apiKey, baseURL string, httpClient *http.Client) (*genai.Client, error) {
	cfg := &genai.ClientConfig{
		APIKey:     apiKey,
		HTTPClient: httpClient,
		HTTPOptions: genai.HTTPOptions{
			APIVersion: "v1alpha",
		},
	}
	if baseURL != "" {
		cfg.HTTPOptions.BaseURL = baseURL
	}

	client, err := genai.NewClient(ctx, cfg)
	if err != nil {
		slog.Error("genai.NewClient (live) error", "err", err)
		return nil, fmt.Errorf("genai.NewClient (live): %w", err)
	}
	return client, nil
}

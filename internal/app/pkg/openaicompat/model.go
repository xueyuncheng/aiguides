package openaicompat

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"iter"
	"net/http"

	"google.golang.org/adk/model"
	"google.golang.org/genai"
)

// Model implements the model.LLM interface for OpenAI-compatible APIs (like X.AI)
type Model struct {
	apiKey     string
	baseURL    string
	modelName  string
	httpClient *http.Client
}

type Config struct {
	APIKey     string
	BaseURL    string
	ModelName  string
	HTTPClient *http.Client
}

func NewModel(cfg Config) (*Model, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("API key is required")
	}
	if cfg.BaseURL == "" {
		return nil, fmt.Errorf("base URL is required")
	}
	if cfg.ModelName == "" {
		return nil, fmt.Errorf("model name is required")
	}

	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	return &Model{
		apiKey:     cfg.APIKey,
		baseURL:    cfg.BaseURL,
		modelName:  cfg.ModelName,
		httpClient: httpClient,
	}, nil
}

// ChatRequest represents the OpenAI chat completion request
type ChatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Stream   bool      `json:"stream"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatResponse represents the OpenAI chat completion response
type ChatResponse struct {
	Choices []Choice `json:"choices"`
}

type Choice struct {
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

// GenerateContent implements model.LLM.GenerateContent
func (m *Model) GenerateContent(ctx context.Context, req *model.LLMRequest, stream bool) iter.Seq2[*model.LLMResponse, error] {
	return func(yield func(*model.LLMResponse, error) bool) {
		// Convert genai.Content to OpenAI messages
		var messages []Message
		for _, content := range req.Contents {
			if len(content.Parts) > 0 {
				// Extract text from parts
				var text string
				for _, part := range content.Parts {
					if part.Text != "" {
						text += part.Text
					}
				}
				if text != "" {
					messages = append(messages, Message{
						Role:    content.Role,
						Content: text,
					})
				}
			}
		}

		chatReq := ChatRequest{
			Model:    m.modelName,
			Messages: messages,
			Stream:   false,
		}

		reqBody, err := json.Marshal(chatReq)
		if err != nil {
			yield(nil, fmt.Errorf("failed to marshal request: %w", err))
			return
		}

		httpReq, err := http.NewRequestWithContext(ctx, "POST", m.baseURL+"/chat/completions", bytes.NewReader(reqBody))
		if err != nil {
			yield(nil, fmt.Errorf("failed to create request: %w", err))
			return
		}

		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Authorization", "Bearer "+m.apiKey)

		resp, err := m.httpClient.Do(httpReq)
		if err != nil {
			yield(nil, fmt.Errorf("failed to send request: %w", err))
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			yield(nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body)))
			return
		}

		var chatResp ChatResponse
		if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
			yield(nil, fmt.Errorf("failed to decode response: %w", err))
			return
		}

		if len(chatResp.Choices) == 0 {
			yield(nil, fmt.Errorf("no choices in response"))
			return
		}

		// Convert OpenAI response to genai.Content
		response := &model.LLMResponse{
			Content: &genai.Content{
				Parts: []*genai.Part{
					genai.NewPartFromText(chatResp.Choices[0].Message.Content),
				},
				Role: "model",
			},
			TurnComplete: true,
		}

		yield(response, nil)
	}
}

// Name returns the model name
func (m *Model) Name() string {
	return m.modelName
}

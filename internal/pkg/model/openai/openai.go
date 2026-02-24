// Package openai implements the model.LLM interface for OpenAI-compatible APIs.
// It supports any provider that follows the OpenAI chat completions API format,
// including OpenAI, DeepSeek, Ollama, and other compatible services.
package openai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"iter"
	"log/slog"
	"net/http"
	"strings"

	"google.golang.org/adk/model"
	"google.golang.org/genai"
)

const defaultBaseURL = "https://api.openai.com/v1"

// openaiModel implements model.LLM for OpenAI-compatible APIs.
type openaiModel struct {
	apiKey     string
	baseURL    string
	modelName  string
	httpClient *http.Client
}

// NewModel creates a new OpenAI-compatible model.LLM.
// apiKey is the API key (Bearer token) for the provider.
// baseURL is the API base URL (e.g. "https://api.openai.com/v1" or "http://localhost:11434/v1" for Ollama).
// modelName is the model identifier (e.g. "gpt-4o", "deepseek-chat").
func NewModel(apiKey, baseURL, modelName string, httpClient *http.Client) model.LLM {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	return &openaiModel{
		apiKey:     apiKey,
		baseURL:    strings.TrimRight(baseURL, "/"),
		modelName:  modelName,
		httpClient: httpClient,
	}
}

// Name returns the model name.
func (m *openaiModel) Name() string {
	return m.modelName
}

// GenerateContent converts the ADK request to OpenAI format, calls the API,
// and converts the response back to ADK format.
func (m *openaiModel) GenerateContent(ctx context.Context, req *model.LLMRequest, stream bool) iter.Seq2[*model.LLMResponse, error] {
	messages, err := buildMessages(req)
	if err != nil {
		return func(yield func(*model.LLMResponse, error) bool) {
			yield(nil, fmt.Errorf("failed to build messages: %w", err))
		}
	}

	tools := buildTools(req)

	chatReq := &chatCompletionRequest{
		Model:    m.modelName,
		Messages: messages,
		Tools:    tools,
		Stream:   stream,
	}

	body, err := json.Marshal(chatReq)
	if err != nil {
		return func(yield func(*model.LLMResponse, error) bool) {
			yield(nil, fmt.Errorf("failed to marshal request: %w", err))
		}
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, m.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return func(yield func(*model.LLMResponse, error) bool) {
			yield(nil, fmt.Errorf("failed to create HTTP request: %w", err))
		}
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+m.apiKey)

	if stream {
		return m.generateStream(httpReq)
	}
	return m.generateSync(httpReq)
}

// generateSync calls the model synchronously and returns a single response.
func (m *openaiModel) generateSync(req *http.Request) iter.Seq2[*model.LLMResponse, error] {
	return func(yield func(*model.LLMResponse, error) bool) {
		resp, err := m.httpClient.Do(req)
		if err != nil {
			slog.Error("openai http request error", "err", err)
			yield(nil, fmt.Errorf("HTTP request failed: %w", err))
			return
		}
		defer resp.Body.Close()

		data, err := io.ReadAll(resp.Body)
		if err != nil {
			slog.Error("openai read response body error", "err", err)
			yield(nil, fmt.Errorf("failed to read response body: %w", err))
			return
		}

		if resp.StatusCode != http.StatusOK {
			slog.Error("openai API error", "status", resp.StatusCode, "body", string(data))
			yield(nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(data)))
			return
		}

		var completion chatCompletionResponse
		if err := json.Unmarshal(data, &completion); err != nil {
			slog.Error("openai unmarshal response error", "err", err)
			yield(nil, fmt.Errorf("failed to unmarshal response: %w", err))
			return
		}

		llmResp, err := convertResponse(&completion)
		if err != nil {
			yield(nil, err)
			return
		}
		llmResp.TurnComplete = true
		yield(llmResp, nil)
	}
}

// generateStream calls the model with streaming and returns chunked responses.
func (m *openaiModel) generateStream(req *http.Request) iter.Seq2[*model.LLMResponse, error] {
	return func(yield func(*model.LLMResponse, error) bool) {
		resp, err := m.httpClient.Do(req)
		if err != nil {
			slog.Error("openai http request error", "err", err)
			yield(nil, fmt.Errorf("HTTP request failed: %w", err))
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			data, _ := io.ReadAll(resp.Body)
			slog.Error("openai API error", "status", resp.StatusCode, "body", string(data))
			yield(nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(data)))
			return
		}

		// Accumulate streaming tool call chunks.
		// OpenAI sends tool_calls incrementally across multiple chunks,
		// so we need to merge them before yielding a final response.
		type toolCallAccumulator struct {
			id        string
			name      string
			arguments strings.Builder
		}
		toolCalls := make(map[int]*toolCallAccumulator)
		var textBuilder strings.Builder

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			payload := strings.TrimPrefix(line, "data: ")
			if payload == "[DONE]" {
				break
			}

			var chunk streamChunk
			if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
				slog.Error("openai unmarshal stream chunk error", "err", err, "payload", payload)
				yield(nil, fmt.Errorf("failed to unmarshal stream chunk: %w", err))
				return
			}

			if len(chunk.Choices) == 0 {
				continue
			}
			delta := chunk.Choices[0].Delta

			// Accumulate text content.
			if delta.Content != nil {
				if s, ok := delta.Content.(string); ok && s != "" {
					textBuilder.WriteString(s)
					// Yield partial text response.
					yield(&model.LLMResponse{
						Content: genai.NewContentFromText(s, "model"),
						Partial: true,
					}, nil)
				}
			}

			// Accumulate tool calls.
			for _, tc := range delta.ToolCalls {
				acc, exists := toolCalls[tc.Index]
				if !exists {
					acc = &toolCallAccumulator{}
					toolCalls[tc.Index] = acc
				}
				if tc.ID != "" {
					acc.id = tc.ID
				}
				if tc.Function.Name != "" {
					acc.name = tc.Function.Name
				}
				acc.arguments.WriteString(tc.Function.Arguments)
			}
		}

		if err := scanner.Err(); err != nil {
			slog.Error("openai stream scanner error", "err", err)
			yield(nil, fmt.Errorf("stream read error: %w", err))
			return
		}

		// Yield final response with all tool calls assembled.
		if len(toolCalls) > 0 {
			parts := make([]*genai.Part, 0, len(toolCalls))
			for i := 0; i < len(toolCalls); i++ {
				acc, ok := toolCalls[i]
				if !ok {
					continue
				}
				var args map[string]any
				if err := json.Unmarshal([]byte(acc.arguments.String()), &args); err != nil {
					slog.Error("openai parse tool call args error", "err", err)
					yield(nil, fmt.Errorf("failed to parse tool call arguments: %w", err))
					return
				}
				parts = append(parts, &genai.Part{
					FunctionCall: &genai.FunctionCall{
						ID:   acc.id,
						Name: acc.name,
						Args: args,
					},
				})
			}
			yield(&model.LLMResponse{
				Content:      &genai.Content{Role: "model", Parts: parts},
				TurnComplete: true,
			}, nil)
			return
		}

		// Yield final text response.
		if textBuilder.Len() > 0 {
			yield(&model.LLMResponse{
				Content:      genai.NewContentFromText(textBuilder.String(), "model"),
				TurnComplete: true,
			}, nil)
		}
	}
}

// buildMessages converts ADK genai.Content messages to OpenAI chat messages.
// It handles text, multimodal (images), function calls, and function responses.
func buildMessages(req *model.LLMRequest) ([]chatMessage, error) {
	var messages []chatMessage

	// Add system instruction if present.
	if req.Config != nil && req.Config.SystemInstruction != nil {
		text := extractText(req.Config.SystemInstruction.Parts)
		if text != "" {
			messages = append(messages, chatMessage{
				Role:    "system",
				Content: text,
			})
		}
	}

	for _, content := range req.Contents {
		if content == nil {
			continue
		}
		msgs, err := contentToMessages(content)
		if err != nil {
			return nil, err
		}
		messages = append(messages, msgs...)
	}

	return messages, nil
}

// contentToMessages converts a single genai.Content to one or more OpenAI chat messages.
// A user-role content with FunctionResponse parts becomes "tool" role messages.
// A model-role content with FunctionCall parts becomes an "assistant" message with tool_calls.
func contentToMessages(content *genai.Content) ([]chatMessage, error) {
	if content == nil || len(content.Parts) == 0 {
		return nil, nil
	}

	role := content.Role
	// "model" role maps to "assistant" in OpenAI.
	if role == "model" {
		role = "assistant"
	}

	// Separate parts by type.
	var textParts []string
	var funcCalls []*genai.FunctionCall
	var funcResponses []*genai.FunctionResponse
	var imageParts []*genai.Blob

	for _, part := range content.Parts {
		if part == nil {
			continue
		}
		switch {
		case part.FunctionCall != nil:
			funcCalls = append(funcCalls, part.FunctionCall)
		case part.FunctionResponse != nil:
			funcResponses = append(funcResponses, part.FunctionResponse)
		case part.InlineData != nil:
			imageParts = append(imageParts, part.InlineData)
		case part.Text != "" && !part.Thought:
			// Skip thought/reasoning parts: they are Gemini-specific internal
			// chain-of-thought tokens and have no equivalent in OpenAI's API.
			textParts = append(textParts, part.Text)
		}
	}

	// Handle function responses → "tool" role messages (one per response).
	if len(funcResponses) > 0 {
		var msgs []chatMessage
		for _, fr := range funcResponses {
			responseJSON, err := json.Marshal(fr.Response)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal function response: %w", err)
			}
			msgs = append(msgs, chatMessage{
				Role:       "tool",
				Content:    string(responseJSON),
				ToolCallID: fr.ID,
				Name:       fr.Name,
			})
		}
		return msgs, nil
	}

	// Handle function calls in assistant (model) messages.
	if len(funcCalls) > 0 {
		toolCalls := make([]toolCall, 0, len(funcCalls))
		for _, fc := range funcCalls {
			argsJSON, err := json.Marshal(fc.Args)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal function call args: %w", err)
			}
			toolCalls = append(toolCalls, toolCall{
				ID:   fc.ID,
				Type: "function",
				Function: functionCallContent{
					Name:      fc.Name,
					Arguments: string(argsJSON),
				},
			})
		}
		msg := chatMessage{
			Role:      "assistant",
			ToolCalls: toolCalls,
		}
		// Include any text alongside function calls.
		if len(textParts) > 0 {
			msg.Content = strings.Join(textParts, "\n")
		}
		return []chatMessage{msg}, nil
	}

	// Build content: text only, or multimodal (text + images).
	if len(imageParts) > 0 {
		var contentParts []contentPart
		for _, img := range imageParts {
			dataURI := fmt.Sprintf("data:%s;base64,%s",
				img.MIMEType,
				base64.StdEncoding.EncodeToString(img.Data),
			)
			contentParts = append(contentParts, contentPart{
				Type: "image_url",
				ImageURL: &imageURL{
					URL: dataURI,
				},
			})
		}
		for _, text := range textParts {
			contentParts = append(contentParts, contentPart{
				Type: "text",
				Text: text,
			})
		}
		return []chatMessage{{Role: role, Content: contentParts}}, nil
	}

	// Plain text message.
	text := strings.Join(textParts, "\n")
	if text == "" {
		return nil, nil
	}
	return []chatMessage{{Role: role, Content: text}}, nil
}

// buildTools converts ADK genai.Tool definitions to OpenAI tool format.
func buildTools(req *model.LLMRequest) []openAITool {
	if req.Config == nil || len(req.Config.Tools) == 0 {
		return nil
	}
	var tools []openAITool
	for _, t := range req.Config.Tools {
		if t == nil {
			continue
		}
		for _, decl := range t.FunctionDeclarations {
			if decl == nil {
				continue
			}
			tools = append(tools, openAITool{
				Type: "function",
				Function: openAIFunction{
					Name:        decl.Name,
					Description: decl.Description,
					Parameters:  schemaToMap(decl.Parameters),
				},
			})
		}
	}
	return tools
}

// schemaToMap converts a genai.Schema to a map[string]any representing OpenAI JSON Schema.
// OpenAI uses lowercase type names while genai uses uppercase (STRING → string).
func schemaToMap(s *genai.Schema) map[string]any {
	if s == nil {
		return nil
	}
	result := make(map[string]any)

	if s.Type != "" {
		result["type"] = strings.ToLower(string(s.Type))
	}
	if s.Description != "" {
		result["description"] = s.Description
	}
	if len(s.Enum) > 0 {
		result["enum"] = s.Enum
	}
	if s.Items != nil {
		result["items"] = schemaToMap(s.Items)
	}
	if len(s.Properties) > 0 {
		props := make(map[string]any, len(s.Properties))
		for k, v := range s.Properties {
			props[k] = schemaToMap(v)
		}
		result["properties"] = props
	}
	if len(s.Required) > 0 {
		result["required"] = s.Required
	}
	if len(s.AnyOf) > 0 {
		anyOf := make([]any, len(s.AnyOf))
		for i, v := range s.AnyOf {
			anyOf[i] = schemaToMap(v)
		}
		result["anyOf"] = anyOf
	}
	return result
}

// convertResponse converts an OpenAI chat completion response to model.LLMResponse.
func convertResponse(resp *chatCompletionResponse) (*model.LLMResponse, error) {
	if len(resp.Choices) == 0 {
		return &model.LLMResponse{
			ErrorCode:    "EMPTY_RESPONSE",
			ErrorMessage: "no choices in response",
		}, nil
	}

	choice := resp.Choices[0]
	msg := choice.Message

	// Handle tool calls.
	if len(msg.ToolCalls) > 0 {
		parts := make([]*genai.Part, 0, len(msg.ToolCalls))
		for _, tc := range msg.ToolCalls {
			var args map[string]any
			if tc.Function.Arguments != "" {
				if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
					return nil, fmt.Errorf("failed to parse tool call arguments for %q: %w", tc.Function.Name, err)
				}
			}
			parts = append(parts, &genai.Part{
				FunctionCall: &genai.FunctionCall{
					ID:   tc.ID,
					Name: tc.Function.Name,
					Args: args,
				},
			})
		}
		return &model.LLMResponse{
			Content: &genai.Content{Role: "model", Parts: parts},
		}, nil
	}

	// Handle text content.
	var text string
	switch v := msg.Content.(type) {
	case string:
		text = v
	case nil:
		// No content (e.g. when only tool calls are present).
	default:
		// Unexpected content type; marshal to string as fallback.
		b, _ := json.Marshal(v)
		text = string(b)
	}

	if text == "" {
		// Use the raw OpenAI finish reason as the error code rather than
		// attempting to cast it to genai.FinishReason (different value spaces).
		return &model.LLMResponse{
			ErrorCode:    strings.ToUpper(choice.FinishReason),
			ErrorMessage: "empty response content",
		}, nil
	}

	return &model.LLMResponse{
		Content: genai.NewContentFromText(text, "model"),
	}, nil
}

// extractText concatenates all text parts from a slice of genai.Part.
func extractText(parts []*genai.Part) string {
	var sb strings.Builder
	for _, p := range parts {
		if p != nil && p.Text != "" {
			sb.WriteString(p.Text)
		}
	}
	return sb.String()
}

// --- OpenAI API request/response types ---

type chatCompletionRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
	Tools    []openAITool  `json:"tools,omitempty"`
	Stream   bool          `json:"stream,omitempty"`
}

// chatMessage represents a single message in an OpenAI chat conversation.
// Content is `any` to support both string (text-only) and []contentPart (multimodal).
type chatMessage struct {
	Role       string     `json:"role"`
	Content    any        `json:"content,omitempty"`
	ToolCalls  []toolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
	Name       string     `json:"name,omitempty"`
}

type contentPart struct {
	Type     string    `json:"type"`
	Text     string    `json:"text,omitempty"`
	ImageURL *imageURL `json:"image_url,omitempty"`
}

type imageURL struct {
	URL string `json:"url"`
}

type toolCall struct {
	Index    int                 `json:"index,omitempty"`
	ID       string              `json:"id,omitempty"`
	Type     string              `json:"type,omitempty"`
	Function functionCallContent `json:"function"`
}

type functionCallContent struct {
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
}

type openAITool struct {
	Type     string         `json:"type"`
	Function openAIFunction `json:"function"`
}

type openAIFunction struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Parameters  map[string]any `json:"parameters,omitempty"`
}

type chatCompletionResponse struct {
	ID      string   `json:"id"`
	Choices []choice `json:"choices"`
}

type choice struct {
	Index        int         `json:"index"`
	Message      chatMessage `json:"message"`
	FinishReason string      `json:"finish_reason,omitempty"`
}

// streamChunk is a single SSE chunk from the OpenAI streaming API.
type streamChunk struct {
	ID      string         `json:"id"`
	Choices []streamChoice `json:"choices"`
}

type streamChoice struct {
	Index        int         `json:"index"`
	Delta        streamDelta `json:"delta"`
	FinishReason *string     `json:"finish_reason"`
}

// streamDelta holds incremental content from a streaming response.
// Content is `any` because it can be a string or null in the JSON.
type streamDelta struct {
	Role      string     `json:"role,omitempty"`
	Content   any        `json:"content,omitempty"`
	ToolCalls []toolCall `json:"tool_calls,omitempty"`
}

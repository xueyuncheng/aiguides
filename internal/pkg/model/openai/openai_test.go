package openai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"google.golang.org/adk/model"
	"google.golang.org/genai"
)

// TestNewModel verifies NewModel sets fields correctly.
func TestNewModel(t *testing.T) {
	m := NewModel("key", "https://api.example.com/v1", "gpt-4o", nil)
	if m.Name() != "gpt-4o" {
		t.Errorf("expected model name gpt-4o, got %s", m.Name())
	}
}

// TestNewModelDefaults verifies empty baseURL uses the default.
func TestNewModelDefaults(t *testing.T) {
	m := NewModel("key", "", "gpt-4o", nil).(*openaiModel)
	if m.baseURL != defaultBaseURL {
		t.Errorf("expected default base URL %s, got %s", defaultBaseURL, m.baseURL)
	}
}

// TestSchemaToMap verifies genai.Schema → OpenAI JSON Schema conversion.
func TestSchemaToMap(t *testing.T) {
	s := &genai.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*genai.Schema{
			"name": {Type: genai.TypeString, Description: "The name"},
			"age":  {Type: genai.TypeInteger},
		},
		Required: []string{"name"},
	}
	result := schemaToMap(s)

	if result["type"] != "object" {
		t.Errorf("expected type object, got %v", result["type"])
	}
	props, ok := result["properties"].(map[string]any)
	if !ok {
		t.Fatal("expected properties to be map[string]any")
	}
	nameProp, ok := props["name"].(map[string]any)
	if !ok {
		t.Fatal("expected name property to be map")
	}
	if nameProp["type"] != "string" {
		t.Errorf("expected name type string, got %v", nameProp["type"])
	}
	if nameProp["description"] != "The name" {
		t.Errorf("expected description, got %v", nameProp["description"])
	}
	required, ok := result["required"].([]string)
	if !ok || len(required) != 1 || required[0] != "name" {
		t.Errorf("unexpected required: %v", result["required"])
	}
}

// TestSchemaToMapNil verifies nil schema returns nil.
func TestSchemaToMapNil(t *testing.T) {
	if schemaToMap(nil) != nil {
		t.Error("expected nil for nil schema")
	}
}

// TestBuildMessagesSystemInstruction verifies system instruction is prepended.
func TestBuildMessagesSystemInstruction(t *testing.T) {
	req := &model.LLMRequest{
		Contents: []*genai.Content{
			genai.NewContentFromText("hello", "user"),
		},
		Config: &genai.GenerateContentConfig{
			SystemInstruction: genai.NewContentFromText("You are helpful.", genai.RoleUser),
		},
	}
	msgs, err := buildMessages(req)
	if err != nil {
		t.Fatalf("buildMessages() error = %v", err)
	}
	if len(msgs) < 2 {
		t.Fatalf("expected at least 2 messages, got %d", len(msgs))
	}
	if msgs[0].Role != "system" {
		t.Errorf("expected first message role system, got %s", msgs[0].Role)
	}
	if msgs[0].Content != "You are helpful." {
		t.Errorf("unexpected system content: %v", msgs[0].Content)
	}
	if msgs[1].Role != "user" {
		t.Errorf("expected second message role user, got %s", msgs[1].Role)
	}
}

// TestBuildMessagesModelRole verifies model role maps to assistant.
func TestBuildMessagesModelRole(t *testing.T) {
	req := &model.LLMRequest{
		Contents: []*genai.Content{
			genai.NewContentFromText("hi", "user"),
			genai.NewContentFromText("hello", "model"),
		},
	}
	msgs, err := buildMessages(req)
	if err != nil {
		t.Fatalf("buildMessages() error = %v", err)
	}
	if msgs[1].Role != "assistant" {
		t.Errorf("expected assistant, got %s", msgs[1].Role)
	}
}

// TestBuildMessagesFunctionCall verifies function calls become tool_calls.
func TestBuildMessagesFunctionCall(t *testing.T) {
	req := &model.LLMRequest{
		Contents: []*genai.Content{
			{
				Role: "model",
				Parts: []*genai.Part{
					{
						FunctionCall: &genai.FunctionCall{
							ID:   "call_1",
							Name: "my_tool",
							Args: map[string]any{"key": "value"},
						},
					},
				},
			},
		},
	}
	msgs, err := buildMessages(req)
	if err != nil {
		t.Fatalf("buildMessages() error = %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	if msgs[0].Role != "assistant" {
		t.Errorf("expected assistant, got %s", msgs[0].Role)
	}
	if len(msgs[0].ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(msgs[0].ToolCalls))
	}
	tc := msgs[0].ToolCalls[0]
	if tc.ID != "call_1" {
		t.Errorf("expected tool call ID call_1, got %s", tc.ID)
	}
	if tc.Function.Name != "my_tool" {
		t.Errorf("expected function name my_tool, got %s", tc.Function.Name)
	}
}

// TestBuildMessagesFunctionResponse verifies function responses become tool messages.
func TestBuildMessagesFunctionResponse(t *testing.T) {
	req := &model.LLMRequest{
		Contents: []*genai.Content{
			{
				Role: "user",
				Parts: []*genai.Part{
					{
						FunctionResponse: &genai.FunctionResponse{
							ID:       "call_1",
							Name:     "my_tool",
							Response: map[string]any{"result": "ok"},
						},
					},
				},
			},
		},
	}
	msgs, err := buildMessages(req)
	if err != nil {
		t.Fatalf("buildMessages() error = %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	if msgs[0].Role != "tool" {
		t.Errorf("expected tool role, got %s", msgs[0].Role)
	}
	if msgs[0].ToolCallID != "call_1" {
		t.Errorf("expected ToolCallID call_1, got %s", msgs[0].ToolCallID)
	}
}

// TestBuildTools verifies function declarations become OpenAI tool definitions.
func TestBuildTools(t *testing.T) {
	req := &model.LLMRequest{
		Config: &genai.GenerateContentConfig{
			Tools: []*genai.Tool{
				{
					FunctionDeclarations: []*genai.FunctionDeclaration{
						{
							Name:        "search",
							Description: "Search the web",
							Parameters: &genai.Schema{
								Type: genai.TypeObject,
								Properties: map[string]*genai.Schema{
									"query": {Type: genai.TypeString},
								},
								Required: []string{"query"},
							},
						},
					},
				},
			},
		},
	}
	tools := buildTools(req)
	if len(tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(tools))
	}
	if tools[0].Type != "function" {
		t.Errorf("expected type function, got %s", tools[0].Type)
	}
	if tools[0].Function.Name != "search" {
		t.Errorf("expected name search, got %s", tools[0].Function.Name)
	}
}

// TestGenerateSyncSuccess verifies a successful non-streaming response is handled correctly.
func TestGenerateSyncSuccess(t *testing.T) {
	respBody := chatCompletionResponse{
		ID: "chatcmpl-1",
		Choices: []choice{
			{
				Index: 0,
				Message: chatMessage{
					Role:    "assistant",
					Content: "Hello!",
				},
				FinishReason: "stop",
			},
		},
	}
	b, _ := json.Marshal(respBody)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(b)
	}))
	defer server.Close()

	m := NewModel("testkey", server.URL, "gpt-4o", server.Client())
	req := &model.LLMRequest{
		Contents: []*genai.Content{
			genai.NewContentFromText("hi", "user"),
		},
	}

	var responses []*model.LLMResponse
	for resp, err := range m.GenerateContent(context.Background(), req, false) {
		if err != nil {
			t.Fatalf("GenerateContent() error = %v", err)
		}
		responses = append(responses, resp)
	}
	if len(responses) != 1 {
		t.Fatalf("expected 1 response, got %d", len(responses))
	}
	if !responses[0].TurnComplete {
		t.Error("expected TurnComplete to be true")
	}
	if len(responses[0].Content.Parts) == 0 || responses[0].Content.Parts[0].Text != "Hello!" {
		t.Errorf("unexpected content: %v", responses[0].Content)
	}
}

// TestGenerateSyncAPIError verifies non-200 responses are surfaced as errors.
func TestGenerateSyncAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"invalid key"}`))
	}))
	defer server.Close()

	m := NewModel("bad-key", server.URL, "gpt-4o", server.Client())
	req := &model.LLMRequest{
		Contents: []*genai.Content{genai.NewContentFromText("hi", "user")},
	}

	for _, err := range m.GenerateContent(context.Background(), req, false) {
		if err != nil {
			if !strings.Contains(err.Error(), "API error") {
				t.Errorf("expected API error, got: %v", err)
			}
			return
		}
	}
	t.Error("expected an error but got none")
}

// TestGenerateStreamSuccess verifies streaming text accumulation.
func TestGenerateStreamSuccess(t *testing.T) {
	sseBody := "data: {\"id\":\"1\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"Hello\"},\"finish_reason\":null}]}\n" +
		"data: {\"id\":\"1\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\" World\"},\"finish_reason\":null}]}\n" +
		"data: [DONE]\n"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Write([]byte(sseBody))
	}))
	defer server.Close()

	m := NewModel("key", server.URL, "gpt-4o", server.Client())
	req := &model.LLMRequest{
		Contents: []*genai.Content{genai.NewContentFromText("hi", "user")},
	}

	var finalResp *model.LLMResponse
	for resp, err := range m.GenerateContent(context.Background(), req, true) {
		if err != nil {
			t.Fatalf("streaming error: %v", err)
		}
		if resp.TurnComplete {
			finalResp = resp
		}
	}

	if finalResp == nil {
		t.Fatal("expected final response with TurnComplete=true")
	}
	text := finalResp.Content.Parts[0].Text
	if text != "Hello World" {
		t.Errorf("expected 'Hello World', got %q", text)
	}
}

// TestGenerateStreamToolCall verifies streaming tool calls are assembled correctly.
func TestGenerateStreamToolCall(t *testing.T) {
	sseBody := `data: {"id":"1","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"id":"call_1","type":"function","function":{"name":"search","arguments":""}}]},"finish_reason":null}]}` + "\n" +
		`data: {"id":"1","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"{\"q\":"}}]},"finish_reason":null}]}` + "\n" +
		`data: {"id":"1","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"\"hello\"}"}}]},"finish_reason":null}]}` + "\n" +
		"data: [DONE]\n"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Write([]byte(sseBody))
	}))
	defer server.Close()

	m := NewModel("key", server.URL, "gpt-4o", server.Client())
	req := &model.LLMRequest{
		Contents: []*genai.Content{genai.NewContentFromText("hi", "user")},
	}

	var finalResp *model.LLMResponse
	for resp, err := range m.GenerateContent(context.Background(), req, true) {
		if err != nil {
			t.Fatalf("streaming error: %v", err)
		}
		if resp != nil && resp.TurnComplete {
			finalResp = resp
		}
	}

	if finalResp == nil {
		t.Fatal("expected final response with TurnComplete=true")
	}
	if len(finalResp.Content.Parts) == 0 {
		t.Fatal("expected at least one part in final response")
	}
	fc := finalResp.Content.Parts[0].FunctionCall
	if fc == nil {
		t.Fatal("expected FunctionCall in final response")
	}
	if fc.Name != "search" {
		t.Errorf("expected function name search, got %s", fc.Name)
	}
	if fc.Args["q"] != "hello" {
		t.Errorf("expected arg q=hello, got %v", fc.Args["q"])
	}
}

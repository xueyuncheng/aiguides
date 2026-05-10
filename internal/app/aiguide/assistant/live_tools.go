package assistant

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/memory"
	"google.golang.org/adk/session"
	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/toolconfirmation"
	"google.golang.org/genai"
)

// maxLiveToolResponseBytes is the safe upper bound for a single tool response
// sent via the Gemini Live API. Responses exceeding this are truncated to prevent
// the server from closing the WebSocket with error 1007 (invalid payload data).
const maxLiveToolResponseBytes = 30 * 1024

// localRunnableTool mirrors the unexported runnableTool interface in the ADK tool package,
// allowing us to extract FunctionDeclarations and invoke tools outside the ADK agent framework.
type localRunnableTool interface {
	tool.Tool
	Declaration() *genai.FunctionDeclaration
	Run(ctx tool.Context, args any) (map[string]any, error)
}

// liveToolRegistry maps function name → runnable tool for Live API execution.
type liveToolRegistry map[string]localRunnableTool

func buildLiveToolRegistry(tools []tool.Tool) liveToolRegistry {
	registry := make(liveToolRegistry)
	for _, t := range tools {
		if rt, ok := t.(localRunnableTool); ok {
			registry[t.Name()] = rt
		}
	}
	return registry
}

// buildGenaiTools converts ADK tools to []*genai.Tool for LiveConnectConfig.
func buildGenaiTools(tools []tool.Tool) []*genai.Tool {
	var decls []*genai.FunctionDeclaration
	for _, t := range tools {
		if rt, ok := t.(localRunnableTool); ok {
			decls = append(decls, rt.Declaration())
		}
	}
	if len(decls) == 0 {
		return nil
	}
	return []*genai.Tool{{FunctionDeclarations: decls}}
}

// executeLiveTool runs a single function call from the Gemini Live API and returns the response.
func executeLiveTool(ctx context.Context, registry liveToolRegistry, call *genai.FunctionCall, client *genai.Client, modelName string) *genai.FunctionResponse {
	rt, ok := registry[call.Name]
	if !ok {
		return &genai.FunctionResponse{
			ID:       call.ID,
			Name:     call.Name,
			Response: map[string]any{"error": fmt.Sprintf("unknown tool: %s", call.Name)},
		}
	}

	slog.Info("executeLiveTool", "tool", call.Name)
	result, err := rt.Run(&liveToolContext{Context: ctx}, call.Args)
	if err != nil {
		slog.Error("executeLiveTool: failed", "tool", call.Name, "err", err)
		return &genai.FunctionResponse{
			ID:       call.ID,
			Name:     call.Name,
			Response: map[string]any{"error": err.Error()},
		}
	}

	if data, jsonErr := json.Marshal(result); jsonErr == nil && len(data) > maxLiveToolResponseBytes {
		slog.Warn("executeLiveTool: response too large, summarizing", "tool", call.Name, "bytes", len(data))
		if summarized, err := summarizeToolResponse(ctx, client, modelName, call.Name, data); err == nil {
			result = summarized
		} else {
			slog.Warn("executeLiveTool: summarization failed, falling back to truncation", "err", err)
			result = truncateToolResponse(result, maxLiveToolResponseBytes)
		}
	}

	return &genai.FunctionResponse{
		ID:       call.ID,
		Name:     call.Name,
		Response: result,
	}
}

// liveToolContext is a minimal tool.Context for executing tools outside the ADK agent framework.
// Our tools only read context values (user ID, session ID) via context.Context — all ADK-specific
// methods are stubbed with safe zero values.
type liveToolContext struct {
	context.Context
}

func (c *liveToolContext) UserContent() *genai.Content          { return nil }
func (c *liveToolContext) InvocationID() string                 { return "" }
func (c *liveToolContext) AgentName() string                    { return "assistant" }
func (c *liveToolContext) ReadonlyState() session.ReadonlyState { return nil }
func (c *liveToolContext) UserID() string                       { return "" }
func (c *liveToolContext) AppName() string                      { return "" }
func (c *liveToolContext) SessionID() string                    { return "" }
func (c *liveToolContext) Branch() string                       { return "" }
func (c *liveToolContext) Artifacts() agent.Artifacts           { return nil }
func (c *liveToolContext) State() session.State                 { return nil }
func (c *liveToolContext) FunctionCallID() string               { return "" }
func (c *liveToolContext) Actions() *session.EventActions       { return &session.EventActions{} }
func (c *liveToolContext) SearchMemory(_ context.Context, _ string) (*memory.SearchResponse, error) {
	return nil, nil
}
func (c *liveToolContext) ToolConfirmation() *toolconfirmation.ToolConfirmation { return nil }
func (c *liveToolContext) RequestConfirmation(_ string, _ any) error            { return nil }

func summarizeToolResponse(ctx context.Context, client *genai.Client, modelName, toolName string, data []byte) (map[string]any, error) {
	if client == nil || modelName == "" {
		return nil, fmt.Errorf("genai client or model name not available")
	}

	prompt := fmt.Sprintf(
		"以下是工具 %s 返回的完整数据（JSON）。请用中文简洁地总结关键信息，保留重要的名称、日期、数字等细节，去掉冗余内容。直接输出总结，不要加额外格式。\n\n%s",
		toolName, string(data),
	)

	resp, err := client.Models.GenerateContent(ctx, modelName, genai.Text(prompt), nil)
	if err != nil {
		return nil, fmt.Errorf("GenerateContent: %w", err)
	}

	summary := resp.Text()
	if summary == "" {
		return nil, fmt.Errorf("empty summary response")
	}

	return map[string]any{
		"summary":    summary,
		"_note":      fmt.Sprintf("original response was %d bytes, summarized by AI", len(data)),
		"tool":       toolName,
		"summarized": true,
	}, nil
}

// truncateToolResponse reduces a tool response to fit within maxBytes.
// For responses with a list field (e.g. "messages", "items", "results"), it removes
// items from the end until the response fits. For other responses, it returns a
// summary indicating the response was too large.
func truncateToolResponse(result map[string]any, maxBytes int) map[string]any {
	listKeys := []string{"messages", "items", "results", "events", "tasks", "files", "servers"}
	for _, key := range listKeys {
		val, ok := result[key]
		if !ok {
			continue
		}
		slice, ok := val.([]any)
		if !ok {
			continue
		}
		if len(slice) == 0 {
			continue
		}

		totalCount := len(slice)
		for len(slice) > 0 {
			result[key] = slice
			data, err := json.Marshal(result)
			if err != nil {
				break
			}
			if len(data) <= maxBytes {
				result["_truncated"] = fmt.Sprintf("showing %d of %d items (response truncated to fit size limit)", len(slice), totalCount)
				return result
			}
			slice = slice[:len(slice)-1]
		}
		result[key] = []any{}
		result["_truncated"] = fmt.Sprintf("all %d items removed — each item exceeds size limit individually", totalCount)
		return result
	}

	data, _ := json.Marshal(result)
	return map[string]any{
		"error": fmt.Sprintf("tool response was %d bytes (limit %d). Please use a smaller limit or more specific query.", len(data), maxBytes),
	}
}

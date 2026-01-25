package assistant

import (
	"aiguide/internal/pkg/tools"
	"context"
	"testing"

	"google.golang.org/adk/model"
)

func TestNewSearchAgent(t *testing.T) {
	// This test verifies that the search agent can be created
	// without errors when a nil genaiClient is passed (basic structure test)
	ctx := context.Background()
	webSearchConfig := tools.WebSearchConfig{
		SearXNG: tools.SearXNGConfig{
			InstanceURL: "https://searx.be",
		},
	}

	var _ context.Context = ctx

	agentConfig := &AssistantAgentConfig{
		Model:             nil,
		GenaiClient:       nil,
		DB:                nil,
		MockImageGen:      true,
		MockEmailIMAPConn: true,
		WebSearchConfig:   webSearchConfig,
	}
	agent, err := NewAssistantAgent(agentConfig)
	if err != nil {
		t.Fatalf("NewSearchAgent() error = %v", err)
	}

	if agent == nil {
		t.Fatal("NewSearchAgent() returned nil agent")
	}
}

func TestNewSearchAgentWithModel(t *testing.T) {
	// Test with a mock model to verify the structure
	// In a real scenario, you would use a proper mock model
	ctx := context.Background()
	webSearchConfig := tools.WebSearchConfig{
		SearXNG: tools.SearXNGConfig{
			InstanceURL: "https://searx.be",
		},
	}

	var _ context.Context = ctx

	agentConfig := &AssistantAgentConfig{
		Model:             model.LLM(nil),
		GenaiClient:       nil,
		DB:                nil,
		MockImageGen:      true,
		MockEmailIMAPConn: true,
		WebSearchConfig:   webSearchConfig,
	}
	agent, err := NewAssistantAgent(agentConfig)
	if err != nil {
		t.Fatalf("NewSearchAgent() with model error = %v", err)
	}

	if agent == nil {
		t.Fatal("NewSearchAgent() returned nil agent")
	}
}

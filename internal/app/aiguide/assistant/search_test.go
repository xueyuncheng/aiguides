package assistant

import (
	"testing"

	"google.golang.org/adk/model"
)

func TestNewSearchAgent(t *testing.T) {
	// This test verifies that the search agent can be created
	// without errors when a nil genaiClient is passed (basic structure test)
	agent, err := NewSearchAgent(nil, nil, true, "http://localhost:3000")
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
	agent, err := NewSearchAgent(model.LLM(nil), nil, true, "http://localhost:3000")
	if err != nil {
		t.Fatalf("NewSearchAgent() with model error = %v", err)
	}

	if agent == nil {
		t.Fatal("NewSearchAgent() returned nil agent")
	}
}

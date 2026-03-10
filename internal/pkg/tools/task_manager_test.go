package tools

import (
	"context"
	"iter"
	"testing"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/memory"
	adksession "google.golang.org/adk/session"
	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/toolconfirmation"
	"google.golang.org/genai"
)

func TestFinishPlanningTransfersBackToRootAgent(t *testing.T) {
	t.Parallel()

	finishPlanningTool, err := NewFinishPlanningTool()
	if err != nil {
		t.Fatalf("NewFinishPlanningTool() error = %v", err)
	}

	runnableTool, ok := finishPlanningTool.(interface {
		Run(tool.Context, any) (map[string]any, error)
	})
	if !ok {
		t.Fatal("finish_planning tool does not implement Run")
	}

	toolCtx := &fakeToolContext{
		Context: context.Background(),
		actions: &adksession.EventActions{},
	}

	result, err := runnableTool.Run(toolCtx, map[string]any{
		"summary":    "已完成规划",
		"task_count": 2,
	})
	if err != nil {
		t.Fatalf("finish_planning Run() error = %v", err)
	}

	if got := toolCtx.actions.TransferToAgent; got != rootAgentName {
		t.Fatalf("finish_planning should transfer to %q, got %q", rootAgentName, got)
	}

	if got := result["status"]; got != "completed" {
		t.Fatalf("finish_planning status = %v, want completed", got)
	}
}

type fakeToolContext struct {
	context.Context
	actions *adksession.EventActions
}

func (f *fakeToolContext) UserContent() *genai.Content {
	return nil
}

func (f *fakeToolContext) InvocationID() string {
	return ""
}

func (f *fakeToolContext) AgentName() string {
	return "planner"
}

func (f *fakeToolContext) ReadonlyState() adksession.ReadonlyState {
	return fakeState{}
}

func (f *fakeToolContext) UserID() string {
	return ""
}

func (f *fakeToolContext) AppName() string {
	return ""
}

func (f *fakeToolContext) SessionID() string {
	return ""
}

func (f *fakeToolContext) Branch() string {
	return ""
}

func (f *fakeToolContext) Artifacts() agent.Artifacts {
	return nil
}

func (f *fakeToolContext) State() adksession.State {
	return fakeState{}
}

func (f *fakeToolContext) FunctionCallID() string {
	return ""
}

func (f *fakeToolContext) Actions() *adksession.EventActions {
	return f.actions
}

func (f *fakeToolContext) SearchMemory(context.Context, string) (*memory.SearchResponse, error) {
	return nil, nil
}

func (f *fakeToolContext) ToolConfirmation() *toolconfirmation.ToolConfirmation {
	return nil
}

func (f *fakeToolContext) RequestConfirmation(string, any) error {
	return nil
}

type fakeState struct{}

func (fakeState) Get(string) (any, error) {
	return nil, adksession.ErrStateKeyNotExist
}

func (fakeState) Set(string, any) error {
	return nil
}

func (fakeState) All() iter.Seq2[string, any] {
	return func(func(string, any) bool) {}
}

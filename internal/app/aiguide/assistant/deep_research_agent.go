package assistant

import (
	_ "embed"
	"fmt"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/agent/workflowagents/parallelagent"
	"google.golang.org/adk/agent/workflowagents/sequentialagent"
	"google.golang.org/adk/model"
	"google.golang.org/adk/tool"
	"google.golang.org/genai"
)

//go:embed research_planner_prompt.md
var researchPlannerInstruction string

//go:embed researcher_breadth_prompt.md
var researcherBreadthInstruction string

//go:embed researcher_depth_prompt.md
var researcherDepthInstruction string

//go:embed researcher_verify_prompt.md
var researcherVerifyInstruction string

//go:embed report_writer_prompt.md
var reportWriterInstruction string

// buildDeepResearchAgent creates a SequentialAgent that orchestrates deep research:
//
//	research_planner → parallel_research(breadth, depth, verify) → report_writer
func buildDeepResearchAgent(researchTools []tool.Tool, m model.LLM, thinkingBudget int32) (agent.Agent, error) {
	thinkingCfg := newThinkingConfig(thinkingBudget)

	var plannerTools []tool.Tool
	for _, t := range researchTools {
		if t.Name() == "current_time" {
			plannerTools = append(plannerTools, t)
		}
	}

	planner, err := llmagent.New(llmagent.Config{
		Name:        "research_planner",
		Model:       m,
		Description: "Breaks research questions into structured sub-topics and search strategies",
		Instruction: researchPlannerInstruction,
		GenerateContentConfig: &genai.GenerateContentConfig{
			ThinkingConfig: thinkingCfg,
		},
		Tools: plannerTools,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create research_planner: %w", err)
	}

	type researcherDef struct {
		name        string
		description string
		instruction string
	}

	researchers := []researcherDef{
		{
			name:        "researcher_breadth",
			description: "Broad coverage researcher: surveys all sub-topics to ensure nothing is missed",
			instruction: researcherBreadthInstruction,
		},
		{
			name:        "researcher_depth",
			description: "Deep analysis researcher: dives deep into the most critical sub-topics with detailed reading",
			instruction: researcherDepthInstruction,
		},
		{
			name:        "researcher_verify",
			description: "Verification researcher: cross-checks key claims and seeks opposing viewpoints",
			instruction: researcherVerifyInstruction,
		},
	}

	researcherAgents := make([]agent.Agent, 0, len(researchers))
	for _, def := range researchers {
		a, err := llmagent.New(llmagent.Config{
			Name:        def.name,
			Model:       m,
			Description: def.description,
			Instruction: def.instruction,
			GenerateContentConfig: &genai.GenerateContentConfig{
				ThinkingConfig: thinkingCfg,
			},
			Tools: researchTools,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create %s: %w", def.name, err)
		}
		researcherAgents = append(researcherAgents, a)
	}

	parallelResearch, err := parallelagent.New(parallelagent.Config{
		AgentConfig: agent.Config{
			Name:      "parallel_research",
			SubAgents: researcherAgents,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create parallel_research: %w", err)
	}

	writer, err := llmagent.New(llmagent.Config{
		Name:        "report_writer",
		Model:       m,
		Description: "Synthesizes all research findings into a comprehensive structured report",
		Instruction: reportWriterInstruction,
		GenerateContentConfig: &genai.GenerateContentConfig{
			ThinkingConfig: thinkingCfg,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create report_writer: %w", err)
	}

	deepResearch, err := sequentialagent.New(sequentialagent.Config{
		AgentConfig: agent.Config{
			Name:      "deep_research_agent",
			SubAgents: []agent.Agent{planner, parallelResearch, writer},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create deep_research_agent: %w", err)
	}

	return deepResearch, nil
}

package assistant

import (
	_ "embed"
	"fmt"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/model"
	"google.golang.org/adk/tool"
	"google.golang.org/genai"
)

//go:embed web_agent_prompt.md
var webAgentInstruction string

//go:embed comms_agent_prompt.md
var commsAgentInstruction string

//go:embed media_agent_prompt.md
var mediaAgentInstruction string

//go:embed file_agent_prompt.md
var fileAgentInstruction string

//go:embed task_agent_prompt.md
var taskAgentInstruction string

//go:embed system_agent_prompt.md
var systemAgentInstruction string

type toolPartition struct {
	Common []tool.Tool
	Web    []tool.Tool
	Comms  []tool.Tool
	Media  []tool.Tool
	File   []tool.Tool
	Task   []tool.Tool
	System []tool.Tool
}

func partitionTools(allTools []tool.Tool) toolPartition {
	var p toolPartition
	for _, t := range allTools {
		switch t.Name() {
		case "current_time":
			p.Common = append(p.Common, t)
			p.Web = append(p.Web, t)
		case "manage_memory":
			p.Common = append(p.Common, t)
		case "web_search", "exa_search", "web_fetch":
			p.Web = append(p.Web, t)
		case "query_emails", "send_email", "manage_calendar":
			p.Comms = append(p.Comms, t)
		case "generate_image", "generate_video", "audio_transcribe",
			"pdf_extract_text", "pdf_generate_document":
			p.Media = append(p.Media, t)
		case "file_download", "file_list", "file_get":
			p.File = append(p.File, t)
		case "task_list", "task_get", "task_update",
			"scheduled_task_create", "scheduled_task_list":
			p.Task = append(p.Task, t)
		case "ssh_list_servers", "ssh_execute":
			p.System = append(p.System, t)
		default:
			p.Common = append(p.Common, t)
		}
	}
	return p
}

func buildSubAgents(p toolPartition, m model.LLM) ([]agent.Agent, error) {
	type subAgentDef struct {
		name        string
		description string
		instruction string
		tools       []tool.Tool
	}

	defs := []subAgentDef{
		{
			name:        "web_agent",
			description: "Handles web search, semantic research (Exa), and web page content fetching",
			instruction: webAgentInstruction,
			tools:       p.Web,
		},
		{
			name:        "comms_agent",
			description: "Manages email (query and send) and Google Calendar operations",
			instruction: commsAgentInstruction,
			tools:       p.Comms,
		},
		{
			name:        "media_agent",
			description: "Generates images and videos, transcribes audio, and processes PDF documents",
			instruction: mediaAgentInstruction,
			tools:       p.Media,
		},
		{
			name:        "file_agent",
			description: "Downloads files from URLs and manages the user's file storage",
			instruction: fileAgentInstruction,
			tools:       p.File,
		},
		{
			name:        "task_agent",
			description: "Manages tasks (list, get, update) and scheduled/recurring tasks",
			instruction: taskAgentInstruction,
			tools:       p.Task,
		},
		{
			name:        "system_agent",
			description: "Lists remote SSH servers and executes commands on them",
			instruction: systemAgentInstruction,
			tools:       p.System,
		},
	}

	agents := make([]agent.Agent, 0, len(defs))
	for _, def := range defs {
		if len(def.tools) == 0 {
			continue
		}
		a, err := createSubAgent(def.name, def.description, def.instruction, def.tools, m)
		if err != nil {
			return nil, err
		}
		agents = append(agents, a)
	}
	return agents, nil
}

func createSubAgent(name, description, instruction string, tools []tool.Tool, m model.LLM) (agent.Agent, error) {
	cfg := llmagent.Config{
		Name:        name,
		Model:       m,
		Description: description,
		Instruction: instruction,
		GenerateContentConfig: &genai.GenerateContentConfig{
			ThinkingConfig: &genai.ThinkingConfig{
				IncludeThoughts: true,
			},
		},
		Tools: tools,
	}
	a, err := llmagent.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create sub-agent %q: %w", name, err)
	}
	return a, nil
}

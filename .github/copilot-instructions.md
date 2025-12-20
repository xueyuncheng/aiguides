# AIGuide Project Instructions

This project is an AI Agent framework built using the [Google ADK (Agent Development Kit)](https://github.com/google/adk).

## Architecture Overview

- **Core Framework**: Uses `google.golang.org/adk` for agent definition and execution.
- **Entry Point**: [cmd/aiguide/aiguide.go](cmd/aiguide/aiguide.go) handles configuration loading and starts the agent launcher.
- **Agent Logic**: Located in [internal/app/aiguide/](internal/app/aiguide/). Agents are defined using `agent.Config` and implemented via a `Run` function.
- **Launcher**: Uses `launcher.Launcher` to execute agents, supporting different execution modes (e.g., `full.NewLauncher`).

## Key Patterns & Conventions

### 1. Agent Implementation
Agents must implement a `Run` function with the following signature:
```go
func runFunc(ctx agent.InvocationContext) iter.Seq2[*session.Event, error] {
    return func(yield func(*session.Event, error) bool) {
        // Implementation logic
        // Use yield(event, nil) to send events back to the session
    }
}
```
- Example: [searchAgentRun](internal/app/aiguide/summary.go#L58) in `internal/app/aiguide/summary.go`.

### 2. Configuration
- Configuration is defined in `Config` structs with `yaml` tags.
- Default config file is `aiguide.yaml`.
- See [Config](internal/app/aiguide/aiguide.go#L16) in `internal/app/aiguide/aiguide.go`.

### 3. Logging
- Use `log/slog` for structured logging.
- Prefer `slog.Error("message", "err", err)` for error logging.

### 4. Dependency Management
- Core dependencies: `google.golang.org/adk`, `google.golang.org/genai`.
- Use `go get` to add new ADK sub-packages (e.g., `google.golang.org/adk/agent/workflowagents/sequentialagent`).

## Developer Workflows

### Running the Application
```bash
go run cmd/aiguide/aiguide.go -f aiguide.yaml
```

### Adding a New Agent
1. Define the agent configuration in [internal/app/aiguide/summary.go](internal/app/aiguide/summary.go) (or a new file in that directory).
2. Implement the `Run` function using the `iter.Seq2` pattern.
3. Register the agent in the `AIGuide` struct or as a sub-agent of an existing agent.

## Important Files
- [internal/app/aiguide/aiguide.go](internal/app/aiguide/aiguide.go): Main application logic and launcher setup.
- [internal/app/aiguide/summary.go](internal/app/aiguide/summary.go): Example agent implementations (`SummaryAgent`, `SearchAgent`).
- [go.mod](go.mod): Module definition and dependencies.

## When to write code

1. Always use Effective Go principles. Reference link is https://go.dev/doc/effective_go
2. Always use google go code style. Reference link is https://google.github.io/styleguide/go/

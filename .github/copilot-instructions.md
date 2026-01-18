# AIGuide Project Instructions

This project is a full-stack AI assistant built with Go (Gin + Google ADK) and a Next.js frontend.

## Architecture Overview

- **Backend entry point**: [cmd/aiguide/aiguide.go](cmd/aiguide/aiguide.go) loads YAML config and starts the service.
- **Core app**: [internal/app/aiguide/](internal/app/aiguide/) contains config, router, handlers, migration, and auth wiring.
- **Assistant runtime**: [internal/app/aiguide/assistant/](internal/app/aiguide/assistant/) builds the ADK agent, runner, SSE streaming, and session APIs.
- **Tools**: [internal/pkg/tools/](internal/pkg/tools/) provides `generate_image` (Imagen) and `query_emails` (IMAP).
- **Auth & middleware**: [internal/pkg/auth/](internal/pkg/auth/) and [internal/pkg/middleware/](internal/pkg/middleware/) handle Google OAuth + JWT cookies.
- **Frontend**: [frontend/app/](frontend/app/) includes login, chat, and settings pages.

## Key Patterns & Conventions

### 1. Agent Implementation
- The assistant agent is built in [internal/app/aiguide/assistant/agent.go](internal/app/aiguide/assistant/agent.go) using `llmagent.New`.
- Streaming responses are handled via SSE in [internal/app/aiguide/assistant/sse.go](internal/app/aiguide/assistant/sse.go).
- Session APIs are in [internal/app/aiguide/assistant/session.go](internal/app/aiguide/assistant/session.go).

### 2. Configuration
- YAML config is defined in `Config` with `yaml` tags in [internal/app/aiguide/aiguide.go](internal/app/aiguide/aiguide.go).
- Default dev config lives at [cmd/aiguide/aiguide.yaml](cmd/aiguide/aiguide.yaml) (see example in [cmd/aiguide/aiguide.yaml.example](cmd/aiguide/aiguide.yaml.example)).

### 3. Logging
- Use `log/slog` for structured logging.
- Prefer `slog.Error("message", "err", err)` for error logging.

### 4. Function Order
- Place main/entry functions above the helper functions they call.

### 5. Dependency Management
- Core dependencies: `google.golang.org/adk`, `google.golang.org/genai`.
- Use `go get` to add ADK sub-packages as needed.

## Developer Workflows

### Running the Application
```bash
go run cmd/aiguide/aiguide.go -f cmd/aiguide/aiguide.yaml
```

### Adding or Extending Tools
1. Implement a new tool in [internal/pkg/tools/](internal/pkg/tools/).
2. Register it in [internal/app/aiguide/assistant/agent.go](internal/app/aiguide/assistant/agent.go).
3. If the tool requires DB/auth context, use middleware utilities in [internal/pkg/middleware/](internal/pkg/middleware/).

## Important Files
- [cmd/aiguide/aiguide.go](cmd/aiguide/aiguide.go): App entry point.
- [internal/app/aiguide/aiguide.go](internal/app/aiguide/aiguide.go): Config + service wiring.
- [internal/app/aiguide/router.go](internal/app/aiguide/router.go): API routes (auth, sessions, assistant chat, settings).
- [internal/app/aiguide/assistant/](internal/app/aiguide/assistant/): Agent, runner, SSE, session APIs.
- [internal/pkg/tools/](internal/pkg/tools/): Image generation + email query tools.
- [frontend/app/](frontend/app/): Next.js UI (login, chat, settings).
- [go.mod](go.mod): Module definition and dependencies.

## When to write code

1. Always use Effective Go principles. Reference link is https://go.dev/doc/effective_go
2. Always use google go code style. Reference link is https://google.github.io/styleguide/go/

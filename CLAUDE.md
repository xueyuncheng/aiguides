# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

AIGuides is a full-stack AI assistant built with Go (Gin + Google ADK/Gemini) backend and Next.js frontend. It supports multimodal chat (text + images), image generation via Imagen, email querying via IMAP, web search/fetch, cross-session memory, session management, and Google OAuth authentication.

**Tech Stack:**
- Backend: Go 1.25.5+, Gin, GORM, SQLite, Google ADK
- Frontend: Next.js 15, React 19, TypeScript, Tailwind CSS 4
- AI: Google Gemini 2.0 + Imagen

## Development Commands

### Backend
```bash
# Run backend (development)
go run cmd/aiguide/aiguide.go -f cmd/aiguide/aiguide.yaml

# Format code
go fmt ./...
make fmt

# Run tests
go test ./...

# Run specific package tests
go test -v ./internal/app/aiguide/assistant

# Run specific test
go test -v ./internal/app/aiguide/assistant -run TestAgentCreation
```

### Frontend
```bash
cd frontend

# Install dependencies
pnpm install

# Development server
pnpm dev

# Production build
pnpm build

# Lint
pnpm lint
```

### Full Stack
```bash
# Start both backend and frontend
./scripts/start.sh

# Backend: http://localhost:8080
# Frontend: http://localhost:3000
```

### Docker
```bash
make build          # Build both backend and frontend images
make deploy         # Deploy with docker-compose (4 services: backend, frontend, SearXNG, Redis)
make down           # Stop services
```

## Architecture

### Backend Structure

**Entry Point:**
- `cmd/aiguide/aiguide.go` - Loads YAML config and starts service

**Core Application (`internal/app/aiguide/`):**
- `aiguide.go` - Service initialization and `Config` struct with YAML tags
- `router.go` - API routes (auth, sessions, assistant chat, settings, share)
- `login.go` - Google OAuth handlers
- `migration/` - Database auto-migration using GORM
- `table/` - Database models: User, Session, Message, EmailServerConfig, ShareSession
- `setting/` - Settings API (email server configs)

**Assistant Runtime (`internal/app/aiguide/assistant/`):**

The assistant uses a **multi-agent architecture**:
- `agent.go` - Builds the root agent with all tools via `llmagent.New`
- `planner_agent.go` + `planner_agent_prompt.md` - Planner agent for task decomposition
- `executor_agent.go` + `executor_agent_prompt.md` - Executor agent for task execution
- `assistant_agent_prompt.md` - Root agent system prompt (embedded via `//go:embed`)
- `runner.go` - Runner initialization and management
- `sse.go` - SSE streaming chat handler with multimodal support
- `session.go` - Session CRUD APIs (create, list, delete, history)
- `session_edit.go` - Session message editing
- `share.go` - Time-limited readonly session sharing

**Tools (`internal/pkg/tools/`):**
- `imagegen.go` - Imagen image generation (mock mode, aspect ratios, multiple images)
- `email.go` - IMAP email query tool (requires user email server config)
- `websearch.go` - SearXNG web search integration
- `webfetch.go` - Web page fetching/scraping via go-readability
- `exasearch.go` - Exa neural search integration
- `memory.go` - Cross-session memory management
- `task_manager.go` - Agent task management
- `time.go` - Time utilities for agents

**Infrastructure (`internal/pkg/`):**
- `auth/` - Google OAuth + JWT cookie implementation
- `middleware/` - Auth middleware, context utilities
- `constant/` - Shared constants

### Frontend Structure

**Pages (`frontend/app/`):**
- `page.tsx` - Landing page
- `login/` - Google OAuth login flow
- `chat/` - Sessions list
- `chat/[sessionId]/` - Main chat interface with multimodal input
  - `components/` - Chat-specific components (ChatInput, MessageContent, ShareModal, etc.)
  - `hooks/` - Custom hooks (useFileUpload)
  - `utils/markdown.tsx` - Markdown + KaTeX rendering
  - `types.ts`, `constants.ts` - Type definitions and constants
- `settings/` - Email server configuration UI
- `share/[shareId]/` - Read-only shared conversation view
- `components/ui/` - Radix UI component wrappers
- `contexts/AuthContext.tsx` - Authentication state

### Key Patterns

**Agent Tool Registration:**
Tools are registered in `internal/app/aiguide/assistant/agent.go` via `llmagent.New`. Planner and executor sub-agents are registered as tools on the root agent.

**SSE Streaming:**
Chat responses stream via Server-Sent Events in `sse.go`:
- Supports text + image multipart input
- Validates image size (5MB max), count (4 max), and MIME types
- Streams agent responses with tool calls and results
- Saves messages to database for session history

**Database:**
- SQLite with GORM, auto-migration on startup via `migration/migration.go`
- Models in `internal/app/aiguide/table/`; register new models in `GetAllModels()`

**Authentication:**
- Google OAuth flow: `/api/auth/login/google` → `/api/auth/callback/google`
- JWT tokens in HTTP-only cookies; refresh via `/api/auth/refresh`
- Email whitelist support via `allowed_emails` config
- Auth middleware protects all routes except public endpoints (e.g., `/api/share/*`)

**Configuration:**
- YAML config at `cmd/aiguide/aiguide.yaml`; see `cmd/aiguide/aiguide.yaml.example`
- Required: `api_key`, `model_name`
- Optional: OAuth credentials, JWT secret, allowed emails, `web_search.instance_url` (SearXNG), `exa_search.api_key`, `mock_image_generation`

## Code Style Guidelines

### Go Code
Follow [Effective Go](https://go.dev/doc/effective_go) and [Google Go Style Guide](https://google.github.io/styleguide/go/).

**Specific conventions:**
- Use `log/slog` for structured logging: `slog.Error("message", "err", err)`
- Place main/entry functions above helper functions they call
- Tool handlers return `(*Output, error)` for ADK integration

**Error Handling and Logging:**
- **At error source**: MUST `slog.Error()` with full context, then return wrapped error
- **At error propagation**: DO NOT log — just wrap and return with `fmt.Errorf("...: %w", err)`
- Log once at the source; never log the same error multiple times as it propagates
- Exception: GORM database errors should always log at source

```go
// At source (stdlib/3rd-party call):
resp, err := http.DefaultClient.Do(req)
if err != nil {
    slog.Error("http.DefaultClient.Do() error", "url", url, "err", err)
    return nil, fmt.Errorf("请求失败: %w", err)
}

// At propagation (internal calls):
result, err := someInternalFunction()
if err != nil {
    return nil, fmt.Errorf("failed to process: %w", err)  // No slog.Error here
}
```

**Dependency Management:**
- Core deps: `google.golang.org/adk`, `google.golang.org/genai`
- Add ADK sub-packages as needed: `go get google.golang.org/adk/...`

### TypeScript/React
- TypeScript with strict type checking; functional components with hooks
- Tailwind CSS for styling; ESLint for linting

## Important Implementation Notes

### Adding New Tools
1. Create tool in `internal/pkg/tools/` with ADK `functiontool` pattern
2. Define Input/Output structs with JSON schema tags
3. Register in `agent.go` via `NewAssistantAgent`
4. For DB/auth context, use middleware utilities from `internal/pkg/middleware/`

### Modifying Agent Behavior
- Root agent system prompt: `internal/app/aiguide/assistant/assistant_agent_prompt.md`
- Planner/executor prompts: `planner_agent_prompt.md`, `executor_agent_prompt.md`
- Model config: Change in YAML (`model_name`, `api_key`)

### Frontend API Integration
- API base: `/api` prefix (proxied to backend via Next.js rewrites in `next.config.ts`)
- Auth: JWT cookie sent automatically
- SSE endpoint: `/api/assistant/chats/:id` (POST)
- Session APIs: `/api/:agentId/sessions/*`
- Share API: `/api/assistant/share` (POST to create), `/api/share/:shareId` (GET, public)

## Testing

16 test files covering agent creation, SSE streaming, session management, sharing, OAuth, router configuration, and all tool implementations.

```bash
go test ./...                                          # All tests
go test -v ./internal/app/aiguide/assistant           # Assistant package
go test -v ./internal/pkg/tools                       # Tools package
```

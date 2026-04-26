# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

AIGuides is a full-stack AI assistant built with Go (Gin + Google ADK/Gemini) backend and Next.js frontend. It supports multimodal chat (text + images), image/video generation, audio transcription, email querying via IMAP, web search/fetch, SSH command execution, cross-session memory, scheduled tasks, project organization, and Google OAuth authentication.

**Tech Stack:**
- Backend: Go 1.26.1, Gin, GORM, SQLite, Google ADK
- Frontend: Next.js 15, React 19, TypeScript, Tailwind CSS 4
- AI: Google Gemini 2.0 + Imagen + Veo 3.1

## Development Commands

### Backend
```bash
go run cmd/aiguide/aiguide.go -f cmd/aiguide/aiguide.yaml   # Run dev server
go fmt ./...  # or: make fmt                                 # Format code
go test ./...                                                 # All tests
go test -count=1 ./...                                        # Tests without cache
go test -v ./internal/app/aiguide/assistant                  # Specific package
go test -v ./internal/app/aiguide/assistant -run TestAgentCreation  # Specific test
go test -v ./internal/pkg/tools -run TestWebFetch            # Test in tools package
go test -count=1 -v ./internal/app/aiguide/assistant -run TestName  # No cache + specific test
```

### Frontend
```bash
cd frontend
pnpm install          # Install dependencies
pnpm dev              # Dev server (http://localhost:3000)
pnpm lint             # ESLint
pnpm test             # Vitest (all tests)
pnpm test -- app/chat/[sessionId]/hooks/useStreamingChat.test.tsx   # Single test file
pnpm test -- -t "test name"                                          # Filter tests by name
pnpm exec vitest run app/chat/[sessionId]/hooks/useStreamingChat.test.tsx -t "test name"  # File + name
pnpm verify           # lint + test + build — run before pushing frontend changes
```

### Full Stack
```bash
./scripts/start.sh    # Starts backend (:8080) and frontend (:3000) in foreground
```

### Docker
```bash
make build            # Build both images
make build-backend    # Backend image only
make build-frontend   # Frontend image only
make deploy           # docker-compose up (backend, frontend, SearXNG, Redis)
make down             # Stop services
make save-images      # Export images to .tar for server transfer
make load-images      # Import images from .tar on server
```

## Architecture

### Backend Structure

**Entry Point:** `cmd/aiguide/aiguide.go` — loads YAML config, starts Gin server.

**Core Application (`internal/app/aiguide/`):**
- `aiguide.go` — Service initialization and `Config` struct with YAML tags
- `router.go` — All API route registration
- `login.go` — Google OAuth handlers
- `migration/` — GORM auto-migration on startup
- `table/table.go` — All database models (register new models in `GetAllModels()`)
- `setting/` — CRUD APIs for email and SSH server configurations

**Assistant Runtime (`internal/app/aiguide/assistant/`):**

Single assistant agent with all tools registered in `agent.go` via `llmagent.New`. A separate planner agent (`planner_agent.go`) exists for task decomposition with its own tool set (task CRUD + `finish_planning`).

Key files:
- `agent.go` — Tool creation and registration for the assistant agent
- `planner_agent.go` + `planner_agent_prompt.md` — Planner agent for task decomposition
- `assistant_agent_prompt.md` — Root agent system prompt (embedded via `//go:embed`)
- `runner.go` — ADK runner initialization
- `assistant.go` — Assistant struct wiring (main runner, executor runner for scheduled tasks, scheduler)
- `sse.go` — SSE streaming chat handler with multimodal support
- `tts_api.go` — Streaming TTS endpoint with sentence-level chunking
- `session.go` — Session CRUD APIs
- `session_edit.go` — Session message editing (branching)
- `share.go` — Time-limited readonly session sharing
- `project.go` — Project CRUD APIs
- `memory_api.go` — Memory CRUD and summary APIs
- `scheduled_task_api.go` — Scheduled task management APIs
- `scheduler.go` — Background scheduler executing due tasks via executor runner

**Tools (`internal/pkg/tools/`):**
Each tool follows the ADK `functiontool` pattern with Input/Output structs and JSON schema tags.
- `imagegen.go` — Imagen image generation
- `videogen.go` — Veo 3.1 video generation
- `audio.go` — Audio transcription (chunked processing)
- `email.go` / `send_email.go` — IMAP query and email sending
- `websearch.go` — SearXNG web search
- `webfetch.go` — Web page fetching via go-readability
- `exasearch.go` — Exa neural search
- `memory.go` — Cross-session memory management
- `task_manager.go` — Task CRUD (list, get, create, update)
- `scheduled_task.go` — Scheduled task creation
- `ssh.go` — SSH command execution (list servers, execute)
- `file_asset.go` / `file_download.go` — File management
- `pdf.go` — PDF text extraction and document generation
- `time.go` — Current time utility

**Infrastructure (`internal/pkg/`):**
- `auth/` — Google OAuth + JWT cookie implementation
- `middleware/` — Auth middleware, rate limiter, locale, context utilities
- `redis/` — Redis client (rate limiting)
- `storage/` — Local file store for uploaded assets
- `constant/` — Shared constants and enums

### Frontend Structure

- `frontend/app/page.tsx` — Landing page
- `frontend/app/login/` — Google OAuth login flow
- `frontend/app/chat/` — Sessions list
- `frontend/app/chat/[sessionId]/` — Main chat interface with multimodal input
  - `components/` — ChatInput, MessageContent, ShareModal, etc.
  - `hooks/` — useFileUpload, etc.
  - `utils/markdown.tsx` — Markdown + KaTeX rendering
- `frontend/app/settings/` — Email/SSH server configuration UI
- `frontend/app/share/[shareId]/` — Read-only shared conversation view
- `frontend/app/contexts/AuthContext.tsx` — Authentication state
- `frontend/app/components/ui/` — Radix UI component wrappers

### Key Patterns

**SSE Streaming:**
`sse.go` handles chat requests — validates multipart input (images: 5MB max, 4 max), streams agent responses with tool calls/results, saves messages to DB.

**Database:**
SQLite with GORM. Models in `table/table.go`; register new models in `GetAllModels()`. Auto-migration on startup via `migration/migration.go`.

**Authentication:**
Google OAuth flow: `/api/auth/login/google` → `/api/auth/callback/google`. JWT tokens in HTTP-only cookies; refresh via `/api/auth/refresh`. Email whitelist via `allowed_emails` config. Auth middleware protects all routes except public endpoints (`/api/share/*`, `/api/health`).

**Frontend API Proxy:**
`next.config.ts` rewrites `/api/*`, `/auth/*`, `/config`, `/health` to backend. Backend URL configured via `NEXT_PUBLIC_BACKEND_URL` env var (defaults to `http://backend:8080`).

**Configuration:**
YAML config at `cmd/aiguide/aiguide.yaml` (see `cmd/aiguide/aiguide.yaml.example`).
- Required: `api_key`, `model_name`
- Required for rate limiting: `redis.addr`
- Optional: OAuth credentials, JWT secret, allowed emails, `web_search.instance_url`, `exa_search.api_key`, `mock_image_generation`, `mock_video_generation`, rate limit settings

### API Routes

Authenticated routes (under `/api`):
- `POST /api/assistant/chats/:id` — SSE streaming chat
- `POST /api/assistant/tts/stream` — TTS streaming
- `/api/assistant/share` — Share management (POST create, GET list, DELETE /:shareId)
- `/api/assistant/memories` — Memory CRUD (GET list, POST create, GET /summary, PATCH /:memoryId, DELETE /:memoryId)
- `/api/assistant/scheduled-tasks` — Scheduled tasks (GET list, PATCH /:taskId, DELETE /:taskId)
- `/api/assistant/projects` — Project management (GET, POST, PATCH /:projectId, DELETE /:projectId)
- `/api/assistant/files/:fileId/download` — File download
- `/api/:agentId/sessions` — Session management (GET list, POST create, POST /:sessionId/edit, PATCH /:sessionId/project, GET /:sessionId/history, DELETE /:sessionId)
- `/api/email_server_configs` — Email server config CRUD
- `/api/ssh_server_configs` — SSH server config CRUD

Public routes: `/api/health`, `/api/auth/*`, `/api/share/:shareId`

## Code Style Guidelines

### Go Code
Follow [Effective Go](https://go.dev/doc/effective_go) and [Google Go Style Guide](https://google.github.io/styleguide/go/).

**Conventions:**
- Use `log/slog` for structured logging: `slog.Error("message", "err", err)`
- Place main/entry functions above helper functions they call
- Tool handlers return `(*Output, error)` for ADK integration

**Error Handling and Logging:**
- **At error source** (stdlib/3rd-party call): MUST `slog.Error()` with full context, then return wrapped error
- **At error propagation** (internal calls): DO NOT log — just `fmt.Errorf("...: %w", err)`
- Log once at the source; never log the same error multiple times
- Exception: GORM database errors should always log at source

```go
// At source:
resp, err := http.DefaultClient.Do(req)
if err != nil {
    slog.Error("http.DefaultClient.Do() error", "url", url, "err", err)
    return nil, fmt.Errorf("请求失败: %w", err)
}

// At propagation:
result, err := someInternalFunction()
if err != nil {
    return nil, fmt.Errorf("failed to process: %w", err)
}
```

**Dependencies:** Core deps are `google.golang.org/adk` and `google.golang.org/genai`. Add ADK sub-packages via `go get google.golang.org/adk/...`.

### TypeScript/React
- TypeScript with strict type checking; functional components with hooks
- Tailwind CSS for styling; ESLint for linting
- Use `import type` for type-only imports
- Use `@/` path alias for frontend-local imports
- Prefer `Record<string, unknown>` over `any` for unknown object payloads
- Reuse shared UI wrappers from `frontend/app/components/ui/` before adding new primitives

## Important Implementation Notes

### Adding New Tools
1. Create tool in `internal/pkg/tools/` with ADK `functiontool` pattern
2. Define Input/Output structs with JSON schema tags
3. Register in `agent.go` via `NewAssistantAgent`
4. For DB/auth context, use middleware utilities from `internal/pkg/middleware/`

### Modifying Agent Behavior
- Root agent system prompt: `internal/app/aiguide/assistant/assistant_agent_prompt.md`
- Planner prompt: `internal/app/aiguide/assistant/planner_agent_prompt.md`
- Model config: Change in YAML (`model_name`, `api_key`)

## Testing

Backend tests across `internal/app/aiguide/assistant`, `internal/app/aiguide`, `internal/pkg/tools`, `internal/pkg/auth`, `internal/pkg/storage`, `internal/pkg/middleware`.

Frontend tests use Vitest with jsdom; setup in `frontend/test/setup.ts`. Test files co-located under `frontend/app/**/*.test.{ts,tsx}`.

```bash
go test ./...                                              # All backend tests
go test -v ./internal/app/aiguide/assistant               # Assistant package
go test -v ./internal/pkg/tools                           # Tools package
cd frontend && pnpm test                                   # All frontend tests
cd frontend && pnpm test -- <path/to/file.test.tsx>        # Single file
cd frontend && pnpm test -- -t "test name"                 # Filter by name
```

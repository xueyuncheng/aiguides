# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

AIGuides is a full-stack AI assistant built with Go (Gin + Google ADK/Gemini) backend and Next.js frontend. It supports multimodal chat (text + images), image generation via Imagen, email querying via IMAP, web search/fetch, session management, and Google OAuth authentication.

**Tech Stack:**
- Backend: Go 1.25.5+, Gin, GORM, SQLite, Google ADK
- Frontend: Next.js 15, React 19, TypeScript, Tailwind CSS
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
# Build images
make build                # Build both
make build-backend        # Backend only
make build-frontend       # Frontend only

# Deploy with docker-compose
make deploy

# Stop services
make down
```

## Architecture

### Backend Structure

**Entry Point:**
- `cmd/aiguide/aiguide.go` - Application entry, loads YAML config and starts service

**Core Application (`internal/app/aiguide/`):**
- `aiguide.go` - Service initialization, config struct with YAML tags
- `router.go` - API routes (auth, sessions, assistant chat, settings)
- `login.go` - Google OAuth handlers
- `user.go`, `avatar.go` - User management
- `migration/` - Database auto-migration using GORM
- `table/` - Database table models
- `setting/` - Settings API (email server configs)

**Assistant Runtime (`internal/app/aiguide/assistant/`):**
- `agent.go` - Builds ADK agent with tools using `llmagent.New`
- `assistant_agent_prompt.md` - Embedded system prompt
- `runner.go` - Runner initialization and management
- `sse.go` - SSE streaming chat handler with multimodal support
- `session.go` - Session CRUD APIs (create, list, delete, history)
- `handler.go` - Gin route handlers

**Tools (`internal/pkg/tools/`):**
- `imagegen.go` - Imagen tool (supports mock mode, aspect ratios, multiple images)
- `email.go` - IMAP email query tool (requires user email server config)
- `websearch.go` - Web search tool integration
- `webfetch.go` - Web page fetching/scraping tool

**Infrastructure (`internal/pkg/`):**
- `auth/` - Google OAuth + JWT cookie implementation
- `middleware/` - Auth middleware, context utilities
- `constant/` - Shared constants

### Frontend Structure

**Pages (`frontend/app/`):**
- `page.tsx` - Landing page
- `login/` - Google OAuth login flow
- `chat/` - Main chat interface with multimodal input
- `settings/` - Email server configuration UI
- `components/` - Shared React components
- `contexts/` - React contexts for state management
- `lib/` - Utility functions

### Key Patterns

**Agent Tool Registration:**
Tools are registered in `internal/app/aiguide/assistant/agent.go`:
```go
agent, err := llmagent.New(llmagent.Config{
    Name: "root_agent",
    Model: model,
    Tools: []tool.Tool{
        imageGenTool,
        emailQueryTool,
        webSearchTool,
        webFetchTool,
    },
    SystemPrompt: assistantAgentInstruction,
})
```

**SSE Streaming:**
Chat responses stream via Server-Sent Events in `sse.go`:
- Supports text + image multipart input
- Validates image size (5MB max), count (4 max), and MIME types
- Streams agent responses with tool calls and results
- Saves messages to database for session history

**Database:**
- SQLite with GORM
- Auto-migration on startup via `migration/migration.go`
- Models defined in `internal/app/aiguide/table/`
- Tables: User, Session, Message, EmailServerConfig

**Authentication:**
- Google OAuth flow: `/api/auth/login/google` → `/api/auth/callback/google`
- JWT tokens in HTTP-only cookies
- Refresh token endpoint: `/api/auth/refresh`
- Email whitelist support via `allowed_emails` config
- Auth middleware protects all routes except public endpoints

**Configuration:**
- YAML-based config at `cmd/aiguide/aiguide.yaml`
- Example: `cmd/aiguide/aiguide.yaml.example`
- Required: `api_key`, `model_name`
- Optional: OAuth credentials, JWT secret, allowed emails, mock settings

## Code Style Guidelines

### Go Code
Follow Effective Go (https://go.dev/doc/effective_go) and Google Go Style Guide (https://google.github.io/styleguide/go/).

**Specific conventions:**
- Use `log/slog` for structured logging: `slog.Error("message", "err", err)`
- Place main/entry functions above helper functions they call
- Error handling: `if err != nil { slog.Error(...); return fmt.Errorf("..."); }`
- Tool handlers return `(*Output, error)` for ADK integration

**Dependency Management:**
- Add ADK sub-packages as needed: `go get google.golang.org/adk/...`
- Core deps: `google.golang.org/adk`, `google.golang.org/genai`

### TypeScript/React
- Use TypeScript with strict type checking
- Functional components with hooks
- Tailwind CSS for styling
- ESLint for linting

## Important Implementation Notes

### Adding New Tools
1. Create tool in `internal/pkg/tools/` with ADK `functiontool` pattern
2. Define Input/Output structs with JSON schema tags
3. Register in `agent.go` via `NewAssistantAgent`
4. For DB/auth context, use middleware utilities

### Modifying Agent Behavior
- System prompt: `internal/app/aiguide/assistant/assistant_agent_prompt.md`
- Model config: Change in YAML (`model_name`, `api_key`)
- Tool availability: Register/unregister in `agent.go`

### Database Changes
- Add models to `internal/app/aiguide/table/`
- Include in `GetAllModels()` for auto-migration
- Migration runs automatically on startup

### Frontend API Integration
- API base: `/api` prefix
- Auth: JWT cookie automatically sent
- SSE endpoint: `/api/assistant/chats/:id` (POST with EventSource polyfill)
- Session APIs: `/api/:agentId/sessions/*`

## Testing

The project has 12 test files covering:
- Agent creation and tool registration
- SSE streaming logic
- Session management
- OAuth flow
- Router configuration
- Tool implementations

Run all tests: `go test ./...`
Run specific package: `go test -v ./internal/app/aiguide/assistant`

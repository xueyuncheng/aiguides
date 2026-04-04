# AGENTS.md

This file is for coding agents working in `aiguide`.
It summarizes the commands, architecture touchpoints, and coding conventions that are already used in this repository.

## Project Summary

- Full-stack AI assistant.
- Backend: Go 1.25.5+, Gin, GORM, SQLite, Google ADK, Gemini, Imagen.
- Frontend: Next.js 16, React 19, TypeScript, Tailwind CSS 4.
- Main backend entrypoint: `cmd/aiguide/aiguide.go`.
- Main frontend app: `frontend/app/`.

## Rule Sources

- Primary repo guidance exists in `CLAUDE.md`.
- Copilot instructions exist in `.github/copilot-instructions.md` and should be treated as active guidance.
- No `.cursorrules` file was found.
- No `.cursor/rules/` directory was found.

## Common Commands

### Backend

- Run backend locally:
  `go run cmd/aiguide/aiguide.go -f cmd/aiguide/aiguide.yaml`
- Format Go code:
  `go fmt ./...`
- Format Go code via Makefile:
  `make fmt`
- Run all Go tests:
  `go test ./...`
- Run verbose tests for one package:
  `go test -v ./internal/app/aiguide/assistant`
- Run one Go test by name:
  `go test -v ./internal/app/aiguide/assistant -run TestNewSearchAgent`
- Run one Go test in another package:
  `go test -v ./internal/pkg/tools -run TestWebFetch`
- Re-run a single failing test without cache:
  `go test -count=1 -v ./internal/app/aiguide/assistant -run TestName`

### Frontend

- Install dependencies:
  `cd frontend && pnpm install`
- Run Next.js dev server:
  `cd frontend && pnpm dev`
- Build frontend:
  `cd frontend && pnpm build`
- Start production frontend:
  `cd frontend && pnpm start`
- Lint frontend:
  `cd frontend && pnpm lint`
- Run all frontend tests:
  `cd frontend && pnpm test`
- Run a single frontend test file:
  `cd frontend && pnpm test -- app/chat/[sessionId]/hooks/useStreamingChat.test.tsx`
- Run frontend tests matching one test name:
  `cd frontend && pnpm test -- -t "keeps /chat for draft sessions until the first message is sent"`
- Run a single file and one named test:
  `cd frontend && pnpm exec vitest run app/chat/[sessionId]/hooks/useStreamingChat.test.tsx -t "keeps /chat for draft sessions until the first message is sent"`

### Full Stack and Docker

- Start backend and frontend together:
  `./scripts/start.sh`
- Build Docker images:
  `make build`
- Build backend image only:
  `make build-backend`
- Build frontend image only:
  `make build-frontend`
- Deploy docker compose stack:
  `make deploy`
- Stop docker compose stack:
  `make down`

## What Counts As Linting Here

- Go: `go fmt ./...` is configured and expected.
- Frontend: ESLint is configured in `frontend/eslint.config.mjs`.
- TypeScript: strict mode is enabled in `frontend/tsconfig.json`.
- No dedicated Go lint command such as `golangci-lint` is configured in the repository today.

## Test Locations

- Backend tests are standard Go `_test.go` files under `internal/`.
- Frontend tests use Vitest and currently live under `frontend/app/**/*.test.{ts,tsx}`.
- Representative backend packages with tests:
  `internal/app/aiguide/assistant`, `internal/app/aiguide`, `internal/pkg/tools`, `internal/pkg/auth`, `internal/pkg/storage`, `internal/pkg/middleware`.

## Architecture Pointers

- Service bootstrap: `cmd/aiguide/aiguide.go`
- App wiring and config: `internal/app/aiguide/aiguide.go`
- Router: `internal/app/aiguide/router.go`
- Assistant runtime: `internal/app/aiguide/assistant/`
- Tool implementations: `internal/pkg/tools/`
- Auth and middleware: `internal/pkg/auth/`, `internal/pkg/middleware/`
- Frontend chat UI: `frontend/app/chat/[sessionId]/`
- Shared UI primitives: `frontend/app/components/ui/`

## Backend Style Guidelines

- Follow Effective Go and the Google Go Style Guide.
- Use `gofmt` formatting; do not hand-format Go files.
- Keep imports grouped the normal Go way: standard library first, then a blank line, then third-party and internal imports as `gofmt` arranges them.
- Package names are short, lowercase, and directory-based.
- Exported Go names use PascalCase.
- Unexported names use camelCase.
- Prefer small, direct functions over unnecessary abstractions.
- Place main or entry functions above helper functions they call.
- Keep helpers near the code that uses them.
- Use concrete structs for tool input/output types.
- Tool handlers typically return `(*Output, error)`.
- Keep JSON field names explicit with struct tags.
- JSON schema descriptions are used on tool inputs where needed.

## Backend Error Handling and Logging

- Use `log/slog` for structured logging.
- Log errors at the source of failure with relevant context keys.
- Do not log the same error repeatedly as it propagates upward.
- At propagation boundaries, wrap and return errors with `fmt.Errorf("...: %w", err)`.
- GORM and other direct I/O failures should be logged where they occur.
- Validation failures often return structured tool outputs instead of hard errors; match the existing pattern in the surrounding package.
- Include useful identifiers in logs such as `user_id`, `session_id`, `file_id`, URL, path, or status code.

## Backend Naming and API Conventions

- Route handlers are methods on app or assistant types.
- Constructor names usually follow `NewX`.
- Config structs use explicit field names and YAML tags where relevant.
- Route registration is centralized in `router.go`.
- New database models belong under `internal/app/aiguide/table/` and should be added to the migration/model registration flow.
- New agent tools belong under `internal/pkg/tools/` and must be registered from the assistant agent setup.

## Frontend Style Guidelines

- Use TypeScript with `strict` mode assumptions.
- Prefer functional React components and hooks.
- Keep component props explicitly typed.
- Use `import type` for type-only imports.
- Use the `@/` path alias for frontend-local imports when appropriate.
- Reuse shared UI wrappers from `frontend/app/components/ui/` before adding new primitives.
- Tailwind CSS is the primary styling mechanism.
- Follow the established component structure in `frontend/app/chat/[sessionId]/components/` and hooks in `.../hooks/`.
- Keep stateful logic in hooks when it materially improves reuse or readability.
- Avoid introducing class name helpers or abstractions unless the surrounding code already needs them.

## Frontend Formatting and Linting

- ESLint is the active frontend lint tool.
- Match the quote style already used in the file you are editing; the repo currently contains both single-quoted and double-quoted TS/TSX files.
- Keep imports organized and remove unused ones.
- Prefer readable inline types or local aliases over overly broad `any`.
- Prefer `Record<string, unknown>` over `any` for unknown object payloads when that matches current usage.

## Frontend Testing Conventions

- Frontend tests use Vitest with `jsdom`.
- Test helpers are loaded from `frontend/test/setup.ts`.
- Name tests with clear behavior-oriented descriptions.
- Prefer focused hook/component tests near the feature directory.

## Agent Workflow Expectations

- Read the local area of the codebase before changing it.
- Make the smallest correct change.
- Preserve existing architecture and naming unless there is a clear reason to refactor.
- Do not add backward-compatibility code unless the repository already needs it.
- Do not invent new commands or tooling that are not present in the repo.
- If you add a tool, wire it into the assistant agent and any required middleware/config paths.
- If you add a model or table, ensure migrations/registration are updated.
- If you change frontend API usage, keep `/api` routing and the existing auth/session flow intact.

## Copilot Instructions Incorporated

- Respect the architecture split between backend app wiring, assistant runtime, tools, auth, and frontend pages.
- Use `log/slog` for backend logging.
- Keep main functions above helpers.
- Use Effective Go and Google Go style as the default backend standard.

## Practical Notes

- Backend config lives at `cmd/aiguide/aiguide.yaml`; there is also `cmd/aiguide/aiguide.yaml.example`.
- `./scripts/start.sh` expects Go, Node.js, `pnpm`, and the YAML config file to exist.
- Frontend API calls are proxied through `/api`.
- Shared public conversations are exposed via `/api/share/:shareId`.

When unsure, prefer the existing pattern in the nearest file over generic framework advice.

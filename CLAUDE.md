# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

@openspec

## Project Overview

Metis is a Go 1.26 web application with an embedded React frontend. It compiles to a single binary that serves both API and static assets. Backend uses Gin + GORM + samber/do (IOC). Frontend uses Vite 8 + React 19 + TypeScript 6 + React Compiler.

## Build, Run & Test Commands

```bash
# Development (run in two separate terminals)
make dev              # Go server on :8080 with -tags dev (no frontend embed needed)
make web-dev          # Vite dev server on :3000, proxies /api → :8080

# First-time setup
make web-install      # Install frontend dependencies (bun)

# Build & release
make build            # Build frontend + compile single binary (./server)
make run              # build + run
make release          # Cross-compile for linux/darwin/windows (amd64+arm64) → dist/

# Edition builds (modular — see Architecture section)
make build EDITION=edition_lite APPS=system      # Kernel only
make build APPS=system,ai                        # Kernel + AI
make build-license                               # License edition (./license)
make release-license                             # Cross-compile license edition

# Sidecar (remote agent execution binary)
make build-sidecar    # Build sidecar binary (./sidecar)
make release-sidecar  # Cross-compile sidecar → dist/

# Frontend only
cd web && bun run build    # Production build
cd web && bun run lint     # ESLint (includes React Compiler rules)
cd web && bun run preview  # Preview production build locally
```

**Go build verification**: `go build -tags dev ./cmd/server/` checks compilation without building the frontend (the `dev` tag provides an empty embed FS).

**Tests**:
```bash
go test ./...                                    # Run all Go tests
go test ./internal/app/ai -run TestName -v      # Run a single test
make test-license                                # License app tests only
make test-fuzz                                   # Fuzz tests (license crypto, 30s each)
make test-bdd                                    # ITSM BDD tests
make test-cover                                  # Coverage report → coverage.html
make test-pretty                                 # Pretty output (requires gotestsum)
make test-llm                                    # LLM integration tests (needs .env.test)
```
Tests exist in `internal/service/` (kernel), `internal/app/license/`, `internal/app/itsm/`, and `internal/app/ai/`. No frontend tests yet.

**Package manager**: Frontend uses **bun** (`bun install`, `bun run dev`, `bun run build`).

## Architecture

### Kernel

Layered backend with dependency injection (samber/do v2):

```
cmd/server/main.go → IOC container → Gin engine + middleware
    ↓
internal/handler/    → HTTP handlers, unified response R{code,message,data}
internal/service/    → Business logic, sentinel errors (ErrUserNotFound, etc.)
internal/repository/ → GORM data access, ListParams/ListResult pagination
internal/model/      → Domain structs (BaseModel, SystemConfig K/V table)
internal/config/     → config.yml (MetisConfig)
internal/database/   → GORM init, SQLite (default) + PostgreSQL
internal/middleware/ → JWT auth, Casbin RBAC, DataScope, Audit slog logging
internal/scheduler/  → Cron + async queue engine, GORM-backed
internal/seed/       → Install() first-time setup, Sync() incremental on restart
```

All dependencies are registered as `do.Provide()` providers in `main.go` and resolved lazily.

### Pluggable App Architecture

```
┌──────────────────────────────────┐
│           Kernel（内核）          │  ← users/roles/menus/auth/settings/tasks/audit
│         始终存在，不可拔除          │
└──────────────┬───────────────────┘
               │
  ┌────┬────┬──┴──┬────┬────┬────┐
  ▼    ▼    ▼     ▼    ▼    ▼    ▼
 AI  Node  Org   APM  Obs  License  ITSM  ← optional Apps, build-tag controlled
```

Each App implements `app.App` (`internal/app/app.go`):

```go
type App interface {
    Name() string
    Models() []any              // GORM AutoMigrate
    Seed(db *gorm.DB, enforcer *casbin.Enforcer) error
    Providers(i do.Injector)    // IOC registration
    Routes(api *gin.RouterGroup)// JWT+Casbin+Audit middleware already applied
    Tasks() []scheduler.TaskDef // nil if none
}
```

Startup order in `main.go`: `Models → Providers → Seed → Routes → Tasks`.

**Adding a new App**:
1. Backend: `internal/app/<name>/app.go` implementing `App` + `func init() { app.Register(&XxxApp{}) }`
2. Edition file: add `import _ "metis/internal/app/<name>"` to `cmd/server/edition_full.go`
3. Frontend: `web/src/apps/<name>/module.ts` calling `registerApp()`
4. Bootstrap: add import to `web/src/apps/_bootstrap.ts` (`gen-registry.sh` manages this during filtered builds)

Apps can resolve kernel services via `do.MustInvoke[*service.UserService](i)`.

**Existing editions**:
- `edition_full.go` — default, all Apps
- `edition_lite.go` (`edition_lite`) — kernel only
- `edition_license.go` (`edition_license`) — kernel + License App

### Middleware Chain

Authenticated routes use this fixed chain (configured in `handler.Register()`):

```
JWTAuth → PasswordExpiry → CasbinAuth → DataScope → Audit → Handler
```

- **CasbinAuth whitelist**: `middleware/casbin.go` defines `casbinWhitelist` (exact) and `casbinWhitelistPrefixes` (prefix) for public routes. Add new public APIs here.
- **Audit**: Only 2xx responses are logged. Handlers set fields via `c.Set()`: `audit_action`, `audit_resource`, `audit_resource_id`, `audit_summary`.
- **DataScope**: `c.Get("deptScope")` returns `*[]uint` — `nil`=all visible, `&[]uint{}`=self only, `&[]uint{1,2,3}`=specific departments. Requires Org App; no-op otherwise.

### Auth & RBAC

JWTAuth extracts `UserID` + `Role` → CasbinAuth enforces `enforce(roleCode, path, method)` using `keyMatch2` for wildcards.

- **JWT**: Access token 30 min, refresh token 7 days. `TokenClaims` has a `purpose` field for 2FA tokens.
- **OAuth 2.0**: Google & GitHub built-in (`internal/pkg/oauth/`).
- **Sessions**: Active session list + admin kick via token blacklist.

### Scheduler Engine

`Engine.Register(taskDef)` before `Start()`. Tasks are either scheduled (cron) or async (queue polled every 3s). Handler signature: `func(ctx context.Context, payload json.RawMessage) error`. Default timeout 30s, retries 3.

Built-in kernel tasks: `scheduler-history-cleanup`, `blacklist-cleanup`, `expired-token-cleanup`, `audit-log-cleanup`.

### AI App: Knowledge Module

Two-phase pipeline in `internal/app/ai/`:

```
KnowledgeBase         → compile status: idle/compiling/completed/error
  └─ KnowledgeSource  → file/URL/text, extract status
  └─ KnowledgeNode    → concept / index nodes
  └─ KnowledgeEdge    → related/contradicts/extends/part_of
  └─ KnowledgeLog     → compilation logs
```

Scheduler tasks:
- `ai-source-extract` (async) — text extraction. md/txt handled immediately; PDF/DOCX/XLSX/PPTX currently return error (TODO).
- `ai-knowledge-crawl` (cron) — re-crawls URL sources by `CrawlSchedule`, triggers recompile on content change.
- `ai-knowledge-compile` (async) — LLM compilation of all `completed` sources into graph, then auto `index` node generation + lint.

LLM client is in `internal/llm/`, supporting OpenAI-compatible and Anthropic protocols via Provider config (protocol + BaseURL + encrypted API Key).

URL crawling supports `crawlDepth` (same-domain recursion) and `urlPattern` (prefix filter). HTML → Markdown via regex, 10MB limit.

### AI App: Agent Runtime

```
Agent
  ├─ AgentTypeAssistant  → ReAct / Plan-and-Execute
  ├─ AgentTypeCoding     → local or remote execution
  └─ AgentTemplate
AgentSession
  └─ SessionMessage
AgentMemory              → long-term memory extraction
```

- **ReAct**: think-act-observe loop
- **Plan-and-Execute**: plan first, then execute
- **Coding Local**: direct CLI invocation (`executor_coding_local.go`)
- **Coding Remote**: via Node/Sidecar
- **Tools**: Tool Registry, MCP Server, Skill (Git-repo skills importable and bindable to agents)

Sessions use SSE streaming (`GET /api/v1/ai/sessions/:sid/stream`).

### Node App: Sidecar Architecture

```
Node
  ├─ NodeProcess    → bound process instance
  └─ ProcessDef     → image, config template
NodeCommand         → start/stop/restart/reload
NodeProcessLog      → log collection
```

- Sidecar runs as a separate binary (`cmd/sidecar`)
- Server→Node commands via SSE `/api/v1/nodes/sidecar/stream`
- Node polling via `GET /api/v1/nodes/sidecar/commands`
- REST for heartbeat, log upload, config download
- Auth: `X-Node-Token` header (not JWT)

Node Token APIs (bypass JWT+Casbin):
- `/api/v1/ai/knowledge/*` — knowledge queries
- `/api/v1/ai/internal/skills/*` — skill package downloads

### Org App

Department tree (self-referencing `parent_id`), Position definitions, UserPosition assignments (many-to-many with primary/secondary). Implements `OrgScopeResolver` for `DataScopeMiddleware`.

### APM App

No own models (`Models()` returns nil). Queries external ClickHouse for OpenTelemetry trace/span data. `NewClickHouseClient` reads connection config from DB `SystemConfig`.

### Observe App

ForwardAuth for external observability tools (e.g., Grafana). `IntegrationToken` with hashed token + scopes. `/api/v1/observe/auth/verify` is registered directly on the Gin engine, bypassing JWT+Casbin.

### License App

License lifecycle management with Ed25519 signing + AES-GCM encryption. Supports key rotation with versioned keys and bulk reissue. Can be compiled standalone via `edition_license`.

### ITSM App

IT Service Management with dual workflow engines:

```
ServiceCatalog → ServiceDefinition → ServiceAction (workflow steps)
  └─ FormDefinition, Priority, SLATemplate, EscalationRule
Ticket → TicketActivity → TicketTimeline
  └─ ProcessVariable, ExecutionToken
  └─ TicketLink, PostMortem (incidents)
```

- **ClassicEngine**: BPMN-style workflow with nodes (user_task, action, gateway, wait_timer, approval), token-based execution, and boundary timers
- **SmartEngine**: AI-powered workflow that uses an Agent to make decisions, resolve participants, and search knowledge bases
- **Tool Registry** (`tools/`): Exposes ITSM operations (ticket CRUD, assignment, state transitions) as AI Agent tools via `app.ToolRegistryProvider`
- Scheduler tasks: `itsm-action-execute`, `itsm-wait-timer`, `itsm-smart-progress`, `itsm-boundary-timer`, `itsm-doc-parse`

### i18n

- **Backend**: `internal/locales/` using go-i18n. Apps can implement `LocaleProvider` to supply extra JSON translation files.
- **Frontend**: `web/src/i18n/` using i18next. App translations live in `web/src/apps/<name>/locales/` and are registered via `registerTranslations(ns, resources)` in `module.ts`. Fallback is `zh-CN`.

### Frontend Stack

```
apps/          → pluggable app modules
pages/         → kernel pages (users, roles, menus, settings, etc.)
stores/        → Zustand (auth, menu, ui), hydrated from localStorage
components/    → shadcn/ui + DashboardLayout
lib/api.ts     → centralized HTTP client with auto token refresh + 401 queueing
hooks/         → useListPage (pagination + react-query), usePermission
```

- **State**: Zustand (client state) + React Query (server state, staleTime 30s)
- **Routing**: React Router 7, lazy-loaded, AuthGuard + PermissionGuard
- **UI**: Tailwind CSS 4, shadcn/ui, Lucide icons
- **Forms**: React Hook Form + Zod
- **表单容器**: 新建/编辑表单统一使用 Sheet（抽屉），不要用 Dialog（弹窗）

## React Compiler Constraints

Enabled via `babel-plugin-react-compiler` and strict `eslint-plugin-react-hooks`. These patterns will fail build or crash at runtime with "Rendered fewer hooks than expected":

1. **No early return before hooks** — all `useState`, `useEffect`, `useCallback`, `useMemo` must be called before any conditional return
2. **No setState inside useEffect** — cannot synchronously call `setState` inside an effect
3. **No ref read/write during render** — `ref.current` only in event handlers or effects
4. **No IIFE** — immediately invoked function expressions break the compiler

Example of correct pattern:
```tsx
function MyComponent({ data }) {
  const [state, setState] = useState(false)
  useEffect(() => { ... }, [])
  if (!data) return null  // early return after all hooks
  return <div>...</div>
}
```

## Key Conventions

- **API prefix**: `/api/v1/*`
- **Response format**: `handler.OK(c, data)` / `handler.Fail(c, status, msg)` → `{"code":0,"message":"ok","data":...}`
- **Database**: SQLite default (pure Go, CGO_ENABLED=0) or PostgreSQL. SQLite DSN uses `_pragma=journal_mode(WAL)`.
- **Configuration**: `config.yml` stores infrastructure config (db_driver, db_dsn, secret_key, jwt_secret, license_key_secret). Everything else (server_port, OTel, site.name) lives in DB `SystemConfig`. No `.env` file.
- **Install wizard**: On first run (no `config.yml`), the server enters install mode serving only `/api/v1/install/*` + SPA. Frontend at `/install` guides DB selection → site info → admin creation. After install, hot-switches to normal mode.
- **Seed pattern**: `seed.Install()` runs once. `seed.Sync()` runs on every subsequent startup (incremental — adds missing roles/menus/policies only, never overwrites existing SystemConfig). Idempotency checks use `db.Where("permission = ?", x).First(&existing)` and `enforcer.HasPolicy()`.
- **New kernel models**: Add struct in `internal/model/`, register in `database.go` AutoMigrate, create repo → service → handler, wire into IOC in `main.go` via `do.Provide()`.
- **New app models**: Return in the App's `Models()` method; `main.go` handles AutoMigrate.
- **BaseModel**: Embed `model.BaseModel` for auto ID + timestamps + soft delete. `SystemConfig` uses `Key` as PK.
- **ToResponse pattern**: Models expose `.ToResponse()` to strip sensitive fields (e.g., User hides password hash).
- **Pagination**: Backend `ListParams{Keyword, IsActive, Page, PageSize}` → `ListResult{Items, Total}`. Frontend `useListPage` wraps React Query.
- **Error handling**: Services define package-level sentinel errors; Handlers match with `errors.Is()` to map to HTTP status codes.
- **Static embedding**: `embed.go` embeds `web/dist/`. `embed_dev.go` (`//go:build dev`) provides empty FS for dev mode. SPA fallback serves `index.html`.
- **Frontend alias**: `@/` maps to `web/src/` in Vite and TS configs.
- **Route registration**: `handler.Register()` returns a `*gin.RouterGroup` already wrapped with JWT+Casbin+Audit middleware. App routes mount under this group.

## Do Not Modify

- `refer/` and `support-files/refer/` — user's reference code, never modify
- `next-app/` — separate Next.js experiment, not part of the main app
- `openspec/` — spec-driven development artifacts, managed via `/opsx:*` commands

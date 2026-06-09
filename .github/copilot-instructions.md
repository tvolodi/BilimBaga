# BilimBaga — Copilot Project Instructions

## What This App Does

**BilimBaga** is a corporate exam platform. It enables HR and department admins to create multilingual question banks, configure timed exams, assign them to employees, auto-grade results, issue verifiable certificates, and analyse performance across departments.

## Repository Layout

```
BilimBaga/
  backend/            Go 1.22 API (port 8080)
    cmd/api/          entry point
    internal/         domain packages per CLAUDE.md layout
    migrations/       numbered SQL files
  frontend/           React 18 + TypeScript SPA (port 5173)
    src/
      api/            React Query hooks + fetch wrappers
      components/     shared shadcn/ui-based components
      pages/          route-level pages
      locales/        i18n JSON (kk, ru, en)
  deploy/             docker-compose.yml, nginx.conf
  docs/               architecture, requirements, reports
  .github/
    agents/           custom Copilot workflow agents
    instructions/     auto-applied convention files
  corporate_exam_platform_roadmap.md   full product spec
```

## Stack

- **Backend**: Go 1.22 · Chi router · sqlx · golang-migrate · bcrypt · golang-jwt/jwt/v5
- **Frontend**: React 18 · TypeScript · Vite · Tailwind CSS · shadcn/ui · TanStack Query · React Router v6 · react-i18next
- **Database**: PostgreSQL 16 (schema-per-tenant ready)
- **Infrastructure**: Docker Compose (db · api · frontend · nginx)
- **AI** (Phase 7): Anthropic Claude via anthropic-sdk-go

## Non-Negotiable Conventions

- **Go**: one package per domain — no circular imports; handlers thin; SQL in repository layer.
- **Go**: typed `Config` struct loaded at startup — never `os.Getenv()` inside handlers.
- **Go**: all errors wrapped with context; never swallow silently.
- **Frontend**: React Query for all server state — no raw `fetch` in components.
- **Frontend**: zero hardcoded user-visible strings — all text in `src/locales/{locale}.json`.
- **Migrations**: never edit existing migration files — always add new numbered files.
- **Secrets**: never in source code or committed `.env`.
- **API contract**: `{ data, error: null }` on success; `{ data: null, error: { code, message } }` on failure.
- **IDs**: UUID v4. **Timestamps**: UTC ISO 8601.

## Dev Commands

```bash
make dev              # starts all Docker Compose services
make migrate          # runs pending migrations
cd backend && go run ./cmd/api        # API only (port 8080)
cd frontend && npm run dev            # SPA only (port 5173)
cd backend && go test ./...           # backend tests
cd frontend && npm test               # frontend tests
```

## Key Docs (read before large changes)

- `corporate_exam_platform_roadmap.md` — full phase-by-phase spec with all tables, endpoints, UI screens
- `docs/architecture-guide.md` — service diagram, DB schema overview, API conventions
- `docs/backend-development-guide.md` — Go patterns, error handling, middleware
- `docs/frontend-development-guide.md` — React Query, shadcn/ui, i18n, routing
- `docs/requirements/README.md` — FR-BBxxx index

## Workflow Agents

For structured, multi-step work, use the custom agents in `.github/agents/`:

| Agent | Use when |
|-------|---------|
| `requirement-development` | Defining a new feature or requirement from scratch |
| `requirement-validation` | Validating a requirement doc before implementation |
| `requirement-implementation` | Implementing a defined and validated requirement |
| `issue-resolution` | Investigating and fixing a reported bug |
| `test-run-error-resolution` | Running the full test suite and fixing failures |
| `infrastructure-configuration` | Setting up env, Docker, migrations, CORS, credentials |
| `release-preparation` | Pre-release quality gate and changelog |
| `04-code-reviewer` | Reviewing code for quality, security, AC coverage |
| `05-code-fixer` | Fixing Code Reviewer findings |
| `06-release-finalizer` | Git commit + inner report |
| `explore` | Read-only codebase exploration (subagent only) |
| `business-analyst` | Define business processes (Mode A), author UAT scenario scripts (Mode B), make UAT pass/fail decisions (Mode C) |
| `uat-runner` | Execute UAT scenario scripts against the live GUI using Playwright + browser tool; produce UAT reports |

### When to use Business Analyst + UAT Runner

- **"Define business process for X"** → `business-analyst` (Mode A) → `requirement-development` → Pipeline A
- **"Run UAT for X"** / **"Verify X as a user"** → `business-analyst` (Mode B) → `uat-runner` → `business-analyst` (Mode C)
  - Prerequisite: stack must be running (`make dev`)
  - UAT Runner uses Playwright for standard interactions; browser screenshots for visual/ambiguous steps
  - Results written to `docs/uat-reports/{run-id}.md`; defects routed to `issue-resolution`

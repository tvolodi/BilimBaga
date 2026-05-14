# FR-BB11 — Project Scaffold

## Metadata
| Field | Value |
|-------|-------|
| ID | FR-BB11 |
| Phase | 1 — Foundation |
| Priority | 1 |
| Status | implemented |
| Depends On | — |

## Scope

| Layer | Changes |
|-------|---------|
| Infrastructure | `docker-compose.yml` with four services (`db`, `api`, `frontend`, `nginx`); named volumes; healthcheck on `db` |
| Backend (Go) | Go module init; Chi router; sqlx pool; golang-migrate wiring; typed `Config` struct; `GET /api/v1/health` handler; multi-stage Dockerfile |
| Frontend (React/TS) | Vite + React 18 + TypeScript project init; Tailwind CSS config; shadcn/ui primitives; `App.tsx` / `main.tsx` entry points; two-stage Dockerfile |
| DevOps/CI | `Makefile` with `dev`, `build`, `migrate`, `test` targets; `.env.example` with all required variables; `nginx.conf` with API proxy and SPA fallback |

## Description
Establishes the complete project skeleton for BilimBaga, including all four Docker Compose services (`db`, `api`, `frontend`, `nginx`), the Go backend module wired with Chi, sqlx, and golang-migrate, and the React 18 + TypeScript frontend wired with Vite, Tailwind CSS, and shadcn/ui. This requirement has no business logic; it delivers the development environment and CI-ready build chain that all subsequent requirements build upon.

## Acceptance Criteria
- [x] AC-1: `docker-compose.yml` defines four services — `db` (PostgreSQL 16), `api` (Go), `frontend` (Node build + static), `nginx` (reverse proxy) — and all four start without error via `make dev`.
- [x] AC-2: Go module is initialised (`go.mod` / `go.sum`) with `github.com/go-chi/chi/v5`, `github.com/jmoiron/sqlx`, `github.com/golang-migrate/migrate/v4`, and `github.com/golang-jwt/jwt/v5` as direct dependencies.
- [x] AC-3: Frontend project is initialised with Vite + React 18 + TypeScript; `npm run dev` starts the dev server on port 5173 without errors.
- [x] AC-4: Tailwind CSS is configured and a `<Button>` component imported from `@/components/ui/button` renders without TypeScript compilation errors and a clickable button element is present in the DOM when the app is opened at `http://localhost:5173`.
- [x] AC-5: Nginx config routes all `/api/*` requests to the Go service on port 8080 and serves the compiled frontend static files for all other paths.
- [x] AC-6: `.env.example` documents every required environment variable with a descriptive comment; no variable has a production secret as its default.
- [x] AC-7: `Makefile` exposes at minimum four targets: `dev` (starts Compose), `build` (builds Go binary + frontend bundle), `migrate` (runs pending migrations), `test` (runs backend and frontend tests).
- [x] AC-8: `go build ./...` and `npm run build` both complete without errors or warnings in CI (clean environment, no pre-installed caches).
- [x] AC-9: Health-check endpoint `GET /api/v1/health` returns `200 OK` with `{"data":{"status":"ok"},"error":null}` when the API service is running.
- [x] AC-10: API process exits with a non-zero exit code and logs a descriptive error if `JWT_SECRET` is absent or shorter than 32 characters.

## Out of Scope

- Authentication, login/logout, JWT issuance — covered by FR-BB14 and FR-BB15.
- Any business logic (questions, exams, grading, certificates, reporting) — Phase 2+.
- Exam engine, session management, anti-cheat — Phase 3.
- Multi-tenancy schema partitioning — Phase 1 follow-on (FR-BB13).
- All Phase 2+ features (question bank, exam configuration, employee portal, analytics, AI).

## Test Strategy

| AC | Verification method |
|----|---------------------|
| AC-1 | `docker compose up --build` in CI; assert all four containers reach `running`/`healthy` state |
| AC-2 | `cat backend/go.mod` and assert each required module appears; `go build ./...` exits 0 |
| AC-3 | `npm run dev` smoke test in CI; assert dev server binds port 5173 |
| AC-4 | `tsc --noEmit` exits 0; Playwright/Vitest DOM test asserts a `<button>` element is rendered at `http://localhost:5173` |
| AC-5 | `curl http://localhost/api/v1/health` via nginx; assert HTTP 200 |
| AC-6 | `grep -n 'changeme\|secret\|password' .env.example` returns no production-grade secrets; presence of all required keys asserted in a shell script |
| AC-7 | Each Makefile target invoked in CI dry-run; assert exit code 0 |
| AC-8 | `go build ./...` and `npm run build` run in clean Docker image; assert exit codes 0 and no stderr warnings |
| AC-9 | `curl -s http://localhost:8080/api/v1/health` asserts HTTP 200 and response body equals `{"data":{"status":"ok"},"error":null}` |
| AC-10 | Start API without `JWT_SECRET` set; assert process exits with non-zero code and stderr contains a descriptive message |

## Technical Specification

### Configuration / Infrastructure

#### `docker-compose.yml` service summary

```yaml
services:
  db:
    image: postgres:16-alpine
    environment:
      POSTGRES_DB: ${DB_NAME}
      POSTGRES_USER: ${DB_USER}
      POSTGRES_PASSWORD: ${DB_PASSWORD}
    ports: ["5432:5432"]
    volumes: [pgdata:/var/lib/postgresql/data]
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ${DB_USER}"]
      interval: 5s
      retries: 10

  api:
    build: ./backend
    env_file: .env
    ports: ["8080:8080"]
    depends_on:
      db:
        condition: service_healthy

  frontend:
    build:
      context: ./frontend
      target: build
    volumes:
      - frontend_dist:/app/dist

  nginx:
    image: nginx:alpine
    ports: ["80:80"]
    depends_on: [api, frontend]
    volumes:
      - ./deploy/nginx.conf:/etc/nginx/conf.d/default.conf:ro
      - frontend_dist:/usr/share/nginx/html:ro
```

#### Nginx routing rules (`deploy/nginx.conf`)

```nginx
server {
    listen 80;

    location /api/ {
        proxy_pass         http://api:8080;
        proxy_set_header   Host $host;
        proxy_set_header   X-Real-IP $remote_addr;
        proxy_set_header   X-Forwarded-For $proxy_add_x_forwarded_for;
    }

    location / {
        root  /usr/share/nginx/html;
        try_files $uri $uri/ /index.html;
    }
}
```

#### `.env.example`

```dotenv
# ── Database ─────────────────────────────────────────────
DB_HOST=db
DB_PORT=5432
DB_NAME=bilimbaga
DB_USER=bilimbaga
DB_PASSWORD=changeme
DB_MAX_OPEN_CONNS=25
DB_MAX_IDLE_CONNS=5
DB_CONN_MAX_IDLE_SECONDS=300

# ── API ──────────────────────────────────────────────────
API_PORT=8080
API_BASE_URL=http://localhost:8080

# ── JWT ──────────────────────────────────────────────────
JWT_SECRET=replace-with-at-least-32-random-chars
JWT_ACCESS_TTL_MINUTES=15
JWT_REFRESH_TTL_DAYS=7

# ── Security ─────────────────────────────────────────────
BCRYPT_COST=12
COOKIE_DOMAIN=localhost
COOKIE_SECURE=false
```

#### Go module layout (`backend/`)

```
backend/
  cmd/api/        main.go — wires config, DB, router, server
  internal/
    config/       typed Config struct; loaded once at startup
    db/           sqlx pool factory
    router/       Chi router factory; registers all sub-routers
    health/       handler for GET /api/v1/health
  migrations/     (populated by FR-BB12)
  go.mod
  go.sum
  Dockerfile
```

#### `Makefile` targets

```makefile
.PHONY: dev build migrate test

dev:
	docker compose up --build

build:
	cd backend && go build -o bin/api ./cmd/api
	cd frontend && npm ci && npm run build

migrate:
	docker compose run --rm api ./bin/api migrate

test:
	cd backend && go test ./...
	cd frontend && npm test -- --watchAll=false
```

### API Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/v1/health` | None | Liveness check |

#### Request / Response shapes

```json
// GET /api/v1/health response
{
  "data": { "status": "ok" },
  "error": null
}
```

### Frontend Components
- `App.tsx` — root component; initialises React Router, React Query client, and i18next.
- `main.tsx` — Vite entry point; mounts `<App />` into `#root`.
- A `Button` shadcn/ui component imported from `@/components/ui/button` must render without type errors (smoke test).

## Notes
- Go binary must be compiled inside the Docker image (multi-stage build) so the final image contains only the binary, not the Go toolchain.
- Frontend Docker image uses a two-stage build: `node` stage compiles the bundle; the output is copied to a named volume consumed by `nginx`.
- `COOKIE_SECURE=false` is acceptable only in local development; production deployments must set it to `true`.
- The `JWT_SECRET` must be at least 256 bits (32 characters) — validated at startup; API exits if the requirement is not met.

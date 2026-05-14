# FR-BB11 — Project Scaffold

## Metadata
| Field | Value |
|-------|-------|
| ID | FR-BB11 |
| Phase | 1 — Foundation |
| Priority | 1 |
| Status | Draft |
| Depends On | — |

## Description
Establishes the complete project skeleton for BilimBaga, including all four Docker Compose services (`db`, `api`, `frontend`, `nginx`), the Go backend module wired with Chi, sqlx, and golang-migrate, and the React 18 + TypeScript frontend wired with Vite, Tailwind CSS, and shadcn/ui. This requirement has no business logic; it delivers the development environment and CI-ready build chain that all subsequent requirements build upon.

## Acceptance Criteria
- [ ] AC-1: `docker-compose.yml` defines four services — `db` (PostgreSQL 16), `api` (Go), `frontend` (Node build + static), `nginx` (reverse proxy) — and all four start without error via `make dev`.
- [ ] AC-2: Go module is initialised (`go.mod` / `go.sum`) with `github.com/go-chi/chi/v5`, `github.com/jmoiron/sqlx`, `github.com/golang-migrate/migrate/v4`, and `github.com/golang-jwt/jwt/v5` as direct dependencies.
- [ ] AC-3: Frontend project is initialised with Vite + React 18 + TypeScript; `npm run dev` starts the dev server on port 5173 without errors.
- [ ] AC-4: Tailwind CSS is configured and at least one shadcn/ui primitive (e.g. `Button`) renders correctly in the dev environment.
- [ ] AC-5: Nginx config routes all `/api/*` requests to the Go service on port 8080 and serves the compiled frontend static files for all other paths.
- [ ] AC-6: `.env.example` documents every required environment variable with a descriptive comment; no variable has a production secret as its default.
- [ ] AC-7: `Makefile` exposes at minimum four targets: `dev` (starts Compose), `build` (builds Go binary + frontend bundle), `migrate` (runs pending migrations), `test` (runs backend and frontend tests).
- [ ] AC-8: `go build ./...` and `npm run build` both complete without errors or warnings in CI (clean environment, no pre-installed caches).
- [ ] AC-9: Health-check endpoint `GET /api/v1/health` returns `200 OK` with `{"status":"ok"}` when the API service is running.

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

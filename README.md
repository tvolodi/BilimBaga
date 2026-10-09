# BilimBaga

Corporate exam platform for organizations — multilingual question banks, configurable timed sessions, auto-grading, certificates, and analytics.

---

## Prerequisites

- [Docker](https://www.docker.com/) with Docker Compose
- Ports **80**, **5432**, and **8080** must be free (or override them, see below)

---

## Start the application

```bash
make dev
```

This builds all services (PostgreSQL, API, frontend, Nginx) and starts them. The frontend image is always rebuilt fresh on every `make dev` run — there is no stale-bundle risk. The app is ready when all containers report healthy.

| Service  | URL                    |
|----------|------------------------|
| Frontend | http://localhost       |
| API      | http://localhost:8080  |

### Running on a free port

If 8080 is taken by another process, set `BB_API_PORT` (host API port) before starting:

```bash
BB_API_PORT=18080 make dev
# E2E / scripts: same var, or a full URL override
BB_API_PORT=18080 npm run test:e2e:live     # in frontend/
E2E_API_URL=http://localhost:18080 npm run test:e2e:live
```

`E2E_BASE_URL` overrides the frontend URL Playwright drives (default `http://localhost:5173`). Nothing defaults to a remote host: `scripts/seed-test-env.ts` requires `E2E_API_URL`; it and the e2e setup only accept an allowlist of hosts (localhost, 127.0.0.1, ::1, `*.localhost`, `bilimbaga-qa.ai-dala.com`) on the normalised hostname, and refuse anything else, notably `bilimbaga-test.ai-dala.com` (customer demo, production-class), unless `ALLOW_PROTECTED_HOST=1`. The API sets no CORS headers (same-origin via proxy/nginx), so there is no CORS setting. The Vite dev proxy honours `BB_API_PORT` / `E2E_API_URL` too. `HOST_DB_PORT` / `HOST_HTTP_PORT` work likewise.

---

## Initial admin login

The first migration seeds a default super-admin account:

| Field    | Value                   |
|----------|-------------------------|
| Email    | `admin@bilimbaga.local` |
| Password | `Admin1234!`            |

> **Security:** this default is public. For any deployment reachable by others, set `BOOTSTRAP_ADMIN_PASSWORD` (8+ chars, upper, lower, digit; must differ from the default) in the API environment before first start: at startup the API bcrypt-hashes it and stores it for the admin, but only while the admin still has the default password (a password someone already chose is never overwritten). Alternatively set `BOOTSTRAP_ADMIN_GENERATE=true` to have a random one-time password generated and logged once at startup (change forced at first login). The value is never logged. If neither is set, the admin keeps the default password but is forced to change it at first login (migration 033 plus a startup check) and the API logs a `SECURITY` warning on every start while the default is still in place.
> For e2e / `scripts/seed-test-env.ts` runs against a stack started with `BOOTSTRAP_ADMIN_PASSWORD`, export the same value as `E2E_ADMIN_PASS`.

---

## Other commands

| Command              | Description                        |
|----------------------|------------------------------------|
| `make migrate`       | Run pending database migrations    |
| `make test`          | Run backend and frontend test suites |
| `make build`         | Build production binaries           |
| `make security-check`| Dependency vulnerability audit     |

---

## Migrations and deploy order

The API applies pending migrations at startup, before it serves requests (`backend/cmd/api/main.go`, `dbpkg.RunMigrations`); the migrations are copied into the image (`backend/Dockerfile`), and `deploy/redeploy-*.sh` have no separate migrate step. If a migration fails, the container exits with status 1 and never serves.

Rule: any migration used by fail-closed code must ship in the same image as the code. Migrations should stay backward compatible (e.g. nullable columns) so that rolling back to the previous image is safe. `make migrate` (`docker compose run --rm api migrate`) runs only the pending migrations and exits with status 0 (non-zero on failure) without starting the server; it is idempotent. The API also applies pending migrations automatically at startup, so `make migrate` is optional and mainly useful to migrate ahead of a deploy.

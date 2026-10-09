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

> **Important:** You will be required to change the password on first login.

---

## Other commands

| Command              | Description                        |
|----------------------|------------------------------------|
| `make migrate`       | Run pending database migrations    |
| `make test`          | Run backend and frontend test suites |
| `make build`         | Build production binaries           |
| `make security-check`| Dependency vulnerability audit     |

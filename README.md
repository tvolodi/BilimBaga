# BilimBaga

Corporate exam platform for organizations — multilingual question banks, configurable timed sessions, auto-grading, certificates, and analytics.

---

## Prerequisites

- [Docker](https://www.docker.com/) with Docker Compose
- Ports **80**, **5432**, and **8080** must be free

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

---
id: ISS-146
title: make migrate fails - no ./bin/api in image and no migrate subcommand
status: resolved
severity: low
layer: config
module: tenant
tags: [make migrate, ./bin/api, ENTRYPOINT, RunMigrations, migrate subcommand]
created: 2026-10-09
resolved: 2026-10-09
recurrence_count: 1
related_issues: [ISS-105]
regression_test: backend/cmd/api/run_test.go
---

## Symptom
`make migrate` ran `docker compose run --rm api ./bin/api migrate`. The image has `ENTRYPOINT ["/app/api"]` (no `./bin/api`), and `main.go` ignored arguments (it just started the server).

## Root Cause
Makefile target written for a non-existent path; no `migrate` argument handling existed.

## Fix
- `backend/cmd/api/run.go`: `run(args, deps) int` dispatcher (no args -> serve; `migrate` -> migrator, prints "migrations applied", exit 0 / 1 on error; anything else -> usage, exit 2). `dbMigrator` loads Config, opens DB, runs `dbpkg.RunMigrations`; no server, scheduler, email jobs or BootstrapAdmin.
- `main.go`: old `main` renamed `serve` (unchanged behaviour); DB open extracted to `openDB`.
- Makefile: `docker compose run --rm api migrate` plus comment. docker-compose.yml needs no change (no entrypoint/command override; env_file supplies DB vars; depends_on db healthy).
- README "Migrations and deploy order" updated. CLAUDE.md untouched (project policy).

## Tests
`backend/cmd/api/run_test.go`: no args serves; migrate calls migrator and not server; migrator error -> exit 1 and no success message; bad args -> usage, exit 2. go vet, staticcheck, `go test -p 1 ./...` pass.

## Unverified
Not run against a real DB/stack. UAT: with db up, `docker compose run --rm api migrate` -> exit 0 and "migrations applied"; run again -> exit 0 (idempotent); `docker compose run --rm api bogus` -> exit 2 with usage.

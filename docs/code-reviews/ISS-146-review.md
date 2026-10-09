# Code Review: ISS-146 (make migrate subcommand)

Verdict: APPROVE

Scope: backend/cmd/api/run.go, run_test.go, main.go, Makefile, README.md, docs/issue-reports/ISS-146-make-migrate-subcommand.md (staged diff, read-only; tests not run per memory constraint, issue report states vet/staticcheck/tests pass).

## Checks
- Normal startup unchanged: `main()` renamed `serve()`; only change inside is the DB open block replaced by `openDB(cfg)` with identical field mapping. Error message becomes "startup error: open database: ..." (same text via %w wrap). Migrations, jobs, BootstrapAdmin, signal handling remain in `serve`. No-args path calls `serve()` and returns 0 (serve exits via os.Exit on failure as before).
- `migrate` path: loads Config, opens DB, `RunMigrations(db, "migrations")`, closes DB. No server, scheduler, email jobs, tenant cache or BootstrapAdmin. Relative "migrations" resolves because Dockerfile WORKDIR /app and migrations copied to /app/migrations; ENTRYPOINT /app/api so `api migrate` is correct. Compose has no entrypoint/command override.
- Exit codes: 0 success, 1 error (stderr), 2 usage for unknown args or extra args. Good.
- Tests: four tests cover no-args serves without migrating, migrate does not serve, error gives exit 1 with no success message, bad args give usage/exit 2. Meaningful via injected deps.
- Docs: README and Makefile accurate; CLAUDE.md untouched.

## Findings (non-blocking)
1. Minor: `dbMigrator` and `openDB` are not unit tested (need a DB); covered only by the UAT noted under "Unverified". Acceptable.
2. Minor: migrate does not initialise the zerolog logger, so any migration logging goes to default logger; output is just plain stdout/stderr. Fine.
3. Info: `serve` still prints "startup error: open database: ..." through the extra wrap; no behavior change.

No blocking issues.

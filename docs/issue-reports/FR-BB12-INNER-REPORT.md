# FR-BB12 Inner Report — Database Bootstrap

**Date**: 2026-05-14  
**Status**: PASS  
**Pipeline**: A — Feature Development  

---

## Summary

FR-BB12 (Database Bootstrap) has been fully implemented. The Go backend now establishes a configured PostgreSQL connection pool on startup, automatically runs pending golang-migrate migrations, and exits with a descriptive error if the database is dirty or unreachable.

---

## Files Created

| File | Action |
|------|--------|
| `backend/internal/db/migrate.go` | Created — `RunMigrations` using golang-migrate v4 |
| `backend/migrations/001_init.up.sql` | Created — placeholder `SELECT 1;` |
| `backend/migrations/001_init.down.sql` | Created — placeholder `SELECT 1;` |
| `backend/internal/db/db_test.go` | Created — unit tests for `db.Config` struct and `db.New` error path |

## Files Modified

| File | Change |
|------|--------|
| `backend/internal/db/db.go` | Replaced `Connect(*config.Config)` with `New(Config)` + local `Config` type; added `SSLMode` field; DSN now dynamic |
| `backend/internal/config/config.go` | Added `DBSSLMode string` field; loaded from `DB_SSLMODE` env var with default `"disable"` |
| `backend/cmd/api/main.go` | Wired `dbpkg.New` + `dbpkg.RunMigrations` into startup; both fatal-exit on error; aliased import as `dbpkg` |
| `backend/internal/config/config_test.go` | Added `TestLoad_DBSSLModeDefault` and `TestLoad_DBSSLModeFromEnv` tests |
| `backend/go.mod` | Added `github.com/golang-migrate/migrate/v4 v4.19.1` as direct dependency |
| `docs/requirements/README.md` | Status updated: Draft → Implemented |
| `docs/requirements/FR-BB12.Database-bootstrap.md` | Status updated: draft → implemented |

---

## Acceptance Criteria Coverage

| AC | Description | Status |
|----|-------------|--------|
| AC-1 | golang-migrate applied on startup before HTTP listen | ✅ Implemented in `main.go` |
| AC-2 | Migration naming convention `{NNN}_{desc}.up/down.sql` | ✅ `001_init.up.sql`, `001_init.down.sql` |
| AC-3 | `schema_migrations` table created by migrate library | ✅ golang-migrate creates this automatically |
| AC-4 | Migration failure → log error + non-zero exit | ✅ `os.Exit(1)` on `RunMigrations` error |
| AC-5 | Pool config from env vars with defaults 25/5/300 | ✅ Already in config, wired in `main.go` |
| AC-6 | `db.Ping()` called (via `sqlx.Connect`) | ✅ `sqlx.Connect` calls `Ping` internally |
| AC-7 | `ErrNoChange` treated as success | ✅ Handled in `RunMigrations` |
| AC-8 | Down migrations exist for every up migration | ✅ `001_init.down.sql` present |
| AC-9 | Dirty flag → descriptive error with version + non-zero exit | ✅ `ErrDirty` handled in `RunMigrations` |
| AC-10 | `DB_SSLMODE` env var applied to DSN | ✅ `cfg.SSLMode` in DSN format string |
| AC-11 | `.env.example` has `DB_SSLMODE=disable` | ✅ Already present (confirmed) |

---

## Blockers Encountered

| Blocker | Resolution |
|---------|------------|
| Existing `db.go` used `Connect(*config.Config)` (circular-ish dep, hardcoded sslmode=disable) | Replaced with `New(Config)` using self-contained `db.Config` type |
| `golang-migrate/migrate/v4` absent from `go.mod` | Added via `go get github.com/golang-migrate/migrate/v4@latest` |
| `go mod tidy` network error during download of test-only transitive deps | Non-blocking — build and all tests pass; `go.mod` correctly lists migrate as direct dep |

---

## Test Results

```
?       github.com/bilimbaga/bilimbaga/cmd/api          [no test files]
ok      github.com/bilimbaga/bilimbaga/internal/config  0.584s
ok      github.com/bilimbaga/bilimbaga/internal/db      0.897s
ok      github.com/bilimbaga/bilimbaga/internal/health  (cached)
?       github.com/bilimbaga/bilimbaga/internal/router  [no test files]
```

All tests pass. No compile errors.

---

## Self-Review Findings

- No secrets hardcoded. No `os.Getenv` in handlers.
- All errors wrapped with `fmt.Errorf("context: %w", err)`.
- No dead code or debug prints.
- DSN uses `%s` format args from `Config` struct fields — no injection risk for internal config values (these come from environment variables, not user input).

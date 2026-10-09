---
id: ISS-026
title: internal/db and internal/email under-tested (13.8% / 32.7%)
status: resolved
severity: low
layer: backend
module: audit
tags: [coverage, internal/db, internal/email, RunMigrations, LogSlowQuery, smtpSend, fakedb]
created: 2026-10-09
resolved: 2026-10-09
recurrence_count: 1
related_issues: [ISS-024]
regression_test: backend/internal/email/service_more_test.go
---

> Note: the number 026 here is the GitHub issue number (#26) as requested by the orchestrator; it collides with the older `ISS-026-change-password-autofill-stale-credential.md`. Distinguish by slug.

## Symptom
GitHub issue #26: `backend/internal/db` (13.8%) and `backend/internal/email` (32.7%) coverage. Re-measured at start of this task: identical (db 13.8%, email 32.7%); `internal/ctxkeys` and `internal/router` had 0% and no tests.

## Root Cause
Tests were never written for the repository, scheduler, handler happy path, SMTP delivery and trigger methods, nor for `LogSlowQuery` / `RunMigrations`, because they need a database or an SMTP server.

## Fix Applied
Tests only, no production code changes (no seam was needed).

- `internal/db`: `instrumented_test.go` (fast / threshold / slow query logging, `$n` sanitising, request id fallback) and `migrate_test.go` (in-process fake `database/sql` driver answering golang-migrate's postgres driver queries; temp migration dirs; cases: apply pending, ErrNoChange, dirty version, failing statement wrapping, driver creation failure, missing source dir).
- `internal/email`: `fakedb_test.go` (copy of the ai/reports fake driver pattern), `repository_test.go` (all 7 repo methods incl. error wrapping, ErrNoRows locale fallback), `smtp_fake_test.go` (loopback-only in-process SMTP server on 127.0.0.1:0, no external network), `service_more_test.go` (smtpSend plain/TLS-mode branches and each failure stage, locale fallback en/ru/kk/unknown/tenant error, render error wrapping, Trigger* goroutines synchronised via channel signalling, handler 401/500/503/200), `scheduler_test.go` (calc durations, invalid TZ fallback, runDeadlineReminders with stub repo and fake SMTP, no sleeping).
- `internal/ctxkeys`: accessors present/absent/wrong-type.
- `internal/router`: full route tree built with nil handlers; `/ping`, 401 on protected routes, 404.

Not covered: `db.New` success path (needs a live Postgres), the time.AfterFunc callback in `scheduleNext` (no clock seam; adding one would be a production change), real STARTTLS upgrade.

## Files Changed
| File | Change |
|------|--------|
| backend/internal/db/instrumented_test.go | new |
| backend/internal/db/migrate_test.go | new |
| backend/internal/email/fakedb_test.go | new |
| backend/internal/email/repository_test.go | new |
| backend/internal/email/smtp_fake_test.go | new |
| backend/internal/email/service_more_test.go | new |
| backend/internal/email/scheduler_test.go | new |
| backend/internal/ctxkeys/ctxkeys_test.go | new |
| backend/internal/router/router_test.go | new |

## Regression Test
The files above. Coverage before -> after: db 13.8% -> 79.3%; email 32.7% -> 94.0%; ctxkeys 0% -> 100%; router 0% -> 100%.

## Resolution Results
- Tests: all packages pass (`go test -p 2 ./...`), `go vet ./...` clean, touched packages pass with `-race -count=2`
- Migration applied: no
- Build clean: yes

## Recurrence Log
| Date | Trigger | Action Taken |
|------|---------|-------------- |

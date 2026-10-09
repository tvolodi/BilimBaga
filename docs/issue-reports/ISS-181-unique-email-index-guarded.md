---
id: ISS-181
title: UNIQUE INDEX on lower(email) without ever breaking startup (follow-up of ISS-164)
status: resolved
severity: medium
layer: database
module: users
tags: [lower(email), idx_users_email_lower_unique, migration 035, DO block, duplicate_emails_detected]
created: 2026-10-09
resolved: 2026-10-09
recurrence_count: 1
related_issues: [ISS-164]
regression_test: backend/internal/db/unique_email_index_test.go
---

## Symptom
users.email has a case-sensitive UNIQUE constraint, so `John@X.com` and `john@x.com` can coexist in
legacy data. A plain `CREATE UNIQUE INDEX ... (lower(email))` migration would fail on such data;
the API self-migrates at startup and exits on failure, so a deployed instance would go DOWN.

## Root Cause
No DB-level case-insensitive uniqueness (ISS-164 only made app lookups/probe case-insensitive), and
adding it naively is unsafe because migrations run (and abort startup) automatically.

## Fix Applied
- Migration 035 (`035_users_email_lower_unique.{up,down}.sql`): DO block counts case-insensitive duplicate
  groups; none -> `CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email_lower_unique ON users (lower(email))`;
  otherwise `RAISE NOTICE` with the count and skip. Never fails, never deletes/merges/modifies users.
  Down: `DROP INDEX IF EXISTS`.
- `users.CheckDuplicateEmails` (internal/users/duplicate_emails.go), called after BootstrapAdmin in
  cmd/api/main.go: if the index is absent and duplicates exist, logs ONE structured WARN
  (duplicate_groups, listed_groups, more_groups_not_listed, groups; list capped at 50 groups) and writes
  ONE audit_log row (`users.duplicate_emails_detected`, tenant 'public', no actor, metadata = group count,
  index name, first 3 groups). Any error or panic is logged and startup continues.
- App-level prevention verified (no change needed): users Create and CSV import both go through
  `repository.Create`, which probes `lower(email) = lower($1)` first (ISS-164) and maps 23505 to
  ErrDuplicateEmail; there is no email-update path.

## Files Changed
| File | Change |
|------|--------|
| backend/migrations/035_users_email_lower_unique.up.sql / .down.sql | guarded index |
| backend/internal/users/duplicate_emails.go | startup check, store interface, pg store |
| backend/internal/users/duplicate_emails_test.go | unit tests with fake store |
| backend/internal/db/unique_email_index_test.go | real-Postgres test (TEST_DATABASE_URL) |
| backend/internal/schemaguard/schema_guard_test.go | migration 035 guard test |
| backend/cmd/api/main.go | wiring |

## Regression Test
Unit: TestCheckDuplicateEmails_* (silent when index present / no duplicates; one WARN + one audit; cap; errors
and panics never abort). Real DB (skipped without TEST_DATABASE_URL):
`cd backend && TEST_DATABASE_URL='postgres://USER:PASS@HOST:5432/DB?sslmode=disable' go test ./internal/db -run TestUniqueEmailIndex -v`
(uses a throwaway schema bb_iss181_*, dropped in t.Cleanup).

## Resolution Results
- Tests: go test -p 1 ./... all packages pass; real-Postgres tests SKIP locally (no Postgres available)
- Migration applied: no (no database in this environment); needs-live-db
- Build clean: yes (go vet, staticcheck 0 findings)

## Notes
- If the index was skipped, it is not retried automatically after an admin resolves the twins; the WARN
  text includes the exact CREATE UNIQUE INDEX statement. The WARN/audit repeat on each restart while twins exist.

- Code review (docs/code-reviews/ISS-181-review.md) PASS; applied: unique_violation handler around CREATE
  (check/create race in rolling deploys) and current_schema() filter on the index lookup. Accepted trade-off:
  full twin emails are logged (capped at 50 groups) as the issue's required admin report; audit holds counts and 3 sample groups.

## Recurrence Log
| Date | Trigger | Action Taken |
|------|---------|-------------- |

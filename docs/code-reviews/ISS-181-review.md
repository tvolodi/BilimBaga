# Code Review: ISS-181 (guarded unique index on lower(email), migration 035)

Reviewer: Code Reviewer subagent. Scope: uncommitted changes in worktree dev1. Read-only review; `go vet ./internal/users ./internal/db ./cmd/api` is clean.

Result: PASS (0 Critical, 0 High, 3 Medium, 3 Low)

## Findings

- [Medium] backend/migrations/035_users_email_lower_unique.up.sql:19-20 - "never fails" is not strictly guaranteed. The duplicate count and `CREATE UNIQUE INDEX` are not atomic. During a rolling deploy an old instance can still be writing. If a case-variant duplicate commits between the count and the index build, the CREATE raises 23505, the migration fails, the migration is marked dirty, and the API exits at startup. That is the failure mode the migration exists to prevent. CREATE UNIQUE INDEX also takes a SHARE lock on `users`, so it blocks writes while it builds. This is harmless for a small table, and golang-migrate's advisory lock serialises replicas. Suggestion: wrap the CREATE in a nested block with `EXCEPTION WHEN unique_violation THEN RAISE NOTICE '...skipped...'`. Optionally add `SET LOCAL lock_timeout = '5s'` and catch `lock_not_available` so the migration cannot hang behind a long transaction. A DO block's EXCEPTION clause works here because plain (non-CONCURRENTLY) CREATE INDEX is transactional.

- [Medium] backend/internal/users/duplicate_emails.go:105-106 - the `pg_indexes` lookup filters only on `tablename = 'users'` and the index name, with no schema filter. It can match a `users` table with the same index name in another schema, such as a leftover test schema or a tenant schema. That would wrongly suppress the report. Suggestion: add `AND schemaname = current_schema()`. Use `to_regclass`/`pg_class` if you prefer. The migration itself resolves `users` through search_path, so `current_schema()` matches it.

- [Medium] backend/internal/users/duplicate_emails.go:75-82, 88-92 - PII handling. Up to 50 groups of full email addresses go to the WARN log, and 3 sample groups go into `audit_log.metadata`. The log is not access-controlled like the DB, and the WARN repeats on every restart, so the emails spread to log aggregators and retention. Suggestion: log only counts, or mask the local part (`j***@x.com`). Alternatively give the admin a query to list the twins, since the WARN already prints the remediation. The audit metadata is lower risk (admin-only) but should stay at counts, or at masked samples. If full emails in the log are an accepted trade-off because the admin needs them to act, state that explicitly in the issue report.

- [Low] duplicate_emails.go:59-69 - the audit row and WARN are emitted on every restart while twins exist, and on every replica. The issue report documents this. Consider de-duplicating, for example skip the audit insert if a row with the same action exists within the last 24h.

- [Low] up.sql:16 - `GROUP BY lower(email)` treats NULL emails as one group. This is moot while `users.email` is NOT NULL. The test schema assumes NOT NULL, but the real column constraint was not re-verified here. Add `WHERE email IS NOT NULL` for defensiveness. Note that `array_agg` plus `pq.Array(&[]string)` would fail on NULL elements.

- [Low] schema_guard_test.go (TestMigration035TargetsExistingColumn) - the substring checks are brittle. Forbidding `"update "` and `"drop "` could trip on a harmless edit. It does serve as a cheap guard. The real-Postgres test is the meaningful coverage, and it is skipped without TEST_DATABASE_URL.

## Verified correct

- The DO block syntax and Postgres semantics are fine. `count(*) INTO` with the subquery works. `IF/ELSE` with `RAISE NOTICE '%'` (one placeholder, one argument) is correct. The migration contains no DML on users, so it cannot modify or delete users. The down migration uses `DROP INDEX IF EXISTS`, which is idempotent.
- `array_agg(email ORDER BY email)` with `ORDER BY lower(email) LIMIT $1` is valid in a grouped query. Scanning with `rows.Scan(pq.Array(&g))` into `[]string` is correct. `rows.Close` and `rows.Err` are handled.
- The audit insert matches the migration 007 schema: `tenant_id` NOT NULL is given as 'public', and `actor_id` and `entity_id` are NULL-able. `ip` is '' and metadata goes in as `$2::jsonb`.
- The startup wiring runs after migrations and bootstrap, uses a 15s timeout, and has recover plus error-log-and-continue, so startup is never aborted. The WARN is exactly one record. Unit tests cover the silent paths, the caps, errors and a panic.
- The real-Postgres test uses an isolated throwaway schema with a pinned single connection (so search_path holds). It uses a notice handler and cleans up with DROP SCHEMA CASCADE. It never touches public.users.
- Errors are wrapped with context. SQL is parameterized. There are no secrets and no `os.Getenv` use. No existing migration was edited, and migration 035 follows the numbering and the .up/.down pair convention.
- There are no HTTP or RBAC changes, so the endpoint and response-envelope checks do not apply.

## AC Coverage (from the task brief)

- Guarded unique index, never failing on duplicates: covered, with the residual race noted above (Medium).
- Never modifies users: covered.
- Startup report of one WARN plus one audit row, capped, never aborts: covered.
- Real-Postgres test gated on TEST_DATABASE_URL: covered (not executed here, no DB).
- Schemaguard addition and issue report: covered.

Summary: Sound and safe in the normal case. No Critical or High issues. Hardening the CREATE with an EXCEPTION handler, adding a schema filter to the index lookup, and reconsidering PII in logs are recommended but not blocking.

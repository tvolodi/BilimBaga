---
id: ISS-004
title: auto-submit audit log query references non-existent exams.tenant_id column
status: resolved
severity: high
layer: backend
module: sessions
tags: [audit_log, tenant_id, exams, auto_submit, session.auto_submit, column does not exist]
created: 2026-05-18
resolved: 2026-05-18
recurrence_count: 1
related_issues: []
regression_test: backend/internal/sessions/autojob_test.go::TestAuditQueryUsesLiteralTenantID
---

## Symptom
```
db-1 | ERROR:  column e.tenant_id does not exist at character 99
db-1 | STATEMENT:
db-1 |   INSERT INTO audit_log (tenant_id, actor_id, action, entity_type, entity_id, ip, metadata)
db-1 |   SELECT e.tenant_id, NULL, 'session.auto_submit', 'exam_session', $1, '', $2::jsonb
db-1 |   FROM exams e WHERE e.id = $3
```
Fires every 60 seconds (auto-submit job polling tick) whenever at least one in_progress session is due for auto-submission. Logged as an error but does not abort the session state transition.

## Root Cause
`backend/internal/sessions/autojob.go:191-194` — The audit INSERT uses a subquery `SELECT e.tenant_id FROM exams e WHERE e.id = $3`. The `exams` table (migration `012_exams.up.sql`) has no `tenant_id` column. The `tenant_id` on `audit_log` is a `TEXT NOT NULL` field representing the tenant slug (always `"public"` in Phase 1), set by the HTTP `TenantContext()` middleware. The auto-submit job is a background goroutine with no HTTP context, so it must supply the tenant ID directly — either from context or as a literal. Since the job always runs in the single-tenant Phase 1 environment where `tenant_id = 'public'`, and the `TenantContext` middleware hardcodes this same value, the correct fix is to supply `'public'` directly as a SQL literal (or pass it via `ctxkeys.CtxTenantID` in context). Using a SQL literal is safer for a background job that has no HTTP request context.

## Fix Applied
Changed the audit INSERT in `processOneExpiredSession` to use the literal `'public'` instead of `SELECT e.tenant_id FROM exams e`. The SELECT subquery form is removed entirely; the INSERT now uses a plain `VALUES` clause, matching the pattern used by `SubmitSession` in `repository.go:871-872`.

## Files Changed
| File | Change |
|------|--------|
| `backend/internal/sessions/autojob.go` | Replace subquery-based INSERT with VALUES-based INSERT using `'public'` as tenant_id |

## Regression Test
`backend/internal/sessions/autojob_test.go` — Added `TestAuditQueryUsesLiteralTenantID` which asserts the SQL constant in `processOneExpiredSession` does NOT reference `e.tenant_id` and DOES contain `'public'`.

## Resolution Results
- Tests: 25 packages passed, 0 failed
- Migration applied: no (schema unchanged)
- Build clean: yes

## Recurrence Log
| Date | Trigger | Action Taken |
|------|---------|--------------|
| 2026-05-18 | Docker log error on make dev startup | ISS-004 created, fix applied |

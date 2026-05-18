# ISS-004 Inner Report — Auto-Submit Audit Log References Non-Existent exams.tenant_id Column

**Run ID:** iss-004
**Pipeline:** B (Bug Fix)
**Date:** 2026-05-18
**Status:** CLOSED

---

## Issue Summary

The background auto-submit job (`autojob.go`) emitted a PostgreSQL error every 60 seconds
whenever an in-progress session was due for auto-submission. The audit INSERT used a
subquery `SELECT e.tenant_id FROM exams e WHERE e.id = $3`, but the `exams` table has no
`tenant_id` column, causing `ERROR: column e.tenant_id does not exist`.

---

## Root Cause

`backend/internal/sessions/autojob.go` lines 191–194 — the audit log INSERT was modelled
after HTTP-handler paths that build `tenant_id` from request context. The background job
has no HTTP context; and the `exams` table (migration `012_exams.up.sql`) never had a
`tenant_id` column. The `audit_log.tenant_id` column is a `TEXT NOT NULL` slug always
equal to `"public"` in Phase 1, set by `TenantContext()` middleware on inbound HTTP
requests. A background goroutine must supply this literal directly.

---

## Fix Applied

**File:** `backend/internal/sessions/autojob.go`

- Replaced the subquery-based `INSERT INTO audit_log ... SELECT e.tenant_id FROM exams e`
  with a `VALUES`-based INSERT that supplies `'public'` as the literal `tenant_id`.
- Pattern now matches `SubmitSession` in `repository.go:871-872`.

---

## Tests

**File:** `backend/internal/sessions/autojob_test.go`

New test case added: `TestAuditQueryUsesLiteralTenantID`
- Asserts the SQL constant in `processOneExpiredSession` does NOT contain `e.tenant_id`.
- Asserts the SQL constant DOES contain the literal `'public'`.

All 25 backend packages pass. No frontend changes. No migration required.

---

## Commit

**Hash:** (see git log for iss-004 commit)
**Message:** `fix(sessions): use literal tenant_id in auto-submit audit INSERT`

**Files changed (3):**
- `backend/internal/sessions/autojob.go`
- `backend/internal/sessions/autojob_test.go`
- `docs/issue-reports/ISS-004-autojob-audit-exams-no-tenant.md`

---

## Verification

| Check | Result |
|-------|--------|
| Backend tests | 25 packages passed, 0 failed |
| Frontend tests | Not applicable (no frontend changes) |
| Migrations | None required |
| Build clean | Yes |
| Handoffs staged | docs/handoffs/iss-004/ (excluded from commit per policy) |

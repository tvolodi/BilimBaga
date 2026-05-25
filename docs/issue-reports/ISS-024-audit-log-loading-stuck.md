---
id: ISS-024
title: audit log page remains in loading state due to unbounded query
status: open
severity: high
status: resolved
module: audit
layer: frontend
created: 2026-05-25
tags: [audit, loading, react-query, query-key, defaultFrom, admin/audit]
recurrence_count: 1
resolved: 2026-05-25
regression_test: null
---
regression_test: frontend/src/pages/admin/AuditLogPage.test.tsx
## Symptom
On /admin/audit, the page can remain on "Loading..." for a long time. The UI sends a default `from` timestamp for last 7 days, but listing still behaves like a full-history scan.
On /admin/audit, the page can remain on "Loading..." and not settle, even when the backend is healthy.
## Root Cause
To be filled after reproduction confirmation.
`frontend/src/pages/admin/AuditLogPage.tsx` rebuilt the default `from` timestamp on every render when URL params did not include `from`.

Because `useAuditLog` uses `['audit-log', filters, page]` as a query key, the changing `filters.from` value produced a new key every render. This continuously re-created the query in first-load state, so the page appeared stuck on loading.
## Fix Applied
To be filled.
Made fallback `from` value stable for the mounted page instance:

- Added `fallbackFrom = useMemo(() => defaultFrom(), [])`.
- Replaced `rawFilters.from ?? defaultFrom()` with `rawFilters.from ?? fallbackFrom`.

This keeps the query key stable across rerenders unless filters actually change.
## Files Changed
| File | Change |
|------|--------|
|------|--------|
| `frontend/src/pages/admin/AuditLogPage.tsx` | Stabilized default `from` filter value with memoization to prevent query-key churn |
| `frontend/src/pages/admin/AuditLogPage.test.tsx` | Added ISS-024 regression test that verifies default `from` stays stable across rerenders |

## Regression Test
`frontend/src/pages/admin/AuditLogPage.test.tsx` — `keeps default from filter stable across rerenders when URL has no from param (ISS-024)`.

## Resolution Results
- Tests: targeted frontend regression test passed (`npx vitest run src/pages/admin/AuditLogPage.test.tsx`)
- Migration applied: no
- Build clean: yes (`cd backend && go build ./...`)

## Recurrence Log
| Date | Trigger | Action Taken |
|------|---------|--------------|
| 2026-05-25 | User report: audit log loads too long | ISS-024 created, fixed query-key churn on audit page |

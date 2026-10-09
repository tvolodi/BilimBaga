---
id: ISS-055
title: ExportCSVButton, exportAuditLog, exportEmployeeRecord use own fetch without 401 refresh or error display
status: resolved
severity: medium
layer: frontend
module: reports
tags: [ExportCSVButton, exportAuditLog, exportEmployeeRecord, downloadFile, 401, refresh]
created: 2026-10-09
resolved: 2026-10-09
recurrence_count: 1
related_issues: [ISS-037]
regression_test: frontend/src/components/analytics/__tests__/ExportCSVButton.test.tsx
---

## Symptom
GitHub #66 (from docs/requirements/conformance/PR48-PR52-conformance-20261009.md). The three CSV exports
used a private `fetch` + blob + anchor flow: an expired access token produced a failed download with no
refresh and no message (unhandled rejection / silently ignored).

## Root Cause
These exports pre-dated the shared helper `downloadFile` (frontend/src/api/download.ts, ISS-037) and were not migrated.
Callers had `try/finally` with no `catch`, so errors were swallowed or left unhandled.

## Fix Applied
- `exportAuditLog(qc, filters)` and `exportEmployeeRecord(qc, userId)` now delegate to `downloadFile` (Bearer, one 401 refresh+retry, revoke).
- `ExportCSVButton` calls `downloadFile` directly.
- All three call sites catch errors and render an inline `role="alert"` with `t(downloadErrorKey(err))`
  (existing `download.failed` / `download.session_expired` keys in en/ru/kk; check:i18n green).
- Filenames now come from the fallback names (Content-Disposition parsing was dropped, consistent with the helper).

## Files Changed
| File | Change |
|------|--------|
| frontend/src/api/audit.ts | exportAuditLog uses downloadFile; signature (qc, filters) |
| frontend/src/api/employees.ts | exportEmployeeRecord uses downloadFile; signature (qc, userId) |
| frontend/src/components/analytics/ExportCSVButton.tsx | downloadFile + error alert |
| frontend/src/pages/admin/AuditLogPage.tsx | catch + error alert |
| frontend/src/pages/admin/EmployeeRecordPage.tsx | catch + error alert |
| frontend/src/components/analytics/__tests__/ExportCSVButton.test.tsx | rewritten: success, 401->refresh, refresh fail, server error |
| frontend/src/api/download.test.ts | endpoint/Bearer cases for the two exports |
| frontend/src/pages/admin/AuditLogPage.test.tsx | export error display tests |

## Regression Test
ExportCSVButton.test.tsx (success, 401 refresh+retry, session-expired alert, generic failure alert), download.test.ts, AuditLogPage.test.tsx.

## Resolution Results
- Tests: 342 passed, 0 failed (59 files, vitest --maxWorkers=2)
- tsc, eslint, check:i18n clean
- Migration applied: no
- Build clean: yes

## Recurrence Log
| Date | Trigger | Action Taken |
|------|---------|-------------- |

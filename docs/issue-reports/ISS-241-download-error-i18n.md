---
id: ISS-241
title: Certificate download server error codes show only generic message
status: resolved
severity: low
layer: frontend
module: certificates
tags: [downloadErrorKey, SESSION_NOT_PASSED, EXAM_NOT_CERTIFIABLE, DownloadError.code, SessionHistoryTable]
created: 2026-10-09
resolved: 2026-10-09
recurrence_count: 1
related_issues: [ISS-249]
regression_test: frontend/src/components/employees/EmployeeComponents.test.tsx
---

## Symptom
FR-BB58 AC-4 requires code-specific i18n feedback when the certificate endpoint returns SESSION_NOT_PASSED or EXAM_NOT_CERTIFIABLE. The UI showed only `download.failed` (or `download.session_expired`).

## Root Cause
`downloadErrorKey` in `frontend/src/api/download.ts` only distinguished ERR_UNAUTHORIZED; `DownloadError.code` was captured but otherwise unused.

## Fix Applied
`downloadErrorKey` now uses a code->key map (ERR_UNAUTHORIZED, SESSION_NOT_PASSED, EXAM_NOT_CERTIFIABLE), falling back to `download.failed`. Added `download.session_not_passed` and `download.exam_not_certifiable` to kk/ru/en. The existing alert in SessionHistoryTable (role=alert) renders the key.

## Files Changed
| File | Change |
|------|--------|
| frontend/src/api/download.ts | code->i18n key map |
| frontend/src/locales/{en,ru,kk}.json | two new keys |
| frontend/src/api/download.test.ts | mapping tests |
| frontend/src/components/employees/EmployeeComponents.test.tsx | SessionHistoryTable tests per code |
| docs/requirements/FR-BB58*.md, README.md | AC-4 ticked, FR-BB58 Implemented |

## Regression Test
EmployeeComponents.test.tsx (SessionHistoryTable block) and download.test.ts.

## Resolution Results
- Tests: 634 passed, 0 failed (vitest, 88 files)
- Migration applied: no
- Build clean: yes (tsc, eslint, check:i18n)

## Recurrence Log
| Date | Trigger | Action Taken |
|------|---------|--------------|

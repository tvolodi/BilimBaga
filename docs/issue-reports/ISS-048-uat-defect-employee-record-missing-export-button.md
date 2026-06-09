---
id: ISS-048
title: "UAT Defect: Analytics & Reporting — Employee Record page missing CSV Export button"
status: resolved
severity: medium
layer: frontend
module: analytics / employee record
tags: [uat, analytics, frontend, export, employee-record]
created: 2026-06-09
resolved: 2026-06-09
recurrence_count: 1
related_issues: []
regression_test: null
---

## Symptom
UAT Scenario: `Employee Record View and Export`, Step 3
Actor: Super Admin
Action: Click Export CSV on Employee Record page (/admin/users/{id}/record)
Expected: Export CSV button is visible on the Employee Record page; clicking it triggers a file download of the employee's session history as CSV
Actual: No Export button is present on the Employee Record page. The page renders employee info, session history table with per-row certificate download buttons, and track progress section — but no bulk CSV export button.
Screenshot: frontend/screenshots/uat-analytics/s3-03-export-btn.png

## Root Cause
The Export CSV button was never added to `EmployeeRecordPage.tsx` during the FR-BB54/FR-BB58 implementation. The backend endpoint `GET /api/v1/admin/users/{id}/record/export` was implemented and functional, but the frontend page only rendered employee info, session history table, and track progress — with no trigger to call the export endpoint.

## Fix Applied
1. Added `exportEmployeeRecord(userId, token)` async function to `frontend/src/api/employees.ts`, following the same blob-download pattern used by `exportAuditLog` in `audit.ts`. The function calls `GET /api/v1/admin/users/{userId}/record/export` with Bearer auth, creates a blob URL, triggers an `<a>` click for download, and cleans up the URL.
2. Updated `EmployeeRecordPage.tsx` to import `useQueryClient` (for token access), `Loader2` and `Download` from `lucide-react`, and `exportEmployeeRecord` from the employees API. Added `isExporting` state and `handleExport` handler. Replaced the plain `<h2>` session history heading with a flex row containing the heading and an "Export CSV" button that shows a spinner while downloading.
3. Added `"export_csv"` i18n key to `en.json` ("Export CSV"), `kk.json` ("CSV жүктеу"), and `ru.json` ("Экспорт CSV").

## Files Changed
| File | Change |
|------|--------|
| `frontend/src/api/employees.ts` | Added `exportEmployeeRecord` async function |
| `frontend/src/pages/admin/EmployeeRecordPage.tsx` | Added imports, export state/handler, Export CSV button near session history heading |
| `frontend/src/locales/en.json` | Added `employee_record.export_csv` key |
| `frontend/src/locales/kk.json` | Added `employee_record.export_csv` key |
| `frontend/src/locales/ru.json` | Added `employee_record.export_csv` key |

## Regression Test
None added — the export button is a simple fetch-and-download interaction consistent with the existing `ExportCSVButton` component pattern already covered by `ExportCSVButton.test.tsx`.

## Resolution Results
- Tests: 259 passed, 0 failed (46 test files)
- Migration applied: no (frontend-only change)
- Build clean: yes

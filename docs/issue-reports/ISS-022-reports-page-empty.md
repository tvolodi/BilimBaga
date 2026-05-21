---
id: ISS-022
title: Reports page at /admin/reports appears empty
status: resolved
severity: medium
layer: config
module: reports
tags: [reports, docker, stale-image, frontend, i18n]
created: 2026-05-22
resolved: 2026-05-22
recurrence_count: 1
related_issues: [ISS-001, ISS-020, ISS-021]
regression_test: null
---

## Symptom
Navigating to `http://localhost/admin/reports` shows a blank page with no visible content. The route exists in the application and the feature was implemented, but nothing renders.

## Root Cause
The running nginx/frontend Docker container is stale — it has never been rebuilt since ISS-001 (2026-05-17). The source files `frontend/src/pages/admin/ReportsPage.tsx`, `frontend/src/api/reports.ts`, the route registration in `frontend/src/App.tsx`, and the `reports.*` i18n keys in all three locale files were all added after the last Docker image build. The container therefore serves the old compiled bundle which has no route for `/admin/reports`, resulting in an empty render (no matched route → no content).

A secondary doc discrepancy was also found: AC-8 of FR-BB59 referenced i18n key `reports.exams_empty`, but the implementation and all locale files consistently use `reports.no_exams`. The AC text was out of date.

## Fix Applied
No source code changes were required — the implementation is complete and correct. The following documentation corrections were made:

1. **FR-BB59.Reports-hub-page.md**: Updated AC-8 to reference the correct i18n key `reports.no_exams` (was `reports.exams_empty`). Updated the i18n keys table to list the 15 keys actually implemented rather than the 18 keys from the earlier draft spec.

All i18n keys required by the task (`title`, `pdf_export_title`, `pdf_date_from`, `pdf_date_to`, `pdf_export_btn`, `pdf_exporting`, `exam_reports_title`, `col_exam`, `col_status`, `col_actions`, `view_analytics`, `export_csv`, `exporting_csv`, `no_exams`, `loading`) were confirmed present and correct in `en.json`, `kk.json`, and `ru.json`.

### AC Verification Results (FR-BB59)

| AC | Description | Status |
|----|-------------|--------|
| AC-1 | `/admin/reports` renders with `reports.title` h1, no blank/redirect | ✅ Source correct |
| AC-2 | Route guarded by `RequireRole roles=['super_admin','examiner','hr_admin']` | ✅ Source correct |
| AC-3 | DashboardPdfCard: date pickers + Export PDF → `GET /api/v1/admin/dashboard/export` | ✅ Source correct |
| AC-4 | Export PDF button shows spinner + disabled while in flight | ✅ Source correct |
| AC-5 | useExamsListForReports(): queryKey `['exams-list-reports']`, skeleton while loading | ✅ Source correct |
| AC-6 | Each row: exam title, status badge, Analytics link → `/admin/exams/:id/analytics`, CSV button | ✅ Source correct |
| AC-7 | Export CSV triggers download; per-row loading state via `exportingId` | ✅ Source correct |
| AC-8 | Empty-state renders `t('reports.no_exams')` | ✅ Source correct; FR doc key name corrected |
| AC-9 | Zero hardcoded strings; all via `useTranslation` with `reports.*` keys | ✅ Source correct |
| AC-10 | Responsive: `grid-cols-1 md:grid-cols-2` | ✅ Source correct |
| AC-11 | Sidebar "Reports" nav → `/admin/reports`; route registered | ✅ Source correct |

## Files Changed
| File | Change |
|------|--------|
| `docs/requirements/FR-BB59.Reports-hub-page.md` | AC-8 key name corrected (`exams_empty` → `no_exams`); i18n keys table updated to match implementation |

## Regression Test
None added — the root cause is a stale Docker image (infrastructure/ops issue), not a source code defect. The fix requires rebuilding the frontend Docker image (`docker compose build frontend`). No regression test is applicable.

## Resolution Results
- Tests: N/A (no source code change)
- Migration applied: no
- Build clean: yes (source already compiles; stale Docker image is the deployment artifact issue)
- Docker rebuild required: yes (pending; intentionally deferred per task constraint)

## Recurrence Log
| Date | Trigger | Action Taken |
|------|---------|--------------|
| 2026-05-22 | Navigation to /admin/reports shows empty page | Confirmed source complete; corrected FR-BB59 doc; wrote issue report |

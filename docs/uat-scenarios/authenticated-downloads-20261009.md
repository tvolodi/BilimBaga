---
slug: authenticated-downloads
title: "Authenticated File Downloads Send Bearer Token — UAT Scenario"
feature: authenticated-downloads (GitHub issue #37; FR-BB45, FR-BB46, FR-BB58, FR-BB59)
version: 1
created: 2026-10-09
author: Business Analyst
---

Target: local | qa (default: local; never the production-class demo instance, see DEC-001)

## Code that must be merged before running

- **Issue #37** fix (`fix(frontend): authenticated file downloads send no Bearer token`; in-progress, no PR at authoring time) must be on `main` and the frontend rebuilt. Affected: `ResultActions.tsx`, `ResultsTable.tsx`, `downloadAdminCertificate` (`api/employees.ts`), `downloadDashboardPdf` and `downloadExamCsv` (`api/reports.ts`).
- Before the fix this scenario is expected to FAIL at S1-S6 (baseline: `docs/handoffs/ba-drift-check/report.md`); a pre-fix run is only a regression baseline and its failures are DEFECT.
- Independent of issue #39: S5/S6 use `super_admin`, so they do not depend on the `/admin/reports` guard change. Role access is covered by `route-guards-20261009.md`.

## Preconditions

1. Platform at `http://localhost`; API at `http://localhost/api/v1`.
2. **Super Admin** `admin@bilimbaga.local` / `Admin1234!`; `{admin_token}` from login.
3. **Employee** `uat.employee@test.com` / `NewPass123!`; `{employee_token}`.
4. **"UAT Result Exam"** (`certificate_enabled=true`) with a submitted passing session of uat.employee: `{passing_session_id}`. **"UAT No-Answers Exam"** (`certificate_enabled=false`) with a submitted session `{noAnswers_session_id}`. A submitted FAILED session `{failed_session_id}` for uat.employee is needed for S3 step 6. Create via S0a/S0b/S3/S5/S10g of `result-and-certification-20260609.md` if absent.
5. `{uatResultExamId}` — UUID of "UAT Result Exam" (admin exam list); at least one submitted session exists so the CSV has a data row.
6. `{employee_user_id}` — UUID of uat.employee (from `/admin/users`).
7. Browser context with downloads enabled (Playwright `waitForEvent('download')`); request headers inspectable via network log.
8. Backend `auth.Authenticate` accepts only `Authorization: Bearer`; cookies alone must not authenticate.

## Test data

| Name | Value |
|------|-------|
| Dashboard PDF range | from = 30 days ago, to = today |
| Certificate filename | `certificate-{uuid}.pdf` |
| Exam CSV | `text/csv`, header row + at least 1 data row |

---

## Scenario S1: Portal certificate download from result screen (FR-BB45 AC-6)

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Employee | Log in as uat.employee; open `/portal/sessions/{passing_session_id}/result` | Result screen with enabled "Download Certificate" button | |
| 2 | Employee | Click it, capturing network and download event | Download event fires; filename matches `certificate-{uuid}.pdf` | |
| 3 | Tester | Inspect request to `/api/v1/portal/sessions/{passing_session_id}/certificate` | `Authorization: Bearer <employee jwt>` present; HTTP 200; `application/pdf` | |
| 4 | Tester | Check the downloaded file | Starts with `%PDF-`; size > 1 KB | |
| 5 | Employee | Check page state | Still on the result screen; no navigation, no blank tab | |

## Scenario S2: Portal certificate download from My Results (FR-BB46 AC-8, AC-3)

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Employee | Open `/portal/results` | Row for "UAT Result Exam" with a Download Certificate control | |
| 2 | Employee | Click the control (capture network and download) | No full-page navigation; download event fires | |
| 3 | Tester | Inspect request | `Authorization: Bearer` present; HTTP 200 PDF | |
| 4 | Employee | Check URL | Still `/portal/results` | |
| 5 | Employee | Inspect rows of failed sessions or exams with `certificate_enabled=false` | No certificate control (FR-BB46 AC-3) | |

## Scenario S3: Error surfacing — portal (FR-BB45 AC-6, FR-BB46 AC-8)

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Tester | Intercept `GET **/portal/sessions/*/certificate` to return HTTP 500 | Interception active | |
| 2 | Employee | On the result screen click Download Certificate | Visible i18n error feedback (toast/alert, no raw key); no download; button usable again | |
| 3 | Employee | Repeat from a `/portal/results` row | Same visible error | |
| 4 | Tester | Intercept to return HTTP 401 envelope | App-standard session handling (refresh attempt, then `/login`); never a silent no-op | |
| 5 | Tester | Remove interception; call API `GET /api/v1/portal/sessions/{noAnswers_session_id}/certificate` with `{employee_token}` | HTTP 422 `EXAM_NOT_CERTIFIABLE` | |
| 6 | Tester | Same for `{failed_session_id}` | HTTP 422 `SESSION_NOT_PASSED` | |

## Scenario S4: Admin certificate download from Employee Record (FR-BB58 AC-4)

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Admin | Log in; open `/admin/users/{employee_user_id}/record` | Session history table loaded | |
| 2 | Admin | Find the "UAT Result Exam" row with a certificate link (run S1 first if the certificate does not exist yet) | Link visible | |
| 3 | Admin | Click it with capture on | Download fires; `GET /api/v1/admin/sessions/{passing_session_id}/certificate` has `Authorization: Bearer <admin jwt>`; HTTP 200 PDF starting `%PDF-` | |
| 4 | Tester | Intercept same endpoint to return 422 `SESSION_NOT_PASSED`; click again | i18n-keyed toast for that code; no download; no crash | |
| 5 | Tester | Intercept to return 500 | Generic error toast | |

## Scenario S5: Dashboard PDF export (FR-BB59 AC-3, AC-4)

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Admin | Open `/admin/reports` | Dashboard PDF Export card with From/To and Export PDF | |
| 2 | Admin | Set the range, click "Export PDF" | Button shows spinner and is disabled while in flight | |
| 3 | Admin | Await download | `GET /api/v1/admin/dashboard/export?from=...&to=...` carries `Authorization: Bearer`; HTTP 200 `application/pdf`; file starts `%PDF-` | |
| 4 | Admin | After completion | Button back to default (enabled, no spinner) | |
| 5 | Tester | Intercept export to return 500; click | i18n error toast; button reverts to default (AC-4) | |
| 6 | Tester | Intercept with network failure | Same handling | |

## Scenario S6: Exam results CSV export (FR-BB59 AC-7)

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Admin | On `/admin/reports` find the "UAT Result Exam" row | "Export CSV" button present | |
| 2 | Admin | Click "Export CSV" | Only that row's button shows loading state | |
| 3 | Admin | Await download | `GET /api/v1/admin/exams/{uatResultExamId}/results/export` carries `Authorization: Bearer`; HTTP 200 `text/csv`; header + at least 1 data row containing uat.employee | |
| 4 | Tester | Intercept to return 500; click | Error toast; loading cleared; other rows unaffected | |
| 5 | Tester | Intercept to return 403 | Permission error toast; no download | |

## Scenario S7: Negative control — Bearer still required

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Tester | API with no `Authorization`: `GET /api/v1/portal/sessions/{passing_session_id}/certificate` | HTTP 401, envelope `{data:null,error:{code,...}}` | |
| 2 | Tester | Same for `/api/v1/admin/dashboard/export` and `/api/v1/admin/exams/{uatResultExamId}/results/export` | HTTP 401 each | |

## Pass / Fail criteria

- PASS: S1-S6 downloads occur with `Authorization: Bearer` on all five code paths (result screen, My Results row, admin certificate, dashboard PDF, exam CSV); every injected failure shows visible feedback and resets loading state; S7 confirms the server still requires Bearer.
- FAIL (defect): any download request without Bearer; 401 on a valid session; silent failure on non-2xx; navigating away from the page; stuck spinner.
- ENV ISSUE: fix for #37 not merged; no certificate/session data; browser blocks downloads.

## Acceptance Criteria Coverage

| FR | AC# | Covered by |
|----|-----|------------|
| FR-BB45 | AC-6 | S1, S3 |
| FR-BB46 | AC-3, AC-8 | S2, S3 |
| FR-BB58 | AC-4 | S4 |
| FR-BB59 | AC-3, AC-4 | S5 |
| FR-BB59 | AC-7 | S6 |

## Out of Scope

Role access to `/admin/reports` (see `route-guards-20261009.md`); PDF layout; FR-BB51 completion-rate defect.

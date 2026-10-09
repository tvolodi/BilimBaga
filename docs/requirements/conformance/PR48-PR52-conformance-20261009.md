# Conformance Review: PR #48 (route guards) and PR #52 (authenticated downloads)

- Date: 2026-10-09
- Reviewer: Business Analyst (Mode C style, static code review; no live stack, no UAT executed)
- Base: origin/main at 3e1a535 (worktree `ba`, branch swarm/62-conformance-2)
- Scope: PR #48 (7224272, issue #39), PR #52 (2f7e473, issue #37)
- Scenarios: `docs/uat-scenarios/route-guards-20261009.md`, `docs/uat-scenarios/authenticated-downloads-20261009.md`
- FR-BB48 / PR #36: see `docs/requirements/conformance/FR-BB48-FR-BB64-conformance-20261009.md` (not repeated here)

## UAT report decision matrix (step 3)

No file in `docs/uat-reports/` covers route-guards or authenticated-downloads (only the 20260609 reports for older features exist). There is nothing to classify as PASS / DEFECT / REQ GAP / ENV ISSUE. Both scenarios are unexecuted: the code verdict below is static only, and the features stay "not uat-verified" until a UAT Runner run produces a report.

## 1. PR #48 - route guards (FR-BB114 AC-1, FR-BB59 AC-2)

| # | Check | Result | Evidence |
|---|-------|--------|----------|
| 1.1 | `/admin/audit` = super_admin only | PASS | `frontend/src/App.tsx:255` uses `AUDIT_READ_ROLES`; `frontend/src/lib/routeRoles.ts:5` = `['super_admin']` |
| 1.2 | `/admin/reports` = super_admin, department_admin, examiner (all roles with reports:read) | PASS | `App.tsx:263`; `routeRoles.ts:10` |
| 1.3 | hr_admin dropped from these two guards | PASS | `App.tsx:255,263` |
| 1.4 | Unauthorised signed-in user goes to `/login` | PASS | `frontend/src/components/RequireRole.tsx:18-30` (`unauthorizedRedirect`); other routes keep old `/admin` / `/portal` behaviour (backward compatible) |
| 1.5 | No bounce loop on `/login` | PASS | `frontend/src/pages/auth/LoginPage.tsx:17-35` only navigates inside the submit handler; there is no auto-redirect for an existing token. Scenario S7 step 4 is therefore satisfied statically. Side effect: a signed-in user is shown the login form, which is acceptable per spec |
| 1.6 | Tests | PASS | `frontend/src/lib/routeRoles.test.tsx` covers 5 roles x 2 guards plus the no-token case, asserting `/login` |
| 1.7 | i18n | PASS (n/a) | No user-visible strings added |
| 1.8 | Sidebar hides links the user cannot open (scenario S2 step 3 and role rows imply it) | GAP p3 | `frontend/src/components/admin/Sidebar.tsx:35-36` shows Reports and Audit to all admin roles. Already tracked as follow-up #49 |
| 1.9 | Residual `hr_admin` in other guards and role checks | GAP p3 | `App.tsx:134,145,203`, `AdminDashboardPage.tsx:13`, `ExamAnalyticsPage.tsx:254`, `EmployeeRecordPage.tsx:215`. The role does not exist in the DB (migration 005), so these are dead entries and not an access hole. Note `ADMIN_ROLES` in `AdminDashboardPage.tsx:13` excludes department_admin (pre-existing, out of scope) |

Verdict PR #48: PASS. Backend permissions were not changed, and the guard remains defence in depth only (the API still enforces 403).

## 2. PR #52 - authenticated downloads (issue #37)

Shared helper: `frontend/src/api/download.ts`. It sends `Authorization: Bearer` (lines 21-28), refreshes once on 401 (55-58), throws a typed `DownloadError` (59-63), uses blob and object URL with revoke in `finally` (65-75), and maps errors to i18n keys via `downloadErrorKey` (16-19). Locale keys `download.failed` / `download.session_expired` exist in en (`en.json:1022`) and, per the PR file list, in ru and kk.

| # | Requirement | Result | Evidence |
|---|-------------|--------|----------|
| 2.1 | FR-BB45 AC-6 result-screen certificate | PASS | `frontend/src/components/results/ResultActions.tsx:28-40` downloadFile; error `role="alert"` at 47-51; test `ResultActions.test.tsx:34-44` (Bearer, error shown) |
| 2.2 | FR-BB46 AC-8 / AC-3 My Results (`MyResultsPage` renders `ResultsTable`) | PASS | `frontend/src/components/results/ResultsTable.tsx:40-54`; `ResultsTable.test.tsx:113+`; certificate control hidden when not eligible (`ResultsTable.test.tsx:84-89`) |
| 2.3 | FR-BB58 AC-4 admin certificate | PASS with note | `api/employees.ts:150` -> downloadFile; `SessionHistoryTable.tsx:28-37` shows an error banner (auto-clears after 5 s); `download.test.ts:123` |
| 2.4 | FR-BB58 AC-4 code-specific toast (`SESSION_NOT_PASSED`, `EXAM_NOT_CERTIFIABLE`) and "toast" | GAP p3 | `downloadErrorKey` (`download.ts:16-19`) maps only ERR_UNAUTHORIZED vs generic `download.failed`. Spec and scenario S4 step 4 expect a message per server code; the server code is captured in `DownloadError.code` but unused. The inline banner is not a toast. `SessionHistoryTable` has no component test |
| 2.5 | FR-BB59 AC-3 dashboard PDF | PASS | `api/reports.ts:39`; `ReportsPage.tsx:41-46` surfaces error |
| 2.6 | FR-BB59 AC-7 exam CSV | PASS | `api/reports.ts:47-48`; `ReportsPage.tsx:122-129` |
| 2.7 | FR-BB59 AC-4 (loading state + error) | PASS | `ReportsPage.tsx:36-47` |
| 2.8 | Tests | PASS | `download.test.ts` (Bearer, no-token, 401 refresh + retry, second 401, server code, i18n mapping, each helper); component tests above; question export test in `QuestionBankPage.test.tsx`; e2e spec `frontend/e2e/downloads-bearer.spec.ts` written but never executed |
| 2.9 | Question bulk export (bonus) | PASS | `QuestionBankPage.tsx:786-788` |
| 2.10 | Remaining raw `<a href>` / `window.open` to authenticated files | PASS | None found (grep of `frontend/src` excluding tests) |
| 2.11 | Remaining raw `fetch` download helpers not migrated | GAP p2 | `frontend/src/components/analytics/ExportCSVButton.tsx:20-38` sends Bearer but has no 401 refresh and no catch: a failure leaves an unhandled rejection with no user message (silent failure). Same pattern: `frontend/src/api/audit.ts:117` (`exportAuditLog`, caller `AuditLogPage.tsx:77`) and `frontend/src/api/employees.ts:123` (`exportEmployeeRecord`, caller `EmployeeRecordPage.tsx:84-90`, `try/finally` with no catch). These are not in the listed ACs of issue #37 but violate its principle ("every download helper ... surfaces errors"). Not a security issue: a Bearer token is sent |

Verdict PR #52: PASS on all named ACs. Two p3/p2 gaps are listed above. Unverified because nothing ran against a live stack: the real 401 refresh, the actual `Content-Disposition` filename (the helper always uses the fallback filename, so the `certificate-{uuid}.pdf` pattern in scenario S1 step 2 holds only because the fallback is built that way, and the CSV filename may differ from the previous Content-Disposition filename), and the Playwright spec.

## Gap summary

| ID | Sev | Gap | Suggested action |
|----|-----|-----|------------------|
| G1 | p2 | ExportCSVButton, exportAuditLog and exportEmployeeRecord not on shared helper (no error surfacing, no 401 refresh) | Route them through `downloadFile` plus `downloadErrorKey` and add tests |
| G2 | p3 | FR-BB58 AC-4 code-specific error message and toast; no SessionHistoryTable test | Map `SESSION_NOT_PASSED` / `EXAM_NOT_CERTIFIABLE` to i18n keys; add a component test |
| G3 | p3 | Sidebar shows audit/reports to every admin role (#49) | Existing follow-up |
| G4 | p3 | Dead `hr_admin` entries in other guards | Cleanup |
| G5 | info | Neither UAT scenario has been executed; the downloads e2e spec is unexecuted | UAT Runner |

p1 gaps: none. No security or data-exposure issue found: guards were tightened (audit narrowed to super_admin), tokens go only in the Authorization header, and the backend remains the authority.

## README status recommendation (README not edited)

| FR | Current | Recommend | Reason |
|----|---------|-----------|--------|
| FR-BB114 | Partial | Implemented | The only drift item (AC-1) is fixed and tested (1.1, 1.6) |
| FR-BB59 | Partial | Implemented | AC-2, AC-3 and AC-7 fixed with tests (1.2, 1.4, 2.5-2.7) |
| FR-BB45 | Partial | Implemented | AC-6 fixed with component test (2.1) |
| FR-BB46 | Partial | Implemented | AC-8 / AC-3 fixed with tests (2.2) |
| FR-BB58 | Partial | Keep Partial until G2 is fixed (or accept the generic message and move it to Implemented) | AC-4 requires an i18n-keyed toast for server error codes; only a generic or session-expired banner exists and the component has no test |

Whether or not statuses move, none should be marked `uat-verified` until `route-guards-20261009.md` and `authenticated-downloads-20261009.md` are run.

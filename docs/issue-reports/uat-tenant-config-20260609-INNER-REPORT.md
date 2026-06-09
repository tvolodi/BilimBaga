# uat-tenant-config-20260609: Implementation Inner Report

**Date**: 2026-06-09T00:00:00Z
**Pipeline**: B (UAT PASS — documentation and status update)
**Commit**: 5bcdf51 (docs commit) / f6a1f7c (fix commit)

## Summary

Two UAT defects discovered during the tenant configuration UAT run were diagnosed and fixed. ISS-028 fixed the admin sidebar hardcoding the application name as the literal "BilimBaga" instead of reading from the tenant configuration API, meaning branding changes were not reflected in the admin shell. ISS-029 fixed a three-part cascading failure that prevented any department admin (and all users with a fresh browser session and no refresh cookie) from reaching the admin panel after successful login: (A) `RequireAuth` read the token cache synchronously before React Query had settled, causing immediate redirect to `/login`; (B) `useRefreshToken` overwrote the valid login token with `null` when the refresh endpoint returned 401; (C) `RequireRole` did not include `department_admin` in the dashboard route's allowed roles and redirected role mismatches to `/login` rather than `/admin`.

## Files Changed

| File | Action |
|------|--------|
| `frontend/src/components/admin/Sidebar.tsx` | modified — reads `app_name` from `useTenantConfig` (ISS-028) |
| `frontend/src/components/admin/Sidebar.test.tsx` | modified — added mock and regression tests for dynamic app name (ISS-028) |
| `frontend/src/components/RequireAuth.tsx` | modified — replaced `getQueryData` with `getQueryState` + `FullPageSpinner` (ISS-029 Fix A) |
| `frontend/src/components/RequireAuth.test.tsx` | created — 3 regression tests for RequireAuth states (ISS-029) |
| `frontend/src/api/auth.ts` | modified — preserve cached token when refresh returns 401 (ISS-029 Fix B) |
| `frontend/src/App.tsx` | modified — add `department_admin` to dashboard route allowed roles (ISS-029 Fix C) |
| `frontend/src/components/RequireRole.tsx` | modified — redirect role mismatch to `/admin` instead of `/login` (ISS-029 Fix C) |
| `frontend/src/components/RequireRole.test.tsx` | modified — updated test expectations to `/admin` redirect (ISS-029 Fix C) |
| `frontend/src/pages/admin/ReportsPage.test.tsx` | modified — updated AC-2 test to expect `/admin` redirect (ISS-029 Fix C) |
| `docs/issue-reports/ISS-028-uat-defect-sidebar-app-name-hardcoded.md` | created — ISS-028 root cause and fix documentation |
| `docs/issue-reports/ISS-029-uat-defect-dept-admin-login-redirect-loop.md` | created — ISS-029 root cause (3 issues A/B/C) and fix documentation |

## Acceptance Criteria Verified

| AC | Verified By |
|----|-------------|
| ISS-028: Sidebar app name reflects tenant config | test: "displays app_name from tenant config instead of hardcoded literal" (Sidebar.test.tsx) |
| ISS-028: Fallback to "BilimBaga" when config not loaded | test: "falls back to 'BilimBaga' when tenant config is not yet loaded" (Sidebar.test.tsx) |
| ISS-029 Fix A: RequireAuth shows spinner during query init | test: "renders spinner while auth query is initialising" (RequireAuth.test.tsx) |
| ISS-029 Fix A: RequireAuth redirects when token is null | test: "redirects to /login when token is null" (RequireAuth.test.tsx) |
| ISS-029 Fix A: RequireAuth renders children when authenticated | test: "renders children when token is present" (RequireAuth.test.tsx) |
| ISS-029 Fix C: RequireRole redirects to /admin on mismatch | test: updated assertions in RequireRole.test.tsx and ReportsPage.test.tsx |

## Test Results

- Backend: no changes — not run
- Frontend: 230 passed, 0 failed (41 test files)

## Migration Applied

none

## UAT Verdict

**PASS** — All tenant configuration acceptance criteria verified on retry 2. FR-BB13 status updated to `uat-verified`.

| Retry | Outcome | Defects Found |
|-------|---------|---------------|
| Initial run | FAIL | ISS-028 (sidebar hardcoded app name), ISS-029 (dept-admin login redirect loop) |
| Retry 1 | FAIL | ISS-029 Fix B/C edge case — RequireRole redirect target still wrong for some paths |
| Retry 2 | PASS | All AC verified, no new defects |

## Known Limitations

- ISS-029 Fix B (preserving cached token when refresh returns 401) relies on the access token being present in the React Query cache at the time the refresh fires. If the token was never written (e.g. SSR or a very unusual race on a slow device), the behavior is unchanged — user is redirected to `/login` as before.
- The `RequireRole` redirect-to-`/admin` on mismatch means that users with a valid session but insufficient role for a deep link will land on `/admin` rather than seeing an explicit "Access Denied" page. A dedicated 403 page is deferred.
- Full E2E Playwright suite re-run is recommended via Pipeline E2E to confirm all 22 scenarios pass end-to-end.

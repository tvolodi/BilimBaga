---
id: ISS-029
title: "UAT Defect: Tenant Configuration — Department Admin redirected back to /login after successful authentication"
status: resolved
severity: high
layer: frontend
module: RequireAuth / useRefreshToken
tags: [uat, auth, requireauth, refresh-token, department-admin, login]
created: 2026-06-09
resolved: 2026-06-09
recurrence_count: 1
related_issues: [ISS-019]
regression_test: src/components/RequireAuth.test.tsx
---

## Symptom
UAT Scenario: `Scenario 3: Department Admin Cannot Access Branding Settings`, Step 1
Actor: Department Admin
Action: Log in as `dept.admin@test.com` / `DeptUAT1!`
Expected: Admin shell visible (the dept admin should reach `/admin` dashboard)
Actual: Login API call returns HTTP 200 and `/users/me` returns HTTP 200, but the frontend immediately redirects back to `/login`. The admin shell is never reached. All subsequent steps in Scenario 3 are blocked.
Screenshot: playwright-report-live/uat-s3-dept-admin-dashboard.png, playwright-report-live/uat-s3-branding-access-attempt.png

## Root Cause

Three compounding issues caused the redirect loop:

**Issue A — RequireAuth race condition**: `RequireAuth` read the access token via `qc.getQueryData(['auth', 'accessToken'])` — a synchronous cache lookup that returns `undefined` when the `useRefreshToken` query has not yet settled. After a fresh `useLogin` mutation navigates to `/admin`, the React Router render of `RequireAuth` runs synchronously in the same tick before React Query propagates the mutation side-effect to the cache.

**Issue B — useRefreshToken overwrites login token**: After login, `useRefreshToken`'s `queryFn` fires on mount at `/admin` and calls `POST /auth/refresh`. With no refresh cookie, this returns 401 and the queryFn returned `null`, overwriting the valid token the login mutation just placed in the cache.

**Issue C — RequireRole routes dept admin to /login**: The `/admin/dashboard` route was wrapped with `RequireRole(['examiner', 'hr_admin', 'super_admin'])` — `department_admin` was excluded. The `/admin` index redirects to `/admin/dashboard`, so `RequireRole` would redirect dept admin back to `/login` on any landing.

## Fix Applied

Three targeted fixes:

**Fix A — RequireAuth**: Replaced synchronous `getQueryData` with `getQueryState` to distinguish loading vs unauthenticated.

**Fix B — useRefreshToken**: When `/auth/refresh` returns non-2xx AND there is already a valid cached token (placed by the login mutation), preserve the existing token instead of returning null.

**Fix C — App.tsx + RequireRole**: Added `department_admin` to the dashboard route's allowed roles. Changed `RequireRole`'s role-mismatch redirect from `/login` to `/admin` to prevent logout-on-access-denied.

## Files Changed

- `frontend/src/components/RequireAuth.tsx` — Fix A: use `getQueryState` + `FullPageSpinner`
- `frontend/src/components/RequireAuth.test.tsx` — new regression test (3 cases)
- `frontend/src/api/auth.ts` — Fix B: preserve cached token when refresh returns 401
- `frontend/src/App.tsx` — Fix C: add `department_admin` to dashboard route allowed roles
- `frontend/src/components/RequireRole.tsx` — Fix C: redirect to `/admin` (not `/login`) on role mismatch

## Regression Test

`src/components/RequireAuth.test.tsx` — three cases:
1. Auth query not settled (no cache entry) → spinner rendered, no redirect to `/login` (ISS-029 regression guard).
2. Auth query settled with `null` → redirects to `/login`.
3. Auth query settled with a token → renders children.

## Resolution Results

All 228 frontend tests passed (41 test files, 0 failures) after the fix.
`npm test -- --run` exit code: 0.

---

## Evidence

`src/components/RequireAuth.tsx` reads the access token via:
```tsx
qc.getQueryData(['auth', 'accessToken'])
```
This is a **synchronous cache read**. The sequence of events for a dept admin in a fresh browser session:

1. Page loads — `useRefreshToken` fires `POST /auth/refresh`.
2. No httpOnly refresh cookie exists for dept admin → `/auth/refresh` returns 401.
3. Cache stays `undefined`; `RequireAuth` synchronously reads `undefined` → redirects to `/login`.
4. User fills in credentials → login mutation fires → API returns 200 + sets access token in cache.
5. React Router navigates to `/admin` → `RequireAuth` re-evaluates **synchronously** before `useRefreshToken` query settles.
6. Cache is still `undefined` at the instant of the synchronous check → redirected back to `/login`.

Scope: affects ALL non-super-admin users who start in a fresh browser session (no pre-existing httpOnly refresh cookie). Super Admin is unaffected in UAT only because Playwright storageState carries the super_admin cookie.

Related: ISS-019 was a similar race condition with `RequireRole`; this is the analogous issue in `RequireAuth` for users without a prior refresh cookie.

## Expected Behaviour

After a successful login (API 200), the user should be navigated to `/admin` and `RequireAuth` should recognise the authenticated session without bouncing the user back to `/login`.

## Acceptance Criteria Impacted

- AC#6: "A Department Admin does not see the Branding settings page; only Super Admin can access it" — RBAC protection of the branding route is completely untestable while dept admin cannot log in at all.
- Requirement: FR-BB112

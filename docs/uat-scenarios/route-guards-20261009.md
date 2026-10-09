---
slug: route-guards
title: "Admin Route Guards (/admin/audit, /admin/reports) — UAT Scenario"
feature: route-guards (GitHub issue #39; FR-BB114 AC-1, FR-BB59 AC-2, FR-BB16)
version: 1
created: 2026-10-09
author: Business Analyst
---

Target: local | qa (default: local; never the production-class demo instance, see DEC-001)

## Code that must be merged before running

- **Issue #39** fix (`fix(frontend): align /admin/audit and /admin/reports route guards`; status ready, no PR at authoring time) must be on `main` and the frontend rebuilt.
- Baseline in `frontend/src/App.tsx`: `/admin/audit` allows `super_admin, hr_admin, examiner`; `/admin/reports` allows `super_admin, examiner, hr_admin` (excludes `department_admin`); `RequireRole` sends an authenticated but unauthorised user to `/admin` (or `/portal` for employee), and to `/login` only when there is no token. A pre-fix run is expected to FAIL S3, S4, S5.
- Implementer note: FR-BB59 AC-2 and issue #39 require redirect to `/login`. If the fix navigates to `/login` while a token exists, `/login` must not bounce the user straight back to `/admin` (checked in S7 step 4).

## Role/permission facts (from `backend/migrations/005_rbac.up.sql`)

| Role | audit:read | reports:read |
|------|-----------|--------------|
| super_admin | yes | yes |
| department_admin | no | yes |
| examiner | no | yes |
| employee | no | no |

`hr_admin` was renamed to `examiner` in migration 005; no `hr_admin` role exists in the DB, so no JWT can carry it (it is not tested and should be dropped from guard lists).

Expected route matrix:

| Route | super_admin | department_admin | examiner | employee | anonymous |
|-------|-------------|------------------|----------|----------|-----------|
| `/admin/audit` | ALLOW | `/login` | `/login` | `/login` | `/login` |
| `/admin/reports` | ALLOW | ALLOW | ALLOW | `/login` | `/login` |

## Preconditions and accounts (GAPS)

- Seeded by migrations: ONLY `admin@bilimbaga.local` (`super_admin`, migration 029; `Admin1234!`).
- Exists from earlier UAT scenarios: `uat.employee@test.com` / `NewPass123!` (`employee`).
- **Do NOT exist and must be created by the UAT Runner as admin (S0):**

| Email | Role | Password |
|-------|------|----------|
| `uat.deptadmin@test.com` | `department_admin` | `NewPass123!` |
| `uat.examiner@test.com` | `examiner` | `NewPass123!` |

New users may have `force_password_change=true`; complete the change-password flow once (keep `NewPass123!`) before role tests.

Other: platform at `http://localhost`; fresh browser context per role; capture `{role}_token` per login.

---

## Scenario S0: Create test accounts (if absent)

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Admin | Log in as admin; open `/admin/users` | Users list | |
| 2 | Admin | Check for `uat.deptadmin@test.com` and `uat.examiner@test.com` | Present, else proceed | |
| 3 | Admin | Create `uat.deptadmin@test.com`, role `department_admin`, any department, password `NewPass123!` | User created, active | |
| 4 | Admin | Create `uat.examiner@test.com`, role `examiner`, password `NewPass123!` | User created, active | |
| 5 | Each new user | Log in once; complete forced password change if prompted | Lands on `/admin/dashboard` | |

## Scenario S1: Anonymous

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Anonymous | Open `/admin/audit` | Redirected to `/login` | |
| 2 | Anonymous | Open `/admin/reports` | Redirected to `/login` | |

## Scenario S2: super_admin

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | super_admin | Log in; navigate to `/admin/audit` | Audit Log page renders; URL stays; audit API call returns HTTP 200 | |
| 2 | super_admin | Navigate to `/admin/reports` | Reports hub renders; URL stays | |
| 3 | super_admin | Check sidebar | Audit Log and Reports links visible and working | |

## Scenario S3: department_admin

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | department_admin | Log in as `uat.deptadmin@test.com`; navigate (typed URL) to `/admin/reports` | Reports hub renders; URL stays; no redirect | |
| 2 | department_admin | Check reports API calls | HTTP 200 (reports:read) | |
| 3 | department_admin | Navigate to `/admin/audit` | Redirected to `/login`; audit content never rendered | |
| 4 | Tester | API with dept-admin token: `GET /api/v1/audit` (route: router.go `Get("/audit")`; confirm the mount prefix) | HTTP 403 | |
| 5 | department_admin | Check sidebar | Reports link visible; Audit Log hidden | |

## Scenario S4: examiner

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | examiner | Log in as `uat.examiner@test.com`; navigate to `/admin/reports` | Reports hub renders | |
| 2 | examiner | Navigate to `/admin/audit` | Redirected to `/login`; page never renders (examiner lacks audit:read) | |
| 3 | Tester | API with examiner token: audit log endpoint | HTTP 403 | |
| 4 | examiner | Check sidebar | Reports visible; Audit Log hidden | |

## Scenario S5: employee

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | employee | Log in as `uat.employee@test.com`; navigate to `/admin/reports` | Redirected to `/portal` (BA decision on #39: employee home, FR-BB111 AC-5); Reports never rendered | |
| 2 | employee | Log in again if needed; navigate to `/admin/audit` | Redirected to `/portal` | |
| 3 | Tester | API with employee token: `GET /api/v1/admin/dashboard/export` and the audit log endpoint | HTTP 403 each | |

## Scenario S6: Deep link and history behaviour

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | super_admin | On `/admin/audit` press F5 | Page re-renders for super_admin; not bounced to `/login` | |
| 2 | department_admin | On `/admin/reports` press F5 | Page re-renders; no bounce | |
| 3 | examiner | Visit `/admin/reports`, then use browser back/forward to `/admin/audit` | Still redirected to `/login` | |

## Scenario S7: Redirect target quality and regression

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Tester | For each denied case (S3 step 3, S4 step 2, S5 steps 1-2) record the final URL path | `/login` for admin-area roles; `/portal` for employee (S5); never `/admin` or blank | |
| 2 | Tester | Check DOM during redirect | No audit table or reports cards ever rendered | |
| 3 | Tester | Check console | No errors or 403 toasts caused by the redirect | |
| 4 | Tester | After redirect, check the login screen | Login form shown and usable; no redirect loop back to `/admin`; logging in as super_admin reaches `/admin/dashboard` | |
| 5 | department_admin | Open `/admin/users`, `/admin/exams`, `/admin/questions` | Still allowed (other guards unchanged) | |

## Pass / Fail criteria

- PASS: every cell of the route matrix observed; API 403 checks agree with UI guards; denied roles land on `/login` with no flash of protected content.
- FAIL (defect): examiner or other non-super_admin sees `/admin/audit`; department_admin cannot see `/admin/reports`; unauthorised role lands on `/admin` or `/portal` instead of `/login`; redirect loop.
- ENV ISSUE: fix for #39 not merged; test accounts not creatable.
- If the implementation keeps redirecting to `/admin`, escalate to BA Mode C as REQ question: FR-BB59 AC-2 and issue #39 say `/login`; FR-BB114 does not specify the target.

## Acceptance Criteria Coverage

| FR | AC# | Covered by |
|----|-----|------------|
| FR-BB114 | AC-1 (audit viewer super_admin only) | S2, S3 steps 3-4, S4 steps 2-3, S5 |
| FR-BB59 | AC-2 (allowed roles; others to /login) | S1, S2, S3 steps 1-2, S4 step 1, S5, S7 |
| FR-BB16 | RBAC alignment (audit:read, reports:read) | API checks in S3-S5 |

## Out of Scope

Guards on other admin routes; removing `hr_admin` beyond the two guard lists; showing a 403 page instead of redirecting.

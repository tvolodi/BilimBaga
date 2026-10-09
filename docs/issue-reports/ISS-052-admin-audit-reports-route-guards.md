---
id: ISS-052
title: /admin/audit and /admin/reports route guards disagree with RBAC (hr_admin role, wrong redirect)
status: resolved
severity: medium
layer: frontend
module: audit
tags: [RequireRole, hr_admin, audit:read, reports:read, FR-BB114, FR-BB59, route-guard]
created: 2026-10-09
resolved: 2026-10-09
recurrence_count: 1
related_issues: []
regression_test: frontend/src/lib/routeRoles.test.tsx
---

## Symptom
GitHub #39 (parent #34, evidence docs/handoffs/ba-drift-check/report.md). `/admin/audit` was guarded with `super_admin, hr_admin, examiner`, so examiners reached a page whose API returns 403 (`audit:read` is super_admin only). `/admin/reports` was guarded with `super_admin, examiner, hr_admin`, excluding `department_admin` who holds `reports:read`. `hr_admin` is not a seeded role. Unauthorised roles were redirected to /admin or /portal instead of /login (FR-BB59 AC-2).

## Root Cause
Hand-written role arrays in `App.tsx` drifted from the RBAC seed (`backend/migrations/005_rbac.up.sql`), and `RequireRole` always redirected to the role's home.

## Fix Applied
- New `frontend/src/lib/routeRoles.ts`: `AUDIT_READ_ROLES = [super_admin]`, `REPORTS_READ_ROLES = [super_admin, department_admin, examiner]` (examiner holds reports:read in the seed).
- `RequireRole` gained optional `unauthorizedRedirect`; default behaviour unchanged for all other routes.
- `App.tsx` audit/reports routes use the constants with `unauthorizedRedirect="/login"`.

## Files Changed
| File | Change |
|------|--------|
| frontend/src/lib/routeRoles.ts | New role constants |
| frontend/src/components/RequireRole.tsx | `unauthorizedRedirect` prop |
| frontend/src/App.tsx | Use constants on /admin/audit, /admin/reports |
| frontend/src/lib/routeRoles.test.tsx | Role x route matrix tests |

## Regression Test
`frontend/src/lib/routeRoles.test.tsx`: for both guards, each of super_admin, department_admin, examiner, hr_admin, employee, plus no token, asserts render vs. /login redirect.

## Resolution Results
- Tests: 309 passed, 0 failed (54 files); tsc, eslint, i18n check clean
- Migration applied: no
- Build clean: yes

## Recurrence Log
| Date | Trigger | Action Taken |
|------|---------|-------------- |

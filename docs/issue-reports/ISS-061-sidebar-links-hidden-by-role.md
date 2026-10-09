---
id: ISS-061
title: Sidebar shows /admin/audit and /admin/reports links to roles the route guards reject
status: resolved
severity: low
layer: frontend
module: audit
tags: [Sidebar, routeRoles, AUDIT_READ_ROLES, REPORTS_READ_ROLES, RequireRole]
created: 2026-10-09
resolved: 2026-10-09
recurrence_count: 1
related_issues: [ISS-052]
regression_test: frontend/src/components/admin/Sidebar.test.tsx
---

## Symptom
Sidebar rendered Audit and Reports links for every role; clicking them as a role lacking audit:read / reports:read redirected away (GitHub #49).

## Root Cause
`NAV_ITEMS` in `Sidebar.tsx` was static and not filtered by role, unlike the route guards added in ISS-052.

## Fix Applied
Added optional `roles` to nav items (reports: `REPORTS_READ_ROLES`, audit: `AUDIT_READ_ROLES`); Sidebar reads the role from the cached access token (new `jwtRole` helper in `lib/routeRoles.ts`) and filters. Items without `roles` remain visible.

## Files Changed
| File | Change |
|------|--------|
| frontend/src/components/admin/Sidebar.tsx | filter nav by role |
| frontend/src/lib/routeRoles.ts | add `jwtRole` helper |
| frontend/src/components/admin/Sidebar.test.tsx | per-role tests |

## Regression Test
`Sidebar.test.tsx` it.each over super_admin, department_admin, examiner, employee.

## Resolution Results
- Tests: 346 passed, 0 failed (59 files)
- Migration applied: no
- Build clean: yes (tsc, eslint)

## Recurrence Log
| Date | Trigger | Action Taken |
|------|---------|--------------|

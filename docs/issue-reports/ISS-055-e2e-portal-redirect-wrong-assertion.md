---
id: ISS-055
title: E2E full-walkthrough test 21 asserts wrong redirect for admin session on /portal
status: resolved
severity: low
layer: frontend
module: auth
tags: [full-walkthrough, test 21, /portal, RequireRole, storageState, chromium-live-admin]
created: 2026-10-09
resolved: 2026-10-09
recurrence_count: 1
related_issues: [ISS-052]
regression_test: frontend/e2e/full-walkthrough.spec.ts
---

## Symptom
Live E2E test 21 (`full-walkthrough.spec.ts`) runs in `chromium-live-admin` (admin storageState), goes to `/portal`, expects URL `/login|/portal`, but lands on `/admin/dashboard`. GitHub issue #59.

## Root Cause
Test defect, not a product bug. `/portal` is wrapped in `RequireRole roles={['employee']}` (App.tsx); a signed-in non-employee is sent to `/admin` (then `/admin/dashboard`). FR-BB313 only requires that unauthenticated users go to `/login`; it does not allow admins into the employee portal. The test comment ("without an employee session -> login") assumed no session, but the project's admin storageState restores one.

## Fix Applied
Split into two tests: 21a uses a fresh browser context with empty storageState and asserts `/login` (FR-BB313 auth guard); 21b uses the admin session and asserts redirect to `/admin`.

## Files Changed
| File | Change |
|------|--------|
| frontend/e2e/full-walkthrough.spec.ts | test 21 -> 21a (unauthenticated -> /login) and 21b (admin -> /admin) |

## Regression Test
The updated e2e tests themselves (live run not performed by this agent: stack owned by UAT).

## Resolution Results
- Tests: vitest 336 passed, 0 failed
- Build clean: yes (tsc --noEmit, eslint clean, playwright --list shows 21a/21b)
- Migration applied: no
- Live E2E: not run

## Recurrence Log
| Date | Trigger | Action Taken |
|------|---------|--------------|

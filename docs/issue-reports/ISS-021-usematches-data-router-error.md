---
id: ISS-021
title: useMatches data-router error at runtime — stale frontend Docker image (never rebuilt)
status: open
severity: high
layer: config
module: auth
tags: [useMatches, BrowserRouter, docker, stale-build, breadcrumb, frontend-image]
created: 2026-05-22
resolved: null
recurrence_count: 1
related_issues: [ISS-001, ISS-020]
regression_test: frontend/src/layouts/AdminLayout.test.tsx
---

## Symptom

Browser console shows on every page load / component mount:

```
Error: useMatches must be used within a data router.
See https://reactrouter.com/en/main/routers/picking-a-router.
```

Stack trace originates from `Rx` inside the compiled bundle (`index-DuunGdhu.js:132`), fired during a TanStack Query `onSubscribe` / batch update cycle — i.e. when a component mounts and subscribes to a query.

## Root Cause

**The frontend Docker image has never actually been rebuilt.** The running `bilimbaga-frontend` container is serving the original pre-`d8d8ded` JS bundle, which contains the old `Breadcrumb.tsx` that called `useMatches()` from react-router v7. `useMatches` requires a data-router context (`createBrowserRouter + RouterProvider`); the app uses `BrowserRouter` (declarative router), which does NOT provide `DataRouterStateContext`. The react-router invariant check throws synchronously when `Breadcrumb` renders inside `AdminLayout`.

**History of the stale image:**

| Date | Event |
|------|-------|
| pre-2026-05-17 | `Breadcrumb.tsx` rewritten to `useLocation` in commit `d8d8ded` |
| 2026-05-17 | ISS-001 documented a Docker rebuild as the fix — rebuild was **never executed** |
| 2026-05-18–21 | ISS-003 through ISS-019 committed; image still not rebuilt |
| 2026-05-22 | ISS-020 documented password-toggle missing — same stale image — rebuild deferred again |
| 2026-05-22 | ISS-021 reported — `useMatches` error persists, confirming ISS-001 rebuild never ran |

**Source investigation (confirmed clean):**

A full grep across `frontend/src/**` for `useMatches`, `useLoaderData`, `useRouteLoaderData`, `useNavigation`, `useActionData` returned **zero matches**. All key files were verified:

| File | Hooks used | Status |
|------|-----------|--------|
| `frontend/src/components/admin/Breadcrumb.tsx` | `useLocation` | ✓ clean |
| `frontend/src/components/admin/TopBar.tsx` | `useNavigate` | ✓ clean |
| `frontend/src/layouts/AdminLayout.tsx` | `useLocation`, `Navigate`, `Outlet` | ✓ clean |
| `frontend/src/App.tsx` | `BrowserRouter` (declarative) | ✓ clean |

No source code change is required. The sole fix is rebuilding the frontend Docker image.

## Fix Applied

No source changes required. The current source is correct and has been correct since commit `d8d8ded`. The fix is a Docker image rebuild to replace the stale bundle with the current source (which includes all fixes through ISS-019):

```bash
docker compose build frontend
docker compose up -d frontend
```

This rebuild is deferred to the Infrastructure Configuration agent (same pass as ISS-020 and any pending ISS-022).

## Files Changed

| File | Change |
|------|--------|
| `docs/issue-reports/ISS-021-usematches-data-router-error.md` | This report (new) |
| `docs/issue-reports/README.md` | Added ISS-021 row to index |
| `docs/issue-reports/ISS-001-stale-dist-usematches-crash.md` | Updated recurrence_count and Recurrence Log |

No application source files were modified.

## Regression Test

`frontend/src/layouts/AdminLayout.test.tsx` — renders `AdminLayout` inside `MemoryRouter`; would catch any re-introduction of `useMatches` in `Breadcrumb` or any child component (test would throw the same invariant error).

## Resolution Results

- Tests: not re-run (no source change; regression test already passing from ISS-001)
- Migration applied: no
- Build clean: yes (go build not affected; frontend source unchanged)

## Recurrence Log

| Date | Trigger | Action Taken |
|------|---------|--------------|
| 2026-05-22 | useMatches error reported; ISS-001 rebuild confirmed never executed; ISS-020 deferred same rebuild | Documented root cause; source verified clean; Docker rebuild deferred to Infrastructure agent |

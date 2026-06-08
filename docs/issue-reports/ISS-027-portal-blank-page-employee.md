---
id: ISS-027
title: Employee portal renders blank after login — no error boundary catches render crash
status: resolved
severity: high
layer: frontend
module: portal
tags: [portal, useCountdown, TDZ, TemporalDeadZone, ErrorBoundary, ChangePasswordPage, currentUser, GarbageCollection, PortalExamItem, show_answers, blank_page]
created: 2026-06-08
resolved: 2026-06-08
recurrence_count: 1
related_issues: [ISS-019]
regression_test: frontend/src/pages/EmployeePortal/EmployeePortal.test.tsx
---

## Symptom
After logging in as an employee (role `"employee"`), the page `http://localhost/portal` renders
completely blank — white page, no content, no layout shell, no error message. The URL stays at
`/portal` (no redirect). The backend portal API returns HTTP 200 with exam data (confirmed in
nginx access logs), proving the `EmployeePortal` component mounts and calls `usePortalExams`,
but the page goes blank when exam cards are about to render.

## Root Cause

Three compounding bugs combine to produce the symptom:

### Bug 1 — TDZ crash in `useCountdown` for expired exams (primary cause)
`useCountdown.ts` declares `const interval = setInterval(...)` AFTER calling `tick()` synchronously.
When `tick()` runs for a past deadline (`remaining <= 0`), it tries `clearInterval(interval)` while
`interval` is still in the temporal dead zone — throwing `ReferenceError: Cannot access 'interval'
before initialization`. Because there is no error boundary, React 18 in production silently unmounts
the entire component tree, producing a blank page with no error message.

Any employee with at least one **expired** exam in their assigned list triggers this crash on the
first render of `EmployeePortal`.

### Bug 2 — No error boundary (amplifies all render crashes)
The entire React app has zero error boundaries. React 18 in production mode silently unmounts the
full component tree when any component throws during render, leaving a blank `<div id="root">`.
This turned Bug 1 into an invisible, hard-to-diagnose blank page.

### Bug 3 — `ChangePasswordPage` reads `currentUser.role` instead of JWT
`ChangePasswordPage.handleSubmit` decides whether to navigate to `/portal` or `/admin` using
`currentUser.role` from the query cache. After ISS-019's fix, `['auth', 'currentUser']` has no
active observer and can be garbage-collected after ~5 min. If that happens, the employee gets
redirected to `/admin` instead of `/portal`. Consistent with ISS-019: role should come from JWT.

### Bug 4 — `PortalExamItem` missing fields present in frontend `PortalExam` type
The backend list endpoint omits `show_answers`, `shuffle_questions`, `shuffle_options`, and
`certificate_enabled` from the response. The frontend type treats them as required. The
`ExamCard` uses `exam.show_answers === 'never'` — when the field is `undefined`, the comparison
silently evaluates to `false`. Adding these fields closes the type gap.

## Fix Applied

1. **Fixed TDZ crash in `useCountdown.ts`**: Changed `const interval` to `let interval` declared
   before the `tick` function so the closure can safely call `clearInterval(interval)` without a
   temporal dead zone error. Moved the assignment to after `tick()` is called.

2. **Added `ErrorBoundary` class component** (`frontend/src/components/ErrorBoundary.tsx`)
   with a user-friendly error screen. Wrapped `<AppRoutes />` in `AppRoot` so any per-route
   render crash shows a recoverable error screen instead of a blank page.

3. **Fixed `ChangePasswordPage.handleSubmit`** to decode the role from the JWT access token
   (same pattern as `RequireRole` after ISS-019) instead of reading the potentially-GC'd
   `currentUser` cache entry.

4. **Added `show_answers`, `shuffle_questions`, `shuffle_options`, `certificate_enabled` to
   `PortalExamItem`** in the backend and mapped them in the service layer, so the list
   endpoint returns the same fields as the detail endpoint.

5. **Added 7 unit tests** for `EmployeePortal` (loading → exam cards → empty state → error → expired → disabled CTA).

## Files Changed
| File | Change |
|------|--------|
| `frontend/src/hooks/useCountdown.ts` | Fix TDZ: `const interval` → `let interval` declared before `tick` |
| `frontend/src/components/ErrorBoundary.tsx` | New — class-based error boundary |
| `frontend/src/App.tsx` | Wrap AppRoutes in ErrorBoundary |
| `frontend/src/pages/auth/ChangePasswordPage.tsx` | Decode role from JWT instead of currentUser |
| `backend/internal/portal/model.go` | Add show_answers/shuffle_questions/shuffle_options/certificate_enabled to PortalExamItem |
| `backend/internal/portal/service.go` | Map new fields in ListMyExams |
| `frontend/src/pages/EmployeePortal/EmployeePortal.test.tsx` | New — 7 unit tests |

## Regression Test
`frontend/src/pages/EmployeePortal/EmployeePortal.test.tsx`
— covers: loading skeleton, exam card render (not_started), empty state, error state, expired exam (no CTA), failed exam (View result), disabled CTA (show_answers=never)

## Resolution Results
- Tests: 225 frontend passed, 0 failed (40 test files); all backend packages passed
- Migration applied: no
- Build clean: yes (go build ./..., vitest --run both clean)

## Recurrence Log
| Date | Trigger | Action Taken |
|------|---------|--------------|
| 2026-06-08 | Employee sees blank portal after login | Bug identified; error boundary + navigation fix + type fix applied |

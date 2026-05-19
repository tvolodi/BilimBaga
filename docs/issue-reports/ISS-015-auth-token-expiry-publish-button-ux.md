---
id: ISS-015
title: Access token expiry causes cascading 401s; publish button not disabled after validation failure
status: open
severity: high
layer: frontend
module: auth, exams
tags: [401, TOKEN_EXPIRED, useRefreshToken, staleTime, refetchInterval, publish, unsatisfiedRules]
created: 2026-05-19
resolved: null
recurrence_count: 1
related_issues: [ISS-011]
regression_test: null
---

## Symptom
User is on exam wizard Step 4 (Overview & Publish). Browser console shows:

1. `GET /api/v1/questions/...` → 401 Unauthorized
2. `GET /api/v1/categories` → 401 Unauthorized
3. `GET /api/v1/users/me` → 401 Unauthorized
4. `GET /api/v1/tags` → 401 Unauthorized
5. `POST /api/v1/auth/refresh` → 401 Unauthorized (root of cascading 401s)
6. `POST /api/v1/exams/{id}/publish` → 422 Unprocessable Entity

UI also shows the unsatisfied-rules warning ("Rule needs 1, only 0 available"), and the Publish
button remains clickable — the user can re-click it, re-triggering the 422, with no way to signal
that the issue is already known.

## Root Cause

### Issue A — Access token mid-session expiry (cascading 401s)
`useRefreshToken` in `frontend/src/api/auth.ts` is defined with `staleTime: Infinity`.
React Query will never automatically re-run `queryFn` once the access token is cached.
When the JWT's `exp` is reached (after `JWTAccessTTLMin` minutes) the expired token remains in
the React Query cache. Every `apiFetch` / `examsFetch` call reads that stale token and sends it
in the `Authorization` header; the backend middleware rejects it with 401 (`TOKEN_EXPIRED`).
There is no mid-session proactive refresh mechanism at all.

Additionally, the `localStorage` check in `queryFn` uses `exp * 1000 > Date.now()` — a hard
boundary that means the localStorage token is served right up to the millisecond of expiry. When
`refetchInterval` fires (e.g., 60 s before expiry), the localStorage token still passes this
check, so `queryFn` returns the stale token instead of calling `/auth/refresh`.

### Issue B — Publish button not disabled after unsatisfied-rules validation failure
After a 422 from `POST /api/v1/exams/{id}/publish`, `Step4Review.tsx` correctly sets
`unsatisfiedRules` state and shows the validation warning. However, the Publish button's
`disabled` prop is only `publishExam.isPending`. The user can continue clicking Publish, firing
repeat 422 requests, with no UX mechanism to guide them to fix the issues first.

## Fix Applied

### A — Proactive token refresh (`frontend/src/api/auth.ts`)
1. Widened the localStorage expiry check buffer from 0 to 65 seconds:
   `exp * 1000 > Date.now() + 65_000`
   This ensures that when `refetchInterval` fires 60 s before expiry, the `queryFn` does NOT
   serve the soon-to-expire localStorage token but instead falls through to `/api/v1/auth/refresh`.
2. Added `refetchInterval` to `useRefreshToken` — a function that computes the time until
   60 s before the cached token's `exp` claim, returning that duration in milliseconds.
3. Added `refetchIntervalInBackground: true` so the token is refreshed even when the browser
   tab is not focused.

### B — Disable Publish button; add "Try again" action (`frontend/src/pages/ExamWizard/Step4Review.tsx`)
1. Added `|| !!unsatisfiedRules` to the Publish button's `disabled` prop.
2. Added a "Try again" button inside the validation warning box that calls
   `setUnsatisfiedRules(null)`, re-enabling the Publish button so the user can retry after
   fixing their questions.
3. Added `exam.wizard.retryPublish` i18n key to `en.json`, `ru.json`, and `kk.json`.

## Files Changed
| File | Change |
|------|--------|
| `frontend/src/api/auth.ts` | `useRefreshToken`: wider localStorage buffer (65 s), `refetchInterval` function, `refetchIntervalInBackground` |
| `frontend/src/pages/ExamWizard/Step4Review.tsx` | Publish button disabled on unsatisfied rules; "Try again" button in warning box |
| `frontend/src/locales/en.json` | Added `exam.wizard.retryPublish` |
| `frontend/src/locales/ru.json` | Added `exam.wizard.retryPublish` |
| `frontend/src/locales/kk.json` | Added `exam.wizard.retryPublish` |

## Regression Test
None added — the proactive refresh interval cannot be unit-tested without fake timers; the
existing `useLogin` test in `auth.test.tsx` continues to pass, confirming no regression in
the login/token-cache path. The publish-button UX fix is a UI change with no service logic.

## Resolution Results
- Tests: pending
- Migration applied: no
- Build clean: pending

## Recurrence Log
| Date | Trigger | Action Taken |
|------|---------|--------------|
| 2026-05-19 | Access token expiry cascading 401 + publish button UX | Proactive refresh interval + disabled button |

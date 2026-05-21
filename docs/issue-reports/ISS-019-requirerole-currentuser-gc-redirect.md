---
id: ISS-019
title: RequireRole redirects to /login after ~5 min — currentUser GC'd from React Query cache
status: resolved
severity: high
layer: frontend
module: auth
tags: [RequireRole, currentUser, gcTime, getQueryData, setQueryData, GarbageCollection, 401, Navigate]
created: 2026-05-21
resolved: 2026-05-21
recurrence_count: 1
related_issues: [ISS-015]
regression_test: frontend/src/components/RequireRole.test.tsx
---

## Symptom
Users are unexpectedly redirected to the login page while navigating through the app after ~5
minutes of use. The issue is intermittent, happens from different pages, and does not correlate
with explicit token expiry (15-minute TTL).

## Root Cause

`['auth', 'currentUser']` is written into the React Query cache exclusively via
`qc.setQueryData()` — in `useLogin.onSuccess` and inside `useRefreshToken.queryFn`. No `useQuery`
hook ever subscribes to this key, so there is no active observer.

TanStack Query v5 starts the garbage-collection (`gcTime`) countdown the moment a cache entry has
no active observers. With the default `gcTime: 5 minutes`, the `['auth', 'currentUser']` entry is
silently removed from the cache ~5 minutes after login (or after the last token refresh, which
happens at ~14 minutes after login — well past the first GC window).

`RequireRole` and `RequireSuperAdmin` read this entry synchronously via
`qc.getQueryData(['auth', 'currentUser'])`. After the GC fires:

```
const user = qc.getQueryData<CurrentUser>(['auth', 'currentUser'])  // → undefined
if (!user || !roles.includes(user.role)) return <Navigate to="/login" replace />
```

`!user` is `true` → every navigation that touches a `RequireRole`-wrapped route redirects to
`/login`, even though the access token is valid and the user is legitimately authenticated.

The access token (`['auth', 'accessToken']`) is NOT affected because `useRefreshToken` calls
`useQuery({ queryKey: ['auth', 'accessToken'], ... })`, which registers a permanent observer that
prevents GC.

## Fix Applied

**`RequireRole.tsx` and `RequireSuperAdmin.tsx`**: Remove the dependency on
`['auth', 'currentUser']`. Instead, decode the `role` claim directly from the access token (JWT),
which is always available in the cache as long as the user is authenticated (kept alive by
`useRefreshToken`'s `useQuery` observer).

Added a local `jwtRole(token)` helper that base64-decodes the JWT payload and returns the `role`
claim. If decoding fails (malformed token), returns `undefined`, which fails the role check and
redirects to `/login`.

## Files Changed
| File | Change |
|------|--------|
| `frontend/src/components/RequireRole.tsx` | Remove `['auth', 'currentUser']` read; decode role from JWT access token |
| `frontend/src/components/RequireSuperAdmin.tsx` | Remove `['auth', 'currentUser']` read; decode role from JWT access token |

## Regression Test
`frontend/src/components/RequireRole.test.tsx` — added test that simulates the GC scenario:
token is present but `['auth', 'currentUser']` is absent. Asserts the route renders (no redirect).

## Resolution Results
- Tests: 206 frontend passed, all backend passed
- Migration applied: no
- Build clean: yes

## Recurrence Log
| Date | Trigger | Action Taken |
|------|---------|--------------|
| 2026-05-21 | User reports intermittent login redirect after a few minutes of session | Root cause identified: currentUser GC; role now decoded from JWT |

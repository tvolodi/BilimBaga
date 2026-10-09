---
id: ISS-249
title: Frontend does not handle 401 TOKEN_REVOKED
status: resolved
severity: medium
layer: frontend
module: auth
tags: [TOKEN_REVOKED, apiFetch, refresh, sessionRevoked]
created: 2026-10-09
resolved: 2026-10-09
recurrence_count: 1
related_issues: [ISS-240, ISS-171]
regression_test: frontend/src/api/apiFetch.test.ts
---

## Symptom
After PR #246 the backend answers 401 `TOKEN_REVOKED` when the role/department/status in the JWT no longer matches the DB. The SPA treated it as a generic error: the user stayed on a broken page with a stale token.

## Root Cause
`apiFetch` had no 401 handling; only `downloadFile` refreshed on 401. Nothing ended the session on a revoked token.

## Fix Applied
- `frontend/src/lib/sessionRevoked.ts`: `TOKEN_REVOKED`, `SESSION_REVOKED_KEY`, `endRevokedSession`, single-flight `refreshAccessTokenOnce`.
- `apiFetch`: on 401 TOKEN_REVOKED refresh once; if a different fresh token is obtained, retry the request once; otherwise (refresh failed, same token, or retry revoked again) clear token/user and raise the notice flag. RequireAuth then redirects to /login. No loops (one refresh, one retry).
- `LoginPage` shows `auth.login.sessionRevoked` (kk/ru/en) when the flag is set; cleared on successful login.

## Files Changed
| File | Change |
|------|--------|
| frontend/src/lib/sessionRevoked.ts | new helpers |
| frontend/src/api/apiFetch.ts | refresh+retry-once / logout |
| frontend/src/pages/auth/LoginPage.tsx | localized notice |
| frontend/src/locales/{kk,ru,en}.json | `auth.login.sessionRevoked` |
| frontend/src/api/apiFetch.test.ts, frontend/src/pages/auth/LoginPage.test.tsx | tests |

## Regression Test
apiFetch.test.ts (refresh+retry, refresh failure, same token, no loop, other codes untouched); LoginPage.test.tsx (notice shown / absent).

## Resolution Results
- Tests: 632 passed, 0 failed (88 files); tsc, eslint, check:i18n clean
- Migration applied: no
- Build clean: yes

## Recurrence Log
| Date | Trigger | Action Taken |
|------|---------|-------------- |

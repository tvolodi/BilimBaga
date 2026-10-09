---
id: ISS-171
title: Self change-password does not revoke older access tokens (only admin/forgot reset does)
status: resolved
severity: medium
layer: backend
module: auth
tags: [change-password, password_changed_at, TOKEN_REVOKED, tokenPredatesPasswordChange, refresh_tokens]
created: 2026-10-09
resolved: 2026-10-09
recurrence_count: 1
related_issues: [ISS-105, ISS-160]
regression_test: backend/internal/auth/self_change_revoke_test.go
---

## Symptom
After `POST /api/v1/auth/change-password`, access tokens issued before the change (other devices,
a stolen token) stayed valid until expiry and the user's other refresh tokens stayed usable.
Reset-password and admin reset already revoked (ISS-105 via `users.password_changed_at`).

## Root Cause
`auth.pgRepository.UpdatePassword` did not stamp `password_changed_at` nor revoke refresh tokens,
deliberately, so the caller's own token (and the ISS-160 forced-change flow) kept working.

## Fix Applied
Design: stamp + revoke + issue a replacement session in the same call.
- `Service.ChangePassword` now returns `(*ChangePasswordResponse, *http.Cookie, error)`.
  `ChangePasswordResponse{message, access_token, token_type, expires_in}` (`message` kept, so the
  response is backward compatible). Cookie is the new `refresh_token` (same attributes as login).
- `UpdatePassword(ctx, userID, hash, changedAt)` runs in one transaction: sets the hash, clears
  `force_password_change`, sets `password_changed_at = $3`, and revokes all of the user's refresh
  tokens (same statement as `CompleteReset`). The caller's NEW refresh token is created afterwards, so it
  survives and every other session's refresh token is revoked.
- The stamp is the application clock truncated to the second (`s.now().Truncate(time.Second)`), not
  the DB `now()`, and the new access token is issued with exactly that `iat` (`issueAccessTokenAt`).
  This removes app/DB clock skew from the equation: the new token always satisfies the epoch check.
- Handler sets the cookie, returns the response, still calls the cache-invalidate hook, so the
  `AccountStateCache` re-reads the new epoch immediately in this process (other processes: <= 10 s TTL).
- Frontend `useChangePassword` stores `data.access_token` in `['auth','accessToken']` (and refreshes the
  E2E-seeded localStorage token when present); `ChangePasswordPage` already reads the role from that
  cache entry, so it continues to `/portal` or `/admin` and clears the forced flag as before.

### Boundary semantics (documented in code)
`tokenPredatesPasswordChange`: reject iff `iat < floor(password_changed_at)` (unix seconds).
- New token: `iat == floor(stamp)`, accepted (inclusive), even in the same second as the stamp.
- Older token with `iat < floor(stamp)`: 401 `TOKEN_REVOKED`.
- Known caveat unchanged: an old token minted in the SAME second as the stamp is also accepted. Closing it
  would need sub-second iat or a token generation counter (out of scope; window is 1 s).
- Wrong current password / weak password return before any write: no stamp, no revocation.

## Files Changed
| File | Change |
|------|--------|
| backend/internal/auth/service.go | ChangePassword returns session; issueAccessTokenAt; boundary doc |
| backend/internal/auth/repository.go | UpdatePassword(+changedAt) stamps epoch and revokes refresh tokens in a tx |
| backend/internal/auth/handler.go, model.go | cookie + response body; ChangePasswordResponse |
| backend/internal/auth/*_test.go | mock signature updates; new self_change_revoke_test.go |
| frontend/src/api/auth.ts | useChangePassword returns + stores new token |
| frontend/src/api/auth.test.tsx, frontend/src/pages/auth/ChangePasswordPage.test.tsx | new tests |
| scripts/lib/e2e-auth.ts (+test), scripts/seed-test-env.ts, frontend/e2e/global-setup.ts | use token from the change response; probe cached token |
| docs/requirements/api-conventions.md | note |

## E2E / script audit (every caller of change-password)
| Caller | Token used afterwards? | Action |
|--------|-----------------------|--------|
| scripts/lib/e2e-auth.ts `loginClearingForceChange` | yes (returned) | returns `access_token` from the change response; null if absent; test updated |
| frontend/e2e/global-setup.ts (page.evaluate) | yes (saved to storage/token.txt) | uses change response token; refresh cookie replaced by the change response. Also `tokenStillAccepted` probe (GET /users/me) before reusing cached `.auth/token.txt`, else fresh login |
| frontend/e2e/fixtures/seed.ts (2 places, employee) | no, only admin token used afterwards; employee logs in later | none needed |
| scripts/seed-test-env.ts `changeForcedAdminPassword` | yes (returned) | returns token from response |
| scripts/seed-test-env.ts admin normalisations (2 places) | re-login follows | none needed |
| scripts/seed-test-env.ts employee (1 place) | no | none needed |
| frontend/e2e/dept-admin-scoping.spec.ts | re-login follows | none needed |
| frontend/e2e/full-walkthrough.spec.ts `handleForcePasswordChange` | UI flow, helper unused | none; hook stores the new token |
| frontend/e2e/auth.spec.ts | only a URL regexp | none |

## Regression Test
`backend/internal/auth/self_change_revoke_test.go`: old token 401 TOKEN_REVOKED and new token 200 right
after; same-second boundary (stamp truncation, iat equality, iat-1 revoked, iat accepted); refresh tokens
revoked before the new one is created; forced-change flow with returned token; wrong current / weak
password do not stamp; handler returns token + cookie + hook. Frontend: `auth.test.tsx` (token stored,
failure keeps old), `ChangePasswordPage.test.tsx` (store, clear flag, redirect). `e2e-auth.test.ts`.

## Resolution Results
- Backend: go vet clean, staticcheck 0 findings, `go test -p 1 ./...` all ok.
- Frontend: tsc clean, check:i18n ok, vitest 78 files / 538 tests pass (includes scripts/lib tests).
- Migration applied: no (no new migration; `password_changed_at` is migration 032).
- PR label: needs-live-db (the SQL `updatePasswordSQL` and the refresh-token revoke run in a tx; the same
  statements as the reset path, but not executed against a real Postgres here).

## UAT must re-run
1. Two sessions (A, B) of one user; change password in A: A continues (new token, no logout), B's next
   API call gets 401 TOKEN_REVOKED and its refresh fails 401.
2. Forced-change user (new user / seeded admin): login, change password on /change-password, lands on home with no 403/401.
3. Wrong current password: nothing changes, session intact.
4. `make e2e` global-setup on a fresh stack (forced admin change) and on a re-run with cached token.

## Recurrence Log
| Date | Trigger | Action Taken |
|------|---------|-------------|

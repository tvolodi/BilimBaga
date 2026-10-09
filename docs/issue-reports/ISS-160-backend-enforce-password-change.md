---
id: ISS-160
title: Backend does not enforce users.force_password_change (only the SPA LoginPage acts on it)
status: resolved
severity: high
layer: backend
module: auth
tags: [force_password_change, PASSWORD_CHANGE_REQUIRED, Authenticate, AccountStateCache, password_epoch]
created: 2026-10-09
resolved: 2026-10-09
recurrence_count: 1
related_issues: [ISS-150, ISS-152, ISS-105]
regression_test: backend/internal/auth/force_password_change_test.go
---

## Symptom
`POST /auth/login` returns `user.force_password_change: true` and only the SPA `LoginPage` redirects to
`/change-password`. Any API client (or a user typing a URL) holding that token can call every
authenticated route with the default / admin-issued temporary password.

## Root Cause
Enforcement was UI-only. `auth.Authenticate` validated signature, expiry and the password epoch
(ISS-105) but never looked at `users.force_password_change`.

## Fix Applied (design decision)
**Enforce in `auth.Authenticate`, driven by the per-user lookup that already exists for the password
epoch, not by a JWT claim.**

- `password_epoch.go`: the epoch lookup became `AccountStateCache` (`AccountState{PasswordChangedAt,
  ForcePasswordChange}`), one query `SELECT password_changed_at, force_password_change FROM users WHERE id=$1`,
  same 10 s per-user TTL, errors never cached, plus `Invalidate(userID)`.
- `middleware.go`: `WithAccountState(lookup)` replaces `WithPasswordEpoch`. After the epoch check, if the flag
  is set and the request is not on the allowlist -> `403 PASSWORD_CHANGE_REQUIRED` in the standard envelope.
  Lookup error -> 500 INTERNAL_ERROR (fail closed); unknown user -> 401 INVALID_TOKEN (unchanged).
- `handler.go` / `recovery_handler.go`: `Handler.SetPasswordChangedHook(fn)`, called after a successful
  change-password and reset-password. `router.New` wires it to `AccountStateCache.Invalidate`, so the user is
  not blocked for up to 10 s after changing. Failures do not invalidate.
- `router.go`: one `AccountStateCache` per router, shared by the middleware and the hook. Nil `db` (route tests)
  keeps signature/expiry-only auth.

Why not a JWT claim: no token reissue, no coupling with refresh rotation, no stale-claim window (a claim stays
true until the token expires, 15 min, even after the change; clearing it needs a reissue the SPA does not
do after change-password). The cost is the same single indexed PK lookup per request that ISS-105 already pays,
now returning one more column, so no extra query and no migration.

Staleness bounds: change/reset through this process is immediate (invalidate). An admin reset of another
user (`users.UpdatePassword`, sets the flag and `password_changed_at`) is seen within 10 s, same bound as the
epoch revocation it already relied on; the old token is revoked by the epoch anyway. Multi-replica: another
replica's cache may be stale for up to 10 s after a change (user sees 403 once, the SPA redirects back to
`/change-password` which then fails to re-trigger since the flag is cleared; retry works after the TTL).

### Route allowlist (flag set)
| Route | Why | Passes through Authenticate? |
|-------|-----|------|
| `POST /api/v1/auth/change-password` | the way out | yes (allowlisted, any method on that exact path) |
| `GET /api/v1/users/me` | SPA profile read; only GET | yes (allowlisted) |
| `POST /api/v1/auth/login` | public | no |
| `POST /api/v1/auth/refresh` | public, cookie based (not in the protected group) | no |
| `POST /api/v1/auth/logout` | public, cookie based | no |
| forgot/reset-password | public | no |
| `/auth/me` | does not exist; `/users/me` is the equivalent | n/a |
| everything else in the protected group | blocked with 403 PASSWORD_CHANGE_REQUIRED | yes |

Matching is exact on `r.URL.Path` (no prefix/suffix), so `/users/me/`, `/auth/change-password/x`, and non-GET
`/users/me` are blocked.

### Frontend
- `lib/passwordChangeRequired.ts`: `createAppQueryClient()` installs QueryCache/MutationCache `onError`:
  any error with `code === 'PASSWORD_CHANGE_REQUIRED'` raises query flag `['auth','passwordChangeRequired']`
  (covers every API module, including those with their own raw `fetch`, since they all run in React Query).
  `downloadFile` (not React Query) raises it directly on 403.
- `components/PasswordChangeGuard.tsx` (mounted in `AuthedRoutes`): flag or `currentUser.force_password_change`
  -> `navigate('/change-password', {replace})`, so a flagged user cannot browse the app (before, a reload lost the
  flag: refresh/E2E-seed builds `currentUser` from JWT with `force_password_change:false`; now the first 403 restores it).
- Cleared on successful change (`ChangePasswordPage`), login (response is authoritative) and logout.
- i18n `auth.changePassword.required` (en/ru/kk) shown on the change-password page.

### Who gets force_password_change = true
| Source | Flag | Effect now |
|--------|------|------------|
| `users.Create` (admin UI/API, `INSERT ... true`) | true | new user must change before any other call |
| `POST /users/import` (CSV) | goes through the same `users.Create` (the only `INSERT INTO users`), so true | imported users must change before any other call |
| Admin reset (`users.UpdatePassword`) | true | same |
| Seeded admin on default hash (migration 033 / `BootstrapAdmin`) | true unless `BOOTSTRAP_ADMIN_PASSWORD` | must change first |
| Self change (`auth.UpdatePassword`) / forgot-password reset | cleared | cache invalidated |

### E2E / scripts audit
| File | Login of | Before | Now |
|------|----------|--------|-----|
| `frontend/e2e/global-setup.ts` | admin | login with E2E_ADMIN_PASS or Admin1234! (flagged on fresh stack) | tries `adminPasswordCandidates` (E2E_ADMIN_PASS, E2E_ADMIN_NEW_PASS/E2eAdmin2024!, Admin2024!, Admin1234!) and, if flagged, POST /auth/change-password with the same token to `E2E_ADMIN_NEW_PASS` (default E2eAdmin2024!) before saving the token/storage state |
| `frontend/e2e/fixtures/seed.ts` getSeedData | admin re-login | E2E_ADMIN_PASS or Admin1234! | `loginClearingForceChange` with the same candidates |
| `seed.ts` createEmployee / setEmployeeKnownPassword | employee (temp pw) | already logs in and changes to `Employee1234!` | unchanged, works (change-password allowlisted) |
| `seed.ts` createTestUser | new user | returns temp password, no token | unchanged; documented that the token is unusable until changed. Consumers (`account-recovery`, `user-management`) only POST login and check status, or drive the admin UI: no authenticated call as that user |
| `frontend/e2e/account-recovery.spec.ts` | test user | login status checks only; post-reset `force_password_change=false` assertion | unchanged, still valid (reset clears flag) |
| `frontend/e2e/loyalty-narrative.spec.ts` | seed employee (`Employee1234!`) | flag already false | unchanged |
| `frontend/e2e/full-walkthrough.spec.ts` | admin via UI | handles /change-password UI with E2E_ADMIN_NEW_PASS | unchanged, still consistent |
| `frontend/e2e/auth.spec.ts` | admin via UI | accepts /admin or /change-password | unchanged |
| `scripts/seed-test-env.ts` ensureAdminToken | admin | E2E_ADMIN_PASS branch returned a possibly flagged token | `changeForcedAdminPassword` (re-sets the same password) for the E2E_ADMIN_PASS and known-password branches; other branches already changed passwords |
| `scripts/seed-test-env.ts` ensureUser | created users | login with temp + change-password | unchanged |
| `scripts/lib/e2e-auth.ts` (new) | shared helper + unit tests | n/a | `loginClearingForceChange`, `adminPasswordCandidates` |

Not run: no live stack (resource limits). The changes are unit tested but not exercised against Postgres.

## Files Changed
| File | Change |
|------|--------|
| backend/internal/auth/password_epoch.go | AccountState + AccountStateCache (+Invalidate), combined query |
| backend/internal/auth/middleware.go | WithAccountState, allowlist, 403 PASSWORD_CHANGE_REQUIRED |
| backend/internal/auth/handler.go, recovery_handler.go | SetPasswordChangedHook, invoked after change/reset |
| backend/internal/router/router.go | shared cache, hook wiring |
| backend/internal/auth/hardening_test.go | adapted to AccountState |
| backend/internal/auth/force_password_change_test.go | new tests |
| frontend/src/lib/passwordChangeRequired.ts(+test), components/PasswordChangeGuard.tsx(+test) | new |
| frontend/src/App.tsx, api/auth.ts, api/download.ts(+test), pages/auth/ChangePasswordPage.tsx, locales/{en,ru,kk}.json | guard, flag lifecycle, i18n |
| frontend/e2e/global-setup.ts, frontend/e2e/fixtures/seed.ts, scripts/seed-test-env.ts, scripts/lib/e2e-auth.ts(+test) | e2e/seed audit |

## Regression Test
`force_password_change_test.go`: blocked routes return 403 + code; allowlist passes; flag false passes;
unknown user 401; lookup error 500 on every route (fail closed, including allowlisted); cache invalidation
(stale within TTL, fresh after `Invalidate`); errors not cached; handler change-password clears the flag then the
same token reaches a protected route immediately (cache invalidated); reset-password invalidates; failed
change does not. Frontend: guard redirect tests, QueryClient error handler tests (query, mutation, apiFetch 403
envelope, other errors ignored), download 403. `scripts/lib/e2e-auth.test.ts`.

## Resolution Results
- Go: `go vet`, `staticcheck` 0 findings, `go test -p 1 ./...` all pass.
- Frontend: `tsc --noEmit` clean, `check:i18n` ok (793 keys), vitest 74+1 files pass.
- Migration applied: no (none needed). SQL changed (`accountStateSQL`): label `needs-live-db`.
- Build clean: yes.

## What UAT must re-run
1. Fresh stack (migration 033 applied, no BOOTSTRAP_ADMIN_PASSWORD): login as admin with Admin1234!, confirm
   every protected API call except change-password and GET /users/me returns 403 PASSWORD_CHANGE_REQUIRED, UI redirects to
   /change-password with the notice, after change the same session works immediately (no 10 s wait) and a reload stays in the app.
2. Admin creates a user: that user's API token is blocked until the password is changed; admin reset-password re-blocks within 10 s.
3. Full live e2e suite (`npm run test:e2e:live`) on a fresh DB and on a re-run (admin on E2eAdmin2024!), plus `scripts/seed-test-env.ts`
   with and without `E2E_ADMIN_PASS`.
4. Refresh-cookie path: `/auth/refresh` and `/auth/logout` still work for a flagged user.
5. Multi-replica deployments: up to 10 s staleness after a change on another replica.

## Recurrence Log
| Date | Trigger | Action Taken |
|------|---------|-------------- |

## Changes after UAT (PR #180 @e936541: backend PASS 24/24)
UAT found two gaps; fixed on the same branch (frontend and e2e only, backend untouched).
1. UI redirect on a full page load of a non-allowlisted route (/portal redirected after ~7.8 s because React Query
   retried the 403; /portal/results never redirected):
   - Root causes: (a) default query retry (3, with backoff) re-sent the 403; (b) many `src/api/*.ts` helpers threw
     `new Error(body.error.message)` and dropped `code`, so the global onError never recognised the error
     (/portal/results was one of them, via sessions.ts).
   - Fix: `createAppQueryClient` sets default `retry` for queries (3) and mutations (0) to never retry
     PASSWORD_CHANGE_REQUIRED; new `api/errors.ts` `errorWithCode` used by analytics, audit, dashboard, employees,
     grading, questions, reports (also parses the body before the `!res.ok` check), sessions, useTenantConfig, users;
     `useRefreshToken` now calls `GET /users/me` (allowlisted; returns `force_password_change`) right after obtaining
     a token, so a hard reload redirects before any data query fails (best effort; the 403 handler remains the net).
   - Tests (`src/lib/passwordChangeRedirect.test.tsx`): retry policy, /portal/results redirect on the first 403 with exactly
     one request (within waitFor's 1 s, which retries would miss), hard-reload bootstrap redirect with zero data calls.
2. `auth.spec.ts` form login hard-coded `Admin1234!` and broke without `E2E_ADMIN_PASS` after global-setup changed the
   password. global-setup now records the effective admin password in `frontend/.auth/admin-pass.txt`;
   `fixtures/admin-pass.ts#currentAdminPassword()` (E2E_ADMIN_PASS, then the file, then the default) is used by
   `auth.spec.ts` and `full-walkthrough.spec.ts` (the only specs that hard-coded the default; grep verified).
   Caveat: when global-setup reuses a cached token the file from the earlier run is used; delete `.auth/` after resetting the DB.
- Checks: tsc, check:i18n, full vitest (75 files, 510 tests, `--maxWorkers=1`) green. Not re-run live.

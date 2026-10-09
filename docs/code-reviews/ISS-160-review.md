# Code Review: ISS-160 (backend enforcement of force_password_change)

Reviewer: Code Reviewer subagent. Static review only (no tests run: host memory constraint).
Scope: uncommitted working tree on `swarm/160-enforce-pw-change`.

## Result: PASS

Zero Critical, zero High findings. Two Medium and several Low/Info items below.

## Allowlist and bypass analysis

- Matching uses `r.URL.Path` against an exact-string map (`/api/v1/auth/change-password`, `/api/v1/users/me`). No prefix or suffix matching, so `/users/me/`, `/auth/change-password/x`, `/users/me/..` all fall to 403 (covered by tests).
- Method: `/users/me` is GET only; change-password allows any method, but the router registers POST only (other methods give 405 after auth, harmless).
- chi routing vs `URL.Path`: chi routes on `RawPath` when set, else `Path`. `Path` is the decoded form of the same string, so an encoded variant (`users%2Fme`) cannot route to a different handler than the decoded allowlisted path would. Encoded slash gives a path that does not match the real `/users/me` route (404), not a bypass. The middleware runs inside `r.Route("/api/v1")` and `r.URL.Path` is not rewritten by chi Route/Group (only RoutePath in the chi context), so the full-path keys are correct. No bypass found.
- Case sensitivity: chi is case-sensitive and the map is exact, consistent.
- Only one `auth.Authenticate` call site (router.go protected group); `/auth/login|refresh|logout|forgot|reset` are public by design. No protected route is registered outside that group.
- Fail-closed: lookup error gives 500, `ErrNotFound` gives 401, missing `iat` gives 401, errors are not cached. Allowlisted routes also fail closed on lookup failure (tested).
- Router wiring with nil db: `accountState` stays a typed nil `*AccountStateCache`; `authenticate()` compares the pointer, so it correctly falls back to plain `Authenticate` (route tests only). `authHandler != nil` is guarded. Production always passes db (main.go), so enforcement is active. Note this is silently fail-open when db is nil; acceptable only because it is a test-only path.

## Findings

### Medium

1. `backend/internal/auth/password_epoch.go` Lookup/Invalidate: race between an in-flight `Lookup` miss and `Invalidate`. Sequence: request A reads DB (force=true), user's change-password commits and calls `Invalidate`, then A stores the stale `force=true` entry. The user then gets 403 PASSWORD_CHANGE_REQUIRED for up to 10 s after a successful change; the frontend guard re-raises the flag from that 403 and bounces the user back to /change-password. Same symptom on any other API instance (invalidation is process-local). Fail-safe direction (blocks, not leaks), so not High. Suggested fix: per-user generation counter (or global invalidation epoch) captured before fetch and compared before store; optionally treat a 403 PASSWORD_CHANGE_REQUIRED on the frontend as authoritative only if `currentUser.force_password_change` is not already false.
2. `backend/internal/users/repository.go:184` (admin `UpdatePassword`, sets `force_password_change = true`) and `auth/bootstrap.go` `RequireAdminPasswordChange` set the flag without cache invalidation. Admin reset therefore relies on the 10 s TTL: if the target user made a request within 10 s before the reset and logs in with the temp password within that window, the stale `force=false` entry lets them through for the remainder of the TTL. Tiny window, practically hard to hit, but this is the one "sets flag without invalidating" path. Suggested fix: invoke the same invalidation hook from `users.Handler/Service.ResetPassword` (or share the cache with the users package via a small interface). Bootstrap runs before the server serves, so it is fine.

### Low / Info

3. `scripts/lib/e2e-auth.ts`, `frontend/e2e/global-setup.ts`: lockout risk. Account locks after 5 failures (service.go AC-4). Candidates list is up to 4 passwords tried in order (`E2E_ADMIN_PASS`, `E2eAdmin2024!`, `Admin2024!`, `Admin1234!`); a successful login resets the counter, so a normal run costs at most 3 failures. But if no candidate is valid, one run burns 4 attempts and the next run locks the admin. Also `seed-test-env.ts` probes with other passwords in the same run. Suggest capping to 3 candidates per run or stopping on a `ACCOUNT_LOCKED` response. Also the auth rate limiter is 10 req/min per IP; worst case 4 logins + change is within it.
4. `page.evaluate` serialisation in global-setup.ts: the callback is self-contained (uses only its argument and globals; `adminPasswordCandidates` is called outside, in the Node side argument object). Correct. `any` types inside the evaluate closure are tolerable in test code.
5. `seed-test-env.ts changeForcedAdminPassword` changes password to the same value; verified the backend service does not reject reuse (only complexity and current-password checks), so it works. The default `Admin1234!` must satisfy complexity (it does: upper, lower, digit, 8+).
6. Frontend guard: no redirect loop found. The guard only navigates when `pathname !== '/change-password'`; the change-password page uses an allowlisted route. Flag is cleared on successful change, login and logout; `useLogin` clears the flag using the login response as source of truth. An in-flight request that was rejected before the change but resolves after could re-raise the flag (cosmetic, related to Medium 1).
7. `PasswordChangeGuard` is mounted once in App.tsx inside the router (`<>...</>` wrapper); queryClient via `createAppQueryClient()` covers all query/mutation errors globally. `downloadFile` (raw fetch outside the query client) is separately handled and tested. Other raw-`fetch` callers, if any, are not covered but the backend still blocks them.
8. i18n: `auth.changePassword.required` added to en, ru, kk. All three present, no hardcoded strings in the new component (guard renders nothing).
9. Handler is thin: hook is invoked after the service succeeds and before the audit write; failure path does not invalidate (tested). Hook is optional (nil-safe). Errors wrapped with context in the SQL fetch. Audit unchanged. `ChangePassword` does not bump `password_changed_at`, so the token used for the change remains valid (intended; the e2e helpers rely on this).
10. Hardening of the allowlist const: `ForcePasswordChangeAllowedPaths` is an exported mutable map; any package could mutate it at runtime. Consider unexported or a switch statement. Low.

## Acceptance criteria (issue #160)

- Every authenticated route except POST /auth/change-password and GET /users/me answers 403 PASSWORD_CHANGE_REQUIRED while flag is true: covered (middleware + tests for blocked, allowlisted, flag false).
- Fail-closed on lookup failure: covered.
- Change/reset invalidates cache so the user is not blocked afterwards: covered for the single-process happy path (see Medium 1 for the race / multi-instance caveat).
- SPA redirects to change-password on the 403 and shows a localised explanation: covered.
- E2E/seed flows adapted to the new enforcement: covered (see Low 3 for lockout hygiene).

## Summary

The enforcement is sound and fail-closed with no allowlist bypass found; address the cache race (Medium 1) and add invalidation to the admin reset path (Medium 2) as follow-ups, they do not block merge.

## Follow-up applied by the author after review
- Medium 1 (invalidate vs in-flight fetch race): `AccountStateCache` now has a generation counter; a fetch that overlapped an `Invalidate` is not stored (test `TestAccountStateCache_InvalidateDuringFetchDoesNotStoreStale`). Cross-replica staleness (<= 10 s) remains and is documented.
- Low (lockout): `adminPasswordCandidates` capped at 3 attempts.
- Low (exported map): allowlist is now unexported.
- Medium 2 (admin reset of another user does not invalidate): not changed; bounded by the 10 s TTL, and the old token is revoked by the epoch within the same window. Left as documented.

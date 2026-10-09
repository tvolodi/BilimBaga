# Code Review: ISS-248 AccountStateCache invalidation (run swarm-248)

**Verdict: PASS** (no blocking findings)

## Scope
users/handler.go, users/handler_test.go, router/router.go, auth/stale_claims_test.go, docs/issue-reports/ISS-248-accountstate-cache.md.

## Checks
- Correctness: hook fires only after the service call succeeds in UpdateUser, DeactivateUser, ResetPassword, UnlockUser. It is nil-safe. It runs before the audit write, so an audit failure cannot skip invalidation.
- Wiring: router wires `usersHandler.SetUserChangedHook(accountState.Invalidate)` inside the `db != nil` block, with a nil guard on the handler. This mirrors `SetPasswordChangedHook`.
- Race safety: `Invalidate` bumps `gen`, so a fetch that raced with the mutation is not re-cached stale.
- Architecture: no new users->auth import, because a func hook is used. No circular imports. Handler stays thin.
- Tests: handler tests cover the hook per mutation, no call on failure, and nil hook. The cache tests cover the spurious revoke after a role change, the old token revoked after demotion, and deactivation. `go build ./...` is clean, and `go test` for users, auth and router passes.

## Other mutation paths considered
- CreateUser and ImportUsers: insert new IDs only, so no cache entry can exist. No hook needed.
- Role delete: blocked by FK when the role is in use (InUseError), so no user can hold a deleted role.
- Role update: changes description and permissions only. Cached state is role name, department, status and password epoch, so it is unaffected. Permissions are evaluated by the RBAC cache, not AccountState.
- auth repository lockouts (failed_attempts, locked_until): not part of AccountState. Auth handler password paths are already hooked. Bootstrap admin password runs at startup, before any cache entry exists.

## Minor / non-blocking
- The cache is per-process. A multi-instance deployment would still wait out the TTL (10s) on other instances. This is acceptable and bounded, and worth noting in the issue report.
- The ISS report claims `go test -p 2 ./...` is fully green. I re-ran only the affected packages.

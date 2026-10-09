# ISS-248 AccountState cache invalidation

- Source: GitHub #248 (follow-up of #240 / PR #246). Run id: swarm-248.
- Root cause: `auth.AccountStateCache` (10s TTL) is invalidated only on password change/reset via
  `auth` handlers. Admin mutations in the `users` domain (update, deactivate, reset-password, unlock)
  left the cached role/department/status in place, so a freshly issued token carrying a new role was
  rejected with TOKEN_REVOKED (stale entry), and a deactivated user's token was allowed for up to 10s.
- Fix: `users.Handler.SetUserChangedHook(fn)`; handler calls it with the target user id after each
  successful UpdateUser / DeactivateUser / ResetPassword / UnlockUser. `router.New` wires it to
  `accountState.Invalidate` (same pattern as `authHandler.SetPasswordChangedHook`). No new import
  between users and auth beyond what exists; no migrations.
- Tests: users/handler_test.go (hook invoked per mutation, not on failure, nil hook safe);
  auth/stale_claims_test.go (role change spurious revoke resolved by Invalidate; old token revoked
  after demotion; deactivation takes effect immediately after Invalidate).
- Verification: `go vet ./...` clean; `go test -p 2 ./...` all packages ok.
- Note: cache is per-process; multi-instance deployments still wait out the 10s TTL on other instances (bounded, accepted).

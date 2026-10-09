# Code Review: ISS-171 (self change-password revokes tokens)

Branch: swarm/171-self-change-revoke. Read-only review of the staged diff; no tests or builds run.

Result: PASS (0 Critical, 0 High)

## Findings

- [Medium] backend/internal/auth/service.go (ChangePassword, after UpdatePassword): the stamp and refresh revocation are committed before `issueAccessTokenAt` and `issueRefreshCookie`. If either fails (for example a refresh-token INSERT error), the API returns 500 but the password has changed and the caller's current access token (iat < stamp) is already dead. Result: a spurious error, then a forced re-login with the new password. This is not a permanent lockout, because the new password works, but the UI shows a failure for a change that succeeded. Suggestion: insert the new refresh token in the same transaction as the revoke, or document the behaviour. Signing the access token is local and effectively cannot fail, so the real risk is the refresh insert.
- [Medium] Multi-instance clock skew (service.go, `s.now().Truncate(time.Second)` stored as `password_changed_at`): the stamp now comes from the application clock, not the DB. If the instance handling the change runs ahead of the others by N seconds, tokens issued by other instances (logins, refreshes) in that N-second window have iat < stamp and are rejected with TOKEN_REVOKED. The caller's own token is unaffected, because its iat equals the stamp second. Typical skew is under 1s, so this is low probability. The admin reset path uses the DB `now()` and does not have this problem. Suggestion: note the NTP assumption in the report, or clamp the stamp to `min(app, db)` if this becomes an issue.
- [Low] Cache staleness across instances (password_epoch.go, `accountStateTTL = 10s`): `Invalidate` only clears the local process's entry. The mechanism is unchanged from ISS-105/160, but ISS-171 makes it more visible in two ways:
  - Another instance serves old access tokens for up to 10s after the change (an accepted window).
  - If the caller's next request lands on a different instance with a cached `ForcePasswordChange=true` entry, it gets a 403 PASSWORD_CHANGE_REQUIRED for up to 10s right after a forced change. This is pre-existing in ISS-160 behaviour; the frontend redirect flow may flicker. The invalidation also runs after commit, so a request that read before the commit can briefly repopulate the cache (tiny race, bounded by the TTL).
- [Low] Same-second survival of old tokens: documented in the `ChangePassword` comment and the report. A token issued in the same second as the stamp is not revoked (`iat < floor(stamp)` is strict). This is the accepted trade-off for the caller's new token being valid immediately. Boundary correctness itself is sound: `changedAt` is truncated, `issueAccessTokenAt(changedAt)` sets iat to the same second, so `iat >= floor(stamp)` always holds for the new token. A fractional-second stamp is covered by a test.
- [Low] Refresh-token reuse (AC-11, `Refresh`): another device or tab holding the pre-change refresh cookie gets a revoked-token response and triggers revoke-all, which also kills the caller's new refresh token. This is intended security behaviour. Revocation (in the tx) runs before the new refresh token is issued, so the order is correct and the new token is not revoked by the change itself.
- [Info] Response backward compatibility: `{message}` is retained and `access_token`, `token_type`, `expires_in` are additive. The handler sets the cookie only when non-nil. OK.
- [Info] Handler stays thin; SQL is in the repository; errors are wrapped; the audit write is unchanged; the stamp is UTC via `s.now()`.

## E2E and script audit (grep of every change-password caller)

Adapted to the new token: `frontend/e2e/global-setup.ts` (adds a stale-cache probe via /users/me and uses the returned token), `scripts/lib/e2e-auth.ts`, `scripts/seed-test-env.ts:changeForcedAdminPassword`, and `frontend/src/api/auth.ts` (stores the token in the query cache and updates the e2e localStorage key).

Unchanged callers, all verified safe because they discard the old token and re-login or never reuse it:
- `scripts/seed-test-env.ts` lines ~194, ~217 and ~320: re-login after the change (the user-creation path does not reuse `loginResult.token`).
- `frontend/e2e/fixtures/seed.ts` lines ~106 and ~149: the employee token is not reused.
- `frontend/e2e/dept-admin-scoping.spec.ts:77`: re-logs in and uses `again.data.access_token`.
- UI paths (`full-walkthrough.spec.ts` form flow, `ChangePasswordPage`): covered by the `onSuccess` token swap.

No missed callers found. Residual: a UI-driven admin password change in the walkthrough would invalidate `.auth/token.txt`; the new probe in global-setup handles this.

## Test quality

The backend tests cover:
- old token revoked and new token accepted immediately
- same-second boundary and fractional-second stamp
- refresh revocation followed by a new cookie
- the forced-change flow
- wrong or weak password does not stamp
- the handler response and cookie

The frontend tests cover the token being stored and the page flow. Gap: there is no repository-level test that the tx rolls back the stamp if the refresh revoke fails (it would need a DB), and no test for the issue-failure path in the first Medium finding. Neither is blocking.

## AC coverage
All behaviours in the stated change (stamp, tx revoke, fresh token with iat == stamp second, new cookie, frontend storage, script updates) are implemented.

Summary: the boundary logic is correct and every caller is audited; only Medium and Low robustness notes (a non-atomic token issue after commit, and clock-skew and cache-TTL windows), so PASS.

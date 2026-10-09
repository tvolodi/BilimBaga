# Code Review: ISS-240 (users authz hardening, FR-BB117 D-4)

Reviewer: Code Reviewer subagent. Scope: uncommitted changes in `backend/internal/auth` and `backend/internal/users`.
Checks run: `go vet ./internal/auth ./internal/users` clean; `go test ./internal/auth ./internal/users ./internal/router` pass (cached).

## Verdict: PASS

Zero Critical, zero High findings. The Medium and Low items below are follow-ups.

## What was verified

- **Strict subset (`permissionSubset`)**: the role's permissions must all be held by the caller, and the caller must hold at least one permission the role lacks. A missing `permsFor` or `canPerm` fails closed. An empty role permission set still passes the first loop, which matches the earlier behaviour.
- **No self role change or self deactivate**: the `isOrgWide` exemption is removed from `UpdateUser`. `DeactivateUser` adds `id == callerUserID` -> 403. Role comparison uses `sameID` (EqualFold), so case differences cannot bypass it.
- **404 for out-of-scope and unknown ids**: `GetUser`, `UpdateUser`, `DeactivateUser`, `ResetPassword` and `UnlockUser` return `ErrNotFound` when the target is out of scope. The handler maps `ErrNotFound` to 404 in every case. Deactivate checks scope (404) before self (403) before D-1, so existence is not leaked.
- **Malformed UUIDs**: `GetByID` and `GetRoleNameByID` return `ErrNotFound` for a bad id. `Create` and `Update` call `validateIDs`, which returns `ErrValidation` (wrapped with `%w`, so `errors.Is` holds) and is mapped to 422. Empty role id and nil department id are allowed through to the existing validation.
- **Stale-claims check**: the production lookup joins `roles` and always sets `Status`. Tokens are issued with `Role = user.RoleName` (roles.name via the same join) and `DepartmentID = *string`, which is omitted when NULL. The middleware reads a missing claim as `""`, and the lookup maps a NULL column to `""`, so a user with no department compares equal. Postgres returns UUIDs in lowercase, the same string as in the token, so the comparison is exact. The status CHECK only allows 'active' and 'inactive', so no valid status is wrongly rejected. Both login and `Refresh` go through the same `issueAccessToken` / `GetUserByID` join, so fresh tokens match the database.
- **Import**: `GetDepartmentIDByName` fetches two rows and returns `ErrAmbiguousName` on a duplicate name. `validateImportRow` resolves the department and role ids once and `commitImportRow` reuses them, which closes the check/use gap. The unused `callerUserID` and `ip` parameters were dropped from `commitImportRow`. Role assignment is still checked per row.
- **Security**: all SQL is parameterised. No new secrets or `os.Getenv` calls.

## Findings

- **[Medium]** `backend/internal/auth/middleware.go:109`, user experience and lockout window. The frontend (`frontend/src/api/auth.ts`) refreshes the access token only on a timer, 60 s before expiry. It has no handler for a 401 `TOKEN_REVOKED` response. After an admin changes a user's role or department, that user gets 401 on every API call until the timer fires, up to the full access TTL, or until they re-login. Nothing is permanently locked out, because the refresh endpoint issues a correct token. Suggestion: on 401 `TOKEN_REVOKED`, have the frontend call `/auth/refresh` once and retry. This can be a follow-up issue.
- **[Medium]** `backend/internal/auth/password_epoch.go` (cache) and `backend/internal/users/service.go`, transient false 401 after refresh. The cache TTL is 10 s and nothing calls `Invalidate` after `UpdateUser` or `DeactivateUser`. If the user's cache entry was fetched before the change and they refresh within 10 s, the new token carries the new role while the cache still holds the old one. The middleware then returns a spurious `TOKEN_REVOKED` for up to 10 s. This also applies across multiple API instances, where Invalidate does not reach other processes. Suggestion: call `state.Invalidate(id)` from the users service (via a small callback) after update, deactivate, reset and unlock, as the password-change path already does. Alternatively, treat a mismatch as stale only after a forced cache re-read.
- **[Low]** `backend/internal/auth/middleware.go:142`, `claimsStale` fails open when `Status == ""`. Production cannot hit this today (`users.status` is NOT NULL with a CHECK and the lookup always sets it). It remains a latent fail-open if a future lookup forgets the field. Suggestion: add `AccountState.HasIdentity bool` set by the production lookup, or make the test doubles set `Status: "active"` and drop the skip. Tests for the empty-Status case should be kept either way.
- **[Low]** `backend/internal/auth/service.go` (`Refresh`), pre-existing. An inactive user can still rotate a refresh token and receive an access token. The middleware now rejects it, but `Refresh` should reject inactive users itself and revoke their tokens.
- **[Low]** Renaming a custom role in the roles module makes every holder's token stale (role name is compared). The result is a forced re-auth within the TTL, not a lockout. This is acceptable, but worth documenting in FR-BB117.
- **[Low]** `backend/internal/users/service.go:174`, a stray double blank line in `CreateUser`. Collapse it to one. `gofmt` reports every file here because of CRLF working-copy endings, so check with `gofmt -d` on this file.
- **[Low]** Import returns the ambiguous-department message to the caller. That is fine (the department name is caller-supplied), and no ids are leaked.

## AC Coverage

- Strict permission subset: covered
- No self role change or self deactivate, including super_admin: covered
- 404 for unknown and out-of-scope targets: covered
- Malformed UUID -> 404/422, no 500: covered
- Stale JWT claims (status, role, department) rejected, 10 s cache: covered (see the two Medium findings on the freshness window)
- Import department ambiguity and ids resolved once per row: covered

Summary: PASS. The security logic is sound and the claim comparison matches how tokens are issued. Follow up on frontend handling of `TOKEN_REVOKED` and on cache invalidation after user mutations.

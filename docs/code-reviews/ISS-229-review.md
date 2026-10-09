# Code Review: ISS-229 (per-caller `assignable` flag on GET /users/roles)

Reviewer: Code Reviewer subagent. Static review only (no builds or tests run, per instruction).
Verdict: **PASS** (0 Critical, 0 High, 1 Medium, 2 Low, all non-blocking)

## Scope
backend/internal/users/{handler.go, handler_test.go, roles_assignable_test.go, service.go, types.go},
frontend/src/api/users.ts, frontend/src/lib/assignableRoles.ts, frontend/src/lib/assignableRoles.test.ts,
docs/issue-reports/ISS-229-roles-assignable-flag.md.

## Goal check
- Shared logic: `checkRoleAssignment` now delegates to the extracted `roleAssignableBy(roleName, callerRole)` (canReachRole + custom-role sensitive-permission ban). `ListRoles` calls the same function. The only difference is the super_admin short-circuit, which is `isOrgWide` in both. No drift is possible. OK.
- Fail-closed: unknown, empty or ghost callers and roles return ErrForbidden, so `assignable=false`. OK.
- No permission leak: only a boolean is added (`db:"-" json:"assignable"`). Other roles' permission lists are not exposed, and the handler test asserts there is no `permissions` key. The caller role comes from the auth context (`auth.RoleFromCtx`), not from request input. OK.
- Layering and conventions: the handler stays thin, the SQL is unchanged (`db:"-"` keeps sqlx from scanning the field), errors pass through the existing wrapped repo error, and the envelope is unchanged. The route already exists and sits behind JWT. OK.
- Backward compatibility: the field is additive. The TS field is optional, and `canAssignRole` falls back to local evaluation when the flag is not a boolean. OK.
- Tests: `TestListRoles_FlagMatchesCheckRoleAssignment` compares the flag with enforcement across 9 callers and every fixture role. There are also per-caller matrices, a handler test and frontend flag/fallback tests. Coverage is good. The `ListRoles` mock signature was updated in both mocks.

## Findings
1. **Medium (non-blocking), stale per-caller cache.** `useRoles()` uses queryKey `['users','roles']`, which does not include the caller. The flag is now caller-specific, and `canAssignRole` prefers it over local evaluation. Nothing in `frontend/src` clears the query cache on logout. If user A logs out and user B logs in within the same SPA session, B can briefly see A's cached flags (for example a department_admin's flags shown to an examiner) until the refetch lands. The default staleTime of 0 triggers a refetch on mount, and the backend still enforces on create and update, so this is cosmetic or UX only and not a security hole. Suggested follow-up: add the caller role to the queryKey, or clear the cache on logout.
2. **Low, intentional design.** Preferring the server flag also overrides local deny. For example, with a stale flag `true` a user sees a role the backend would reject. The backend remains authoritative (403 is localized per ISS-226). Acceptable.
3. **Low, minor.** `ListRoles` calls `roleAssignableBy` once per role, and each call may hit `permsFor` (an in-memory cache lookup). The cost is negligible for a role list.

## Checklist notes
- No secrets, no `os.Getenv`, no SQL changes, no new routes or migrations, and no state change (so no audit log is needed).
- No new user-visible strings, so i18n is unaffected.
- The frontend API module still uses `apiFetch` with the token.

## Result
PASS. Finding 1 is optional follow-up and need not block release.

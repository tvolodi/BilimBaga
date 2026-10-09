# Code Review: ISS-250 (router-level users authz test, tests only)

Verdict: APPROVE (0 Critical, 0 High)

Verified: `go test -p 2 -count=3 ./internal/router -run RouterUsersAuthz` passes on 3 repeated runs, and `go vet` is clean. `npx vitest run --maxWorkers=2 UsersListPage.roles.test.tsx` passes, 3 of 3. No source files were modified.

## Findings

1. [Medium] users_authz_test.go ~L412-441 (StaleClaims): the subtest closures call `prep()`, which mutates the `repo` variable captured from the outer scope. The subtest reassigns `h, repo = newUsersRouter(t)` before calling `prep()`, so it works today. It is fragile: the outer router built at the top of the test is unused, and `prep` only works because of the variable-capture ordering. Suggestion: give `prep` a `*authzRepo` parameter and build the router inside the subtest.
2. [Medium] users_authz_test.go L301-305 (StrictSubset): the test mutates `repo.users[...]` directly without taking `repo.mu`. It is safe because requests are sequential and in-process, but it bypasses the lock the rest of the harness uses. Suggestion: add a locked `setRole` helper.
3. [Low] users_authz_test.go L324 (Examiner 403): every route receives the same `updateBody` payload, including import and deactivate. The 403 comes from RBAC before body parsing, so the assertion is valid. It would be more meaningful to also assert `errCode == "FORBIDDEN"` (or whichever code RBAC emits), and to confirm that a department_admin gets past the RBAC layer on the same routes. The latter is covered indirectly by the 200 cases elsewhere.
4. [Low] users_authz_test.go L435-438 (StaleClaims): the PUT assertion checks only the error code and not the status. Add an `assert.Equal(401, ...)`.
5. [Low] users_authz_test.go L458: `strings.Contains` is used on the body, while `errCode` already exists. Use `assert.Equal(t, "TOKEN_REVOKED", errCode(t, rec))` for consistency.
6. [Low] The test relies on the real `time.Now()`, with iat set to now minus 1 minute and exp set to now plus 1 hour. It does not depend on cache TTL expiry, because the ISS-248 test checks invalidation, not expiry. This is not flaky.
7. [Low] UsersListPage.roles.test.tsx: the `/api/v1/users/roles` mock returns all four roles regardless of caller, so the test correctly exercises client-side filtering. There is no positive control showing that `super_admin` sees `department_admin` and `super_admin` in the dropdown. Without that control, a bug that hides all roles would still pass the `queryByRole(...).toBeNull()` checks, although the `waitFor` on `employee` and `examiner` guards against that. Optional: add a super_admin case.
8. [Low] UsersListPage.roles.test.tsx L63 and L72 rely on the `Edit` button order (`getAllByRole(...)[0]` and `[1]`). The order follows the fixed mock data, so it is deterministic. The names are not tied to the rows, though. Suggestion: scope with `within(row)`.
9. [Low] ISS-250 doc: "Tests: see PR" is vague. Record the pass counts instead. Also `severity: low` is fine.

## Non-vacuity and behavior match

- The router is the real `router.New`, with real `auth.Authenticate` plus the DB-backed account-state cache (via a driver stub), real `rbac.RequirePermission`, and a real `users.Service` and `Handler`. Only the repository is faked, and it mirrors the real UUID validation.
- Each negative case also asserts that no write occurred (`repo.writes` is zero or the snapshot is unchanged), and each has a positive control (proper subset gives 200, self-service without a role change gives 200, 404 parity covers 5 operations). These are non-vacuous.
- The frontend asserts a positive presence (`employee`, `examiner`) before the negative absence checks, and checks `disabled` plus `value` for a peer admin.
- Not run: mutation testing (removing a guard to confirm the test fails). It is recommended for a follow-up but is not required.

## Acceptance criteria

The ISS-250 scope is covered: strict-subset, examiner RBAC 403, no self role change or deactivate, 404 parity, malformed UUIDs, TOKEN_REVOKED, ISS-248 immediate revocation, and the frontend dropdown.

Summary: Tests are real, deterministic, and non-vacuous. Only Medium and Low polish items remain, so APPROVE.

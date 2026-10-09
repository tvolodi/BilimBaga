# Code Review ISS-217 (FR-BB117 D-1 target-role rule)

Verdict: PASS (0 High, 1 Medium accepted-residual, 3 Low). `go test ./internal/users/...` passes.

## Findings
- Medium (residual, matches spec decision): built-in caller vs built-in target skips the subset check, so department_admin can reset password / deactivate examiner, whose perms (questions:write, exams:write) department_admin lacks -> lateral takeover of a higher-privilege-in-those-areas account within the same department. Cannot reach super_admin or custom roles beyond own perms, so no org-level escalation. Acceptable as documented historical behaviour; recommend follow-up to make examiner/employee an explicit allow-list for department_admin rather than a blanket built-in skip.
- Low: fail-closed on unknown roles covers only RoleName=="" (repo uses INNER JOIN roles so legacy can't occur in prod). A role name absent from rbac cache (stale cache after role create) yields empty perms -> subset trivially true -> allowed. Consider refusing when a non-built-in target has no cache entry. Same pre-existing behaviour in checkRoleAssignment.
- Low: if permsFor/canPerm is nil (misconfiguration) the subset check is silently skipped; main.go wires both, so only a test-mode fail-open.
- Low: ImportUsers/CreateUser only create (checkRoleAssignment, now sharing checkRoleReach incl. same-role shortcut, which is fine: role names are UNIQUE and system roles immutable so a custom role cannot impersonate a built-in name). Import cannot update existing users (duplicate email rejected).

## Coverage
Only users mutation paths are in internal/users: Create, Update, Deactivate, ResetPassword, Unlock, Import (create only). All four target-based ones now call checkTargetActionable after scope check and before generateTempPassword/repo writes/email (TriggerPasswordReset after UpdatePassword). UpdateUser check precedes self-role-change and checkRoleAssignment (new role) so both old and new role are gated. Other UPDATE users statements are auth-owned (login lockout counters, self password change, token-based recovery, bootstrap) - not admin-caller paths. RemindEmployee is a stub. roles package does not write users. Router: all mutating routes require users:manage.

## Tests
Table-driven, covers 4 actions x forbidden/allowed matrix, no-write and no-temp-password assertions, cross-dept, self-service, handler 403 bodies. Gaps: no test for a stale/unknown non-built-in role name, nor nil permsFor; allowed "update/legacy" branch is weakly asserted.

## Cycle 2 (rank rule, FR-BB117 D-1 amended; commit d0012b2 + follow-up)

Verdict: PASS (0 Critical, 0 High, 2 Medium, 3 Low). `go vet` and `go test -count=1 -p 1 ./internal/users/...` clean.

Scope: explicit `builtinRank` table and single decision function `canReachRole`; target checks (reset, update, deactivate, unlock), assignment checks (create, update, import), self-service, fail-closed.

Verified: strictly-lower-rank rule for built-in pairs (peer department_admin, examiner vs examiner/department_admin, employee vs anyone all 403); custom callers refused department_admin/super_admin targets and otherwise bound by permission subset; no path lets a lower-rank caller affect a higher-rank target; update with unchanged role skips only the assignment check (target check still runs); self-service requires same id and stored role equal to token role; authorisation precedes password generation, writes and email; missing permission lookup or empty/unknown names give 403.

Findings:
- Medium (fixed): unknown/stale role with empty permission list counted as a subset. `permissionSubset` now refuses an empty permission list.
- Medium (accepted): sensitive-permission ban (roles:read/roles:manage/tenant:manage) applies to assignment only; a custom caller can still act on a same-role peer, as D-1 allows equal permission subsets.
- Low: custom peers manageable while built-in peers are not (matches D-1 wording); self-deactivation still allowed (unchanged); RevokeAllTokens error ignored (pre-existing).

Cycle 1 Medium (department_admin acting on peers/examiners via blanket built-in skip) is resolved by the rank table.

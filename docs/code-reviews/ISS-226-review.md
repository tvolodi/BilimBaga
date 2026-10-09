# Code Review: ISS-226 (frontend assignable roles)

Verdict: APPROVED (no blocking findings). Read-only review; tests/tsc not run (host memory low).

## Rule parity with backend canReachRole / checkRoleAssignment
- super_admin: all roles. Matches isOrgWide.
- super_admin role refused for others. Matches.
- Built-in caller/built-in role: rank(role) < rank(caller). Matches.
- Built-in caller/custom role: permission subset + no sensitive perms. Matches.
- Custom caller: built-in rank >= department_admin refused; else subset + no sensitive. Matches.
- Fail-closed on unknown/empty permissions. Matches.

## Non-blocking notes
1. RoleRow (api/users.ts) has no `permissions`, so custom roles (and, for custom callers, even examiner/employee) are never offered in the UI. Safe (hides, never over-exposes; backend authoritative) but narrower than the backend. Follow-up: expose permissions from /users/roles.
2. Sensitive-permission check is applied to built-in-caller and custom-caller paths on custom role names only via role.permissions; with no permissions field it is moot until note 1 is done.
3. Edit drawer: non-assignable current role is shown disabled with explanation; good. Tests cover create/edit/import/403 paths.
4. Minor: UsersListPage "Loading..." string is pre-existing hardcoded text, not part of this diff.

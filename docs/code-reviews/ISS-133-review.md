# Code Review: ISS-133 (Users list role filter)

Result: PASS

Scope: backend/internal/users/handler.go (+handler_test.go, service_test.go), frontend/src/pages/admin/users/UsersListPage.tsx (+test), docs/issue-reports/ISS-133-users-role-filter-broken.md.

## Findings
- [Medium] handler.go:50-56 - 422 on non-UUID role_id is correct, but department_id is not validated the same way (same 500 risk if non-UUID). Out of scope; consider follow-up.
- [Low] UsersListPage.tsx - role filter has no loading/error state for useRoles; the select degrades to only the "all" option while loading or on failure. Acceptable.
- [Low] service_test.go TestListUsers_RoleIDFilter exercises the mock repo, so it does not verify the SQL. The SQL (`u.role_id = $n`) is unchanged and live-DB check is deferred to UAT, as the issue report states.

## Checklist
- Parameterized SQL unchanged; no secrets; no os.Getenv. google/uuid already in go.mod (v1.6.0).
- Error envelope via api.WriteError, 422 VALIDATION_ERROR, consistent with other handlers in the file.
- Handler stays thin; no SQL added to handler.
- Frontend uses the existing useRoles hook (apiFetch with token), no raw fetch. Option value = role id; label via existing users.roles.* i18n keys (present in en; defaultValue fallback for custom roles). aria-label reuses an existing key, so no new strings.
- Tests: handler pass-through and 422 (service not called), service filter, frontend regression (UUID sent, not name; cleared on reset). Issue report states full suites pass (not re-run here; memory constraint).

## Verdict
PASS: zero Critical, zero High findings. The fix addresses the root cause (names sent as UUID).

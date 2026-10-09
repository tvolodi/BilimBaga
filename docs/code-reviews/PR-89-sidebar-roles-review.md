# Code Review: PR #89 (swarm/49-sidebar-links) - Sidebar role-based links

**Verdict: PASS** (no Critical/High/Medium findings)

## Checks
- Role sets vs `backend/migrations/005_rbac.up.sql`: `audit:read` is granted only to super_admin (AUDIT_READ_ROLES = [super_admin]); `reports:read` is granted to super_admin, department_admin, examiner (REPORTS_READ_ROLES matches). Employee has neither. Match.
- `jwtRole` robustness: null/undefined/empty token returns undefined; malformed token (missing segments, bad base64, bad JSON) is caught by try/catch and returns undefined; base64url (`-`/`_`) is normalized, and `atob` tolerates missing padding. Unknown role leads to gated links hidden (fail closed).
- Authority: links are hidden only as a UX nicety; `RequireRole` route guards (using the same role constants) remain the enforcement layer, and backend RBAC is authoritative. The token is decoded without signature verification, documented in the helper's comment.
- Tests: Sidebar.test.tsx (10 tests) pass, including a per-role matrix; `tsc --noEmit` clean.

## Findings
- Low: `jwtRole` is now duplicated in RequireRole.tsx, RequireSuperAdmin.tsx and ChangePasswordPage.tsx, in addition to the new exported helper in routeRoles.ts. Suggest follow-up consolidation (not blocking).
- Low: Sidebar reads the token with `getQueryData` (non-reactive), same pattern as RequireRole; acceptable because the sidebar is mounted under guarded routes and remounts on login. No test for malformed-token case in jwtRole directly (covered implicitly).

No fixes required.

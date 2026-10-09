# Code Review: ISS-052 route guards (self-review, subagent unavailable)

Verdict: PASS

- Correctness: guard lists match RBAC seed (audit:read super_admin; reports:read super_admin/department_admin/examiner). hr_admin removed (unseeded).
- Scope: RequireRole default behaviour unchanged; new prop optional. Sidebar still lists both items for all admin roles (out of scope; not a security issue, backend enforces). Noted as follow-up.
- Redirect to /login for a signed-in user: LoginPage navigates by role only after a login action, so no redirect loop.
- i18n: no user-visible strings added.
- Tests: full role x route matrix plus no-token; tsc, lint, 309 tests green.

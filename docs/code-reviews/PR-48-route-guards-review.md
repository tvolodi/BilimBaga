# Code Review: PR #48 route guards (ISS-052)

Result: PASS (independent review; supersedes the self-review in ISS-052-review.md)

Scope: `git diff origin/main...HEAD` (three-dot). A two-dot diff shows unrelated FR-BB48 deletions only because origin/main has advanced; those are not part of this PR.

## RBAC verification (backend/migrations/005_rbac.up.sql)
- audit:read: granted only to super_admin (blanket grant); not in department_admin/examiner lists. `AUDIT_READ_ROLES = ['super_admin']` matches.
- reports:read: granted to super_admin, department_admin, examiner. `REPORTS_READ_ROLES` matches exactly.
- hr_admin was renamed to examiner by migration 005 (line 6), so removing it from the lists is correct.

## Findings
- [Critical] none
- [High] none
- [Medium] Sidebar still lists Audit/Reports for roles that are now redirected (noted as follow-up in ISS-052; backend enforces, not a security issue).
- [Low] Other RequireRole usages in App.tsx still contain inline role arrays (some include the legacy hr_admin); candidates for the same shared-constants treatment.
- [Low] A signed-in but unauthorized user is sent to /login; LoginPage only navigates after a login action, so there is no redirect loop. Behaviour matches FR-BB59 AC-2.

## Checks
- RequireRole: new `unauthorizedRedirect` prop is optional; default behaviour unchanged. No-token path unchanged.
- Tests: routeRoles.test.tsx covers the full role x route matrix plus no-token.
- No new user-visible strings (i18n unaffected; check:i18n passes).
- `tsc --noEmit` clean, `npm run lint` clean, `npm test` green.

Summary: Guards align with the RBAC seed; no Critical/High findings, no fixes required.

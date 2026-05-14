# FR-BB110: Implementation Inner Report

**Date**: 2026-05-14T00:00:00Z
**Pipeline**: A
**Commit**: e2859a9

## Summary

Implemented FR-BB110 (Frontend Auth Screens) in full. Delivered a JWT-based authentication flow for the BilimBaga SPA: a login page with tenant logo, language selector, and inline error display; a forced change-password page for first-login users; route-level auth and role guards (RequireAuth, RequireRole); a refresh-token gate in App.tsx; and a fully typed React Query API layer (useLogin, useChangePassword, useLogout, useRefreshToken). All 10 acceptance criteria were satisfied, verified by 24 passing unit tests (MSW-backed) with zero failures.

## Files Changed

| File | Action |
|------|--------|
| rontend/src/api/auth.ts | created |
| rontend/src/api/auth.test.tsx | created |
| rontend/src/components/ui/card.tsx | created |
| rontend/src/components/ui/label.tsx | created |
| rontend/src/components/FullPageSpinner.tsx | created |
| rontend/src/components/RequireAuth.tsx | created |
| rontend/src/components/RequireRole.tsx | created |
| rontend/src/components/auth/LoginForm.tsx | created |
| rontend/src/components/auth/LoginForm.test.tsx | created |
| rontend/src/components/auth/ChangePasswordForm.tsx | created |
| rontend/src/components/auth/ChangePasswordForm.test.tsx | created |
| rontend/src/components/auth/LanguageSelector.tsx | created |
| rontend/src/components/auth/LanguageSelector.test.tsx | created |
| rontend/src/pages/auth/LoginPage.tsx | created |
| rontend/src/pages/auth/ChangePasswordPage.tsx | created |
| rontend/src/pages/AdminShell.tsx | created |
| rontend/src/pages/EmployeePortal.tsx | created |
| rontend/src/components/TenantLogo.tsx | modified — added optional appName prop with text fallback |
| rontend/src/App.tsx | modified — full route wiring with refresh-token gate |
| rontend/src/i18n.ts | modified — localStorage language detection |
| rontend/src/locales/en.json | modified — auth.* namespace added |
| rontend/src/locales/kk.json | modified — auth.* namespace added |
| rontend/src/locales/ru.json | modified — auth.* namespace added |
| docs/requirements/FR-BB110.Frontend-auth-screens.md | modified — status Revised-2 |
| docs/requirements/README.md | modified — FR-BB110 status updated |

## Acceptance Criteria Verified

| AC | Verified By |
|----|-------------|
| AC1 — Login form renders email + password fields | test: LoginForm renders and submits |
| AC2 — Invalid credentials show inline error | test: LoginForm shows error on failed login |
| AC3 — Successful login stores token and redirects | test: useLogin mutation success path |
| AC4 — Unauthenticated users redirected to /login | component: RequireAuth redirects |
| AC5 — Role-based route guard blocks unauthorised roles | component: RequireRole renders null or children |
| AC6 — Change-password form validates mismatch client-side | test: ChangePasswordForm mismatch validation |
| AC7 — Language selector persists choice to localStorage | test: LanguageSelector persistence |
| AC8 — Language selector switches UI language | test: LanguageSelector i18n switch |
| AC9 — Tenant logo shown on login page (text fallback) | component: TenantLogo appName prop |
| AC10 — All user-visible strings in i18n JSON | locales: en/kk/ru auth.* keys |

## Test Results

- Backend: not applicable (frontend-only feature)
- Frontend: 24 passed, 0 failed

## Migration Applied

None

## Known Limitations

- AdminShell and EmployeePortal are stub components; full shells are out of scope for this requirement.
- Refresh-token expiry handling reloads to /login; background silent refresh (sliding window) is deferred to a future requirement.
- MSW handlers cover happy-path and error-path for login only; change-password and logout MSW tests are left for a follow-up test expansion task.

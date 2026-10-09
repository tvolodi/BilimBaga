# FR-BB115 Account Recovery - Inner Report

- Branch: swarm/33-account-recovery (Refs #33)
- Migration: 031_password_reset_tokens (.up/.down). NOT applied locally (no docker allowed); UAT/Infra applies it on the live stack. Migration lock held by dev1 until merge.
- Backend: auth recovery service/repository/handlers (forgot-password, reset-password), users unlock + is_locked, email template password_reset_link (en/ru/kk) + TriggerPasswordResetLink, routes under AuthLimiter, unlock under users:manage.
- Frontend: ForgotPasswordPage, ResetPasswordPage, api/recovery.ts, login link + success notice, public routes in App.tsx, admin unlock action, i18n auth.recovery.* (en/ru/kk).
- Tests: backend auth/users/email unit+handler tests, schemaguard covers new SQL (verified by sabotage); vitest pages/App/unlock tests; Playwright live spec e2e/account-recovery.spec.ts (not run, needs stack + Mailhog).
- Code review: PASS (no Critical/High). Medium: timing difference found vs unknown email (partly mitigated: purge moved to found path only; residual DB-write timing remains), throttle count-then-insert race (low impact), plain fetch in recovery.ts intentional (public, unauthenticated). Low: Referrer-Policy not set, access JWTs live until expiry, email lookup case-sensitive like login.
- Deviations: RBAC key users:manage (spec said users:update; no such key). Policy failure code VALIDATION_ERROR per AC-4.

# Static E2E coverage audit (no live stack) - 2026-10-09 (issue #14)

Method: keyword scan of `frontend/e2e/**` against `docs/requirements/FR-BB*` frontend/user-visible features.
Non-live baseline: `go test ./...` ok; vitest 46 files / 259 tests pass; i18n 730 keys in kk/ru/en.

## Hard blocker for live runs
Port 8080 is hardcoded in `vite.config.ts` proxy, `e2e/fixtures/seed.ts`, `exam-lifecycle.spec.ts`, `loyalty-narrative.spec.ts`; a foreign process (qoimeks-server) holds it. Tracked by issue #12.

## Coverage gaps (no dedicated spec; only touched, if at all, inside full-walkthrough.spec.ts)
| Area | FR | Finding |
|------|----|---------|
| Department management UI | BB113 | only walkthrough |
| Audit log viewer | BB114/BB55 | only walkthrough |
| Dashboard / per-exam analytics / reports hub | BB56-59 | only walkthrough, no dedicated assertions |
| Employee record page | BB58 | no spec references it |
| Certificate generation/verify (QR, PDF) | BB43/44 | seed only + loyalty narrative |
| Tab-switch / anti-cheat events | BB38 | no spec |
| AI question generation assist | BB71 | no spec |
| Forgot/reset password | BB110 | no spec |
| Portal locale switcher | BB316 | no spec (only locale in seed) |

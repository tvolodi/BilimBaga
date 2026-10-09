# Non-live coverage baseline - 2026-10-09 (issue #20, main a62e0ce)

## Backend (`go test -cover ./...`, all green)
| Package | Coverage |
|---------|----------|
| api, ctxkeys, router (no tests) | 0.0% |
| db | 13.8% |
| email | 32.7% |
| ai | 36.9% (client.go, handler Insights/LoyaltyNarrative, most repository funcs 0%) |
| audit | 37.7% |
| rbac | 40.0% |
| sessions | 43.1% |
| questions | 45.1% |
| reports | 46.8% |
| exams | 48.6% |
| tags / users | 50.0% |
| auth | 52.4% |
| categories 56.2, departments 61.3, portal 62.1, certificates 71.4, tenant 76.9, middleware 81.8, config 83.1, upload 85.0 | |
| health, ratelimit | 100% |
`cmd/api/main.go` (main, initLogger) 0%.

## Frontend (vitest v8: 46 files, 259 tests pass)
Total: lines 45.6%, branches 71.2%, functions 43.4%.
Zero-coverage src files (largest first): `App.tsx` (270 lines), `api/audit.ts`, `api/employees.ts`, `api/grading.ts`, `components/audit/*` (AuditFilterBar, AuditLogTable), `components/employees/*` (5 files), `components/grading/*`, `components/results/*` (QuestionBreakdownTable, ResultActions, ScoreDial), `components/ErrorBoundary.tsx`, `components/admin/Breadcrumb.tsx`, `hooks/useCountdownTimer.ts`.
Note: `@vitest/coverage-v8` is not a devDependency; it was installed with `--no-save` for this run.

## Follow-ups
Issues filed for the worst areas (see issue #20 comments). Frontend overlaps in-flight branch swarm/4-frontend-coverage (dev2).

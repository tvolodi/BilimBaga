# ISS-027 - Frontend coverage delta (2026-10-09)

Command: `cd frontend && npx vitest run --maxWorkers=2 --coverage` (`@vitest/coverage-v8@^2.1.9` already a devDependency on main, matching vitest 2.1.9; no change needed).

| Metric | Baseline 2026-10-09 (#20) | origin/main at start of #27 | After #27 |
|--------|------|------|------|
| Lines | 45.6% | 64.98% | 66.29% |
| Branches | 71.2% | 81.6% | 82.01% |
| Functions | 43.4% | 57.34% | 58.37% |
| Test files / tests | 46 / 259 | 89 / 637 | 92 / 652 |

Most of the baseline's named targets (employees, results, grading components, useCountdownTimer, audit table) were covered by other merged work before this issue; no `swarm/4-frontend-coverage` branch or PR exists on origin, so no overlap.

New tests: `api/grading.test.tsx` (grading.ts 0% to covered), `components/audit/AuditFilterBar.test.tsx` (0% to covered, fake timers for debounce), `components/admin/Breadcrumb.test.tsx`.

Still low (follow-up candidates): App.tsx, ExamWizard pages, ExamTaking pages, ResultPage, QuestionCreateDialog, api/analytics.ts, api/sessions.ts, api/exams.ts.

`tsc --noEmit` and `npm run lint` clean.

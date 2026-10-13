# FR-BB69 — Locale-Aware Date and Number Formatting Everywhere

## Metadata
| Field | Value |
|-------|-------|
| ID | FR-BB69 |
| Phase | 6 — Polish & Hardening (gap closure; completes roadmap 6.2 "number and date formatting via `Intl` respecting active locale") |
| Priority | 2 |
| Status | Draft |
| Depends On | FR-BB62, FR-BB320 |

## Description
FR-BB62 requires dates and numbers to follow the active app language. `frontend/src/utils/format.ts` (`formatDate`, `formatNumber`, `formatPercent`) exists, but at least these call sites use bare `toLocaleString()` / `toLocaleDateString()` and so follow the browser locale, not the language chosen in the app (a kk user on an English browser sees mixed formats):
`components/audit/AuditLogTable.tsx`, `components/employees/SessionHistoryTable.tsx` (2), `components/grading/GradingQueueTable.tsx`, `components/results/ResultsTable.tsx`, `pages/admin/questions/QuestionBankPage.tsx`, `pages/admin/tags/TagsPage.tsx`, `pages/ExamWizard/Step3Assignments.tsx`, `pages/ExamWizard/Step4Review.tsx`. Other sites pass `i18n.language` ad hoc (`ExamCard.tsx`, `ResultPage.tsx`, `ExamAnalyticsPage.tsx`), so the same instant renders differently across screens.

## Acceptance Criteria
- [ ] AC-1: All user-visible dates, times and numbers in `frontend/src` are produced through `utils/format.ts` helpers (extended with `formatDateOnly` and `formatRelative` as needed) taking the active `i18n.language`; no bare `toLocale*String()` and no `new Intl.*` outside `utils/format.ts`.
- [ ] AC-2: The relative-date logic in `QuestionBankPage` ("N days ago" up to 30 days, then absolute date) uses `Intl.RelativeTimeFormat` and the shared absolute formatter, with kk/ru/en output.
- [ ] AC-3: One instant renders identically on every screen for a given language (documented fixed `dateStyle`/`timeStyle` per helper); timestamps are displayed in the browser's timezone, ISO inputs remain UTC.
- [ ] AC-4: A lint guard (ESLint `no-restricted-syntax` rule, run in `npm run lint` and CI) fails on `.toLocaleString(`, `.toLocaleDateString(`, `.toLocaleTimeString(` outside `utils/format.ts` and test files.
- [ ] AC-5: Tests: Vitest for each helper in kk, ru, en (fixed instant, expected strings); component tests for at least `AuditLogTable`, `SessionHistoryTable`, `ResultsTable` switching language and asserting the rendered date changes; lint guard proven by a fixture failing the rule.
- [ ] AC-6: Visual baselines (FR-BB320) affected by changed date text are regenerated in the pinned Playwright container in the same PR.

## Technical Specification
Frontend only; no API or migration change. Extend `utils/format.ts`; add a `useFormatters()` hook returning helpers bound to `i18n.language` so components do not thread the locale; replace listed call sites; add the ESLint rule to `frontend/eslint.config.*`.

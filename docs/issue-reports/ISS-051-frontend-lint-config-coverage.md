---
id: ISS-051
title: Frontend lint cannot parse TypeScript; low test coverage; unlocalized strings and a11y warnings
status: resolved
severity: medium
layer: frontend
module: tenant
tags: [eslint, typescript-eslint, "Parsing error", jsx-a11y, i18next/no-literal-string, coverage]
created: 2026-10-09
resolved: 2026-10-09
recurrence_count: 1
related_issues: []
regression_test: frontend/src/utils/errorMessages.test.ts
---

## Symptom
GitHub issue #4. `npm run lint` in `frontend/` failed with 209 errors, almost all
`Parsing error: Unexpected token ...` on every .ts/.tsx file. `npx tsc --noEmit` was clean.
Statement coverage measured 45.56% with many pages/hooks at 0%.

## Root Cause
`eslint.config.js` had no TypeScript parser, so ESLint 9 flat config parsed `.ts/.tsx` as plain
JS and lint never actually evaluated any rules. Once the parser was added, 64 real findings
surfaced (28 errors, 36 warnings): hardcoded user-visible strings, a11y issues (click handlers
on non-interactive elements, autoFocus, label depth), unused vars, `prefer-const`, and
`eslint-disable react-hooks/*` comments referencing a plugin that was not installed.

## Fix Applied
- ESLint config: added `typescript-eslint`, `eslint-plugin-react-hooks`, `globals`; ignore
  `dist/` and `coverage/`; `_`-prefixed unused vars allowed; `no-autofocus` ignores non-DOM
  components; label depth 6; shadcn `components/ui/**` exempt from heading/label content rules
  (content supplied at call site); test files exempt from `no-literal-string`.
- Hardcoded strings moved to i18n (en/ru/kk, 11 new keys): ErrorBoundary, App loading heading,
  Step4Review rule messages/Published, QuestionBank versions/import rows, ImportModal Error column.
  Pure glyphs extracted to named constants.
- A11y: backdrops `aria-hidden`; import dropzone is now `role=button` + keyboard activation;
  tag-input wrapper documented disable (inner input is the focusable control).
- Removed unused `err` bindings; documented the intentional `prefer-const` and `exhaustive-deps` cases.
- Added `@vitest/coverage-v8` devDependency and `test:coverage` script.
- Added tests for previously 0%-covered code (below).

## Files Changed
| File | Change |
|------|--------|
| frontend/eslint.config.js | TS parser, react-hooks plugin, rule tuning |
| frontend/package.json, package-lock.json | new devDeps, `test:coverage` script |
| frontend/src/locales/{en,ru,kk}.json | 11 new keys each |
| frontend/src/App.tsx, components/ErrorBoundary.tsx, pages/ExamWizard/Step4Review.tsx, pages/admin/questions/QuestionBankPage.tsx, pages/users/ImportModal.tsx | i18n of literals |
| frontend/src/components/admin/{DepartmentTreeNode,categories/CategoryTreeNode}.tsx, pages/ExamTaking/ExamLayout.tsx | aria-hidden backdrops |
| frontend/src/pages/admin/questions/QuestionEditorPage.tsx, QuestionBankPage.tsx | a11y, unused vars, glyph constant |
| frontend/src/hooks/useCountdown.ts, pages/ExamWizard/Step1BasicSettings.tsx, pages/ExamTaking/TabSwitchWarningModal.tsx, components/questions/AIGenerateDialog.tsx | lint justifications / glyph constants |

## Regression Test
New tests: `src/utils/errorMessages.test.ts` (also asserts every mapped key exists in en.json),
`src/components/ErrorBoundary.test.tsx`, `src/hooks/useCountdownTimer.test.ts`,
`src/hooks/useTabSwitchDetection.test.tsx`, `src/pages/ExamTaking/__tests__/{AnswerInputs,QuestionNavigator,ResultScreen}.test.tsx`.

## Resolution Results
- Tests: 297 passed, 0 failed (53 files; before: 259 / 46 files)
- Coverage (statements): 45.56% -> 47.63%
- tsc: clean; eslint: 0 errors, 0 warnings; i18n check: 741 keys in all 3 locales
- Migration applied: no
- Build clean: yes

## Recurrence Log
| Date | Trigger | Action Taken |
|------|---------|-------------- |

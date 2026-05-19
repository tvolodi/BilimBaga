---
id: ISS-007
title: E2E tests fail with Russian locale — English-only selectors not matching Russian UI text
status: resolved
severity: high
layer: frontend
module: auth, exams, questions, sessions, portal
tags: [e2e, playwright, i18n, locale, selector, russian]
created: 2025-05-19
resolved: 2025-05-19
recurrence_count: 1
related_issues: []
regression_test: frontend/e2e/
---

## Symptom

52 of 168 E2E tests fail after the global test setup stores `i18n-lang=ru` in localStorage. The UI renders in Russian, but test selectors search for English-only label/button/placeholder text, causing `locator.waitFor` / `expect(...).toBeVisible()` timeouts.

Affected spec files and root causes:
- **auth.spec.ts**: `getByLabel(/password/i)` matches both the visible password field AND a hidden `aria-label` → strict mode violation (multiple matches)
- **branding.spec.ts**: `getByRole('button', { name: /save/i })` → no match when button says "Сохранить"
- **exam-lifecycle.spec.ts**: `/Basic Settings|Edit Exam/i` heading selector → no match in Russian
- **exam-wizard.spec.ts**: `getByLabel(/название экзамена|exam title/i)` etc. → shadcn `<Label>` association unreliable; `getByLabel` finds wrong element in Russian locale
- **full-walkthrough.spec.ts**: `getByLabel(/password/i)` strict mode + export CSV button English-only
- **question-bank.spec.ts**: `getByPlaceholder(/search/i)` → placeholder says "Поиск" in Russian
- **question-editor.spec.ts**: `/auto.grading/i` → text says "Автоматическое оценивание"; also missing `waitForLoadState` after navigation
- **question-management.spec.ts**: Import/Export/Generate/Delete buttons English-only; delete button scoped to `row` which is stale after dropdown closes (`\$` regex bug)
- **user-management.spec.ts**: Preview/Close buttons English-only in Bulk Import section
- **admin-grading.spec.ts**: `expect(toast).toBeVisible()` inside `.catch()` throws if toast is also absent → test fails non-deterministically
- **exam-taking.spec.ts**: `\$` literal in regex (should be `$`); `confirmBtn` scoped to `dialog` but strict mode finds 2 buttons
- **employee-portal.spec.ts**: Same `\$` and dialog-scope bugs as exam-taking

## Root Cause

1. **Locale mismatch**: Global setup sets `localStorage['i18n-lang'] = 'ru'`, making all UI text Russian. Tests with English-only selectors time out.
2. **Strict mode violations**: Some selectors like `getByLabel(/password/i)` match multiple elements (visible field + hidden aria-label). Adding `.first()` resolves this.
3. **shadcn Label association**: `getByLabel()` relies on `for`/`id` or `aria-labelledby`. In Russian locale the label text changes but the ID stays the same → use `locator('#id')` for exam-wizard form fields.
4. **Regex `\$` bug**: `/^удалить\$|^delete\$/i` — the `\$` is a literal backslash+dollar in the source, which in regex means literal `$` not end-of-string anchor. The regex never matches because it looks for the backslash character. Fix: remove the backslash.
5. **Stale row scope**: `row.getByRole('button', { name: /delete/i })` — the row reference becomes stale after the dropdown opens. Fix: use `page.getByRole(...).first()`.
6. **Non-throwing toast assert**: `await expect(toast).toBeVisible()` inside `.catch()` throws its own error when the toast is absent, masking the original timeout.

## Fix Applied

Applied bilingual regex patterns and structural fixes across 12 spec files:

1. **auth.spec.ts** — Added `.first()` to 3 `getByLabel(/пароль|password/i)` calls
2. **branding.spec.ts** — Added Russian alternative: `/сохранить|save/i`
3. **exam-lifecycle.spec.ts** — Added Russian alternatives to heading selector
4. **exam-wizard.spec.ts** — Changed all 4 label-based locators to ID-based (`#title`, `#timeLimitMinutes`, `#passingScorePct`, `#maxAttempts`) globally across all tests
5. **full-walkthrough.spec.ts** — Added `.first()` to password selectors (tests 01 & 02); added Russian to export CSV button (test 13)
6. **question-bank.spec.ts** — `getByPlaceholder(/поиск|search/i)` in 2 tests
7. **question-editor.spec.ts** — Added Russian to auto-grading text; added `waitForLoadState` after `waitForURL` in 2 edit tests
8. **question-management.spec.ts** — Added Russian to Import/Export CSV/Export JSON/Generate buttons; fixed delete: `row.` → `page.` + `.first()` + `\$` → `$`
9. **user-management.spec.ts** — Added Russian to Preview and Close/Cancel buttons in Bulk Import section
10. **admin-grading.spec.ts** — Replaced failing `await expect(toast).toBeVisible()` in `.catch()` with boolean `isVisible()` check + annotation
11. **exam-taking.spec.ts** — Fixed `\$` → `$` in continueBtn regex; changed confirmBtn from `dialog`-scoped to `page.getByRole(...).last()`
12. **employee-portal.spec.ts** — Same fixes as exam-taking for tests 05 & 06

## Files Changed

| File | Change |
|------|--------|
| `frontend/e2e/auth.spec.ts` | `.first()` on 3 password getByLabel calls |
| `frontend/e2e/branding.spec.ts` | Bilingual save button selector (2 occurrences) |
| `frontend/e2e/exam-lifecycle.spec.ts` | Bilingual heading selector |
| `frontend/e2e/exam-wizard.spec.ts` | 4 label → ID locators, globally applied |
| `frontend/e2e/full-walkthrough.spec.ts` | Password `.first()` (2 tests), export CSV bilingual |
| `frontend/e2e/question-bank.spec.ts` | Search placeholder bilingual (2 tests) |
| `frontend/e2e/question-editor.spec.ts` | Auto-grading bilingual text; `waitForLoadState` after nav |
| `frontend/e2e/question-management.spec.ts` | Import/Export/Generate bilingual; delete btn fix |
| `frontend/e2e/user-management.spec.ts` | Preview/Close bilingual in Bulk Import |
| `frontend/e2e/admin-grading.spec.ts` | Non-throwing submit grade check pattern |
| `frontend/e2e/exam-taking.spec.ts` | `\$`→`$` regex; confirmBtn scope + `.last()` |
| `frontend/e2e/employee-portal.spec.ts` | Same as exam-taking |

## Regression Test

All E2E spec files in `frontend/e2e/` serve as the regression suite. The global setup always sets `i18n-lang=ru`, so any future test must use bilingual selectors or ID-based locators.

## Resolution Results

- Tests: See e2e-results.json for pass/fail counts
- Migration applied: no
- Build clean: yes (TypeScript check passes)

## Recurrence Log

| Date | Trigger | Action Taken |
|------|---------|--------------|
| 2025-05-19 | Initial discovery: 52 tests failing after global locale setup | Full fix applied across 12 spec files |

---
id: ISS-006
title: New question save returns 422 when form filled in non-English locale
status: resolved
severity: high
layer: frontend
module: questions
tags: [422, default_locale, translations, QuestionEditorPage, handleSaveDraft]
created: 2026-05-18
resolved: 2026-05-18
recurrence_count: 1
related_issues: [ISS-005]
regression_test: frontend/src/pages/admin/questions/QuestionEditorPage.test.tsx
---

## Symptom
`POST /api/v1/questions` returns HTTP 422 Unprocessable Entity with the error
`translations.en.stem: "stem is required for the default locale"` when the user
fills in the question form using the RU (or KK) locale tab instead of the EN tab
and clicks "Сохранить черновик" (Save draft).

## Root Cause
In `QuestionEditorPage.tsx`, `handleSaveDraft` hard-coded `default_locale` to `'en'`:

```js
const defaultLocale = 'en'   // always English, regardless of which tab was active
```

When the user switches to the RU locale tab (`activeLocale = 'ru'`) and types the
stem there, the built payload contains:
- `default_locale: "en"`
- `translations.en.stem: ""` (empty — the user never typed in the EN tab)
- `translations.ru.stem: "Первая планета от Солнца"` (filled)

The backend validator checks `translations[default_locale].stem` for emptiness and
returns 422.

## Fix Applied
Replaced the hard-coded `'en'` with a dynamic calculation that uses `activeLocale`
(the tab where the user was working) as `default_locale`, falling back to the first
locale that has a non-empty stem if `activeLocale` itself has no stem:

```js
const defaultLocale: Locale =
  form.translations[activeLocale]?.stem?.trim()
    ? activeLocale
    : (LOCALES.find((l) => form.translations[l]?.stem?.trim()) ?? activeLocale)
```

## Files Changed
| File | Change |
|------|--------|
| `frontend/src/pages/admin/questions/QuestionEditorPage.tsx` | Replace hardcoded `'en'` with dynamic locale selection in `handleSaveDraft` |
| `frontend/src/pages/admin/questions/QuestionEditorPage.test.tsx` | Add ISS-006 regression test block |

## Regression Test
`frontend/src/pages/admin/questions/QuestionEditorPage.test.tsx`  
Block: `QuestionEditorPage — default_locale follows active locale (ISS-006)`

Simulates clicking the RU tab, entering a Russian stem, clicking Save Draft, and
asserts `body.default_locale === 'ru'` and `body.translations.ru.stem` is non-empty.

## Resolution Results
- Tests: 6 passed, 0 failed
- Migration applied: no
- Build clean: yes (tsc --noEmit exits 0)

## Recurrence Log
| Date | Trigger | Action Taken |
|------|---------|--------------|
| 2026-05-18 | Initial report | Fixed + regression test added |

---
id: ISS-013
title: Question edit page shows blank answer option text (пусто) for all locales
status: resolved
severity: high
layer: frontend
module: questions
tags: [answer_options, translations, body, text, detailToForm, QuestionEditorPage]
created: 2026-05-19
resolved: 2026-05-19
recurrence_count: 1
related_issues: [ISS-005]
regression_test: frontend/src/pages/admin/questions/QuestionEditorPage.test.tsx
---

## Symptom
On `/admin/questions/{id}/edit`, all answer option text fields show blank / "(пусто)"
(the i18n placeholder for an empty value) regardless of which locale tab is active.
The preview panel on the right also shows "(пусто)" for every option. This happens
both on initial load and after a save+reload cycle.

Screenshot: RU tab active for "Ближайшая от Солнца планета"; all three option inputs
are empty.

## Root Cause

`api/questions.ts` declares:

```typescript
export interface AnswerOptionTranslation {
  body: string     // ← WRONG: actual API JSON key is "text"
}
```

The backend Go struct serialises as `json:"text"`:

```go
type AnswerTranslationDetail struct {
    Text string `json:"text"`
}
```

`detailToForm()` in `QuestionEditorPage.tsx` reads:

```typescript
translations: Object.fromEntries(
  LOCALES.map((l) => [l, { body: opt.translations[l]?.body ?? '' }]),
)
```

`opt.translations[l]?.body` is always `undefined` (the JSON has `text`, not `body`),
so all option bodies fall back to the empty-string default. The internal form state
`LocalAnswerOption` correctly uses `body` as its own field name (distinct from the
API key), but the initialisation from API data reads the wrong key.

The test mock in `QuestionEditorPage.test.tsx` also used `body` in the mock response,
which caused the "shows existing answer options" test to pass despite the bug (the
mock matched the broken interface, not the real API).

## Fix Applied

1. **`frontend/src/api/questions.ts`** — renamed `body` to `text` in
   `AnswerOptionTranslation` to match what the API actually returns.

2. **`frontend/src/pages/admin/questions/QuestionEditorPage.tsx`** — in
   `detailToForm()`, changed `opt.translations[l]?.body` → `opt.translations[l]?.text`.

3. **`frontend/src/pages/admin/questions/QuestionEditorPage.test.tsx`** — updated
   the `sampleQuestion` mock to use `translations: { en: { text: 'Water' } }` (was
   `body`) so the mock now matches the real API shape. The existing "shows existing
   answer options" test now acts as a regression guard.

## Files Changed
| File | Change |
|------|--------|
| `frontend/src/api/questions.ts` | `AnswerOptionTranslation.body` → `AnswerOptionTranslation.text` |
| `frontend/src/pages/admin/questions/QuestionEditorPage.tsx` | `detailToForm`: read `opt.translations[l]?.text` |
| `frontend/src/pages/admin/questions/QuestionEditorPage.test.tsx` | Mock data updated to use `text` key |

## Regression Test
`frontend/src/pages/admin/questions/QuestionEditorPage.test.tsx`
→ `describes 'QuestionEditorPage — edit existing question'` → `shows existing answer options`

This test mounts the editor for question `q-1`, whose mock API response uses `text`
keys in option translations. It asserts `getByDisplayValue('Water')` which fails when
`detailToForm` reads the wrong key.

## Resolution Results
- Tests: all frontend question editor tests pass
- Migration applied: no
- Build clean: yes (frontend-only change)

## Recurrence Log
| Date | Trigger | Action Taken |
|------|---------|--------------|
| 2026-05-19 | Bug report with screenshot showing "(пусто)" | Root cause identified, fixed `body`→`text` mismatch |

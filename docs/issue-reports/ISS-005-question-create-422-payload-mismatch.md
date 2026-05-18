---
id: ISS-005
title: POST /api/v1/questions returns 422 — frontend sends flat stem/body instead of nested translations map
status: resolved
severity: high
layer: frontend
module: questions
tags: [422, question-create, translations, stem, body, text, payload-mismatch]
created: 2026-05-18
resolved: 2026-05-18
recurrence_count: 1
related_issues: []
regression_test: frontend/src/pages/admin/questions/QuestionEditorPage.test.tsx
---

## Symptom

When creating a new question at `/admin/questions/new` and clicking "Сохранить черновик", the
backend returns **HTTP 422 Unprocessable Entity** with validation error
`"stem is required for the default locale"`. The frontend shows
`"Не удалось сохранить. Попробуйте ещё раз."`.

Network tab: `POST http://localhost:5173/api/v1/questions → 422`.

## Root Cause

Two-part payload mismatch between frontend and backend:

1. **Question stem/explanation** — `QuestionCreatePayload` in `frontend/src/api/questions.ts`
   defines flat top-level fields `stem: string` and `explanation?: string`. The frontend's
   `handleSaveDraft` sends `{ stem: "...", ... }`. However, the backend's `createQuestionReq`
   expects `translations: map[string]translationReq` keyed by locale
   (e.g. `{ "en": { "stem": "...", "explanation": null } }`). Because `translations` is absent
   from the payload, `validateCreateRequest` finds no stem for the default locale and returns 422.

2. **Answer option body** — `AnswerOptionCreatePayload` defines `body: string` (flat). The
   backend's `answerOptionReq` expects `translations: map[string]answerTranslationReq` where
   each entry has `text` (not `body`). The create path sends `{ body: "..." }` causing answer
   text to be silently lost. Additionally, `buildUpdatePayload` in `QuestionEditorPage.tsx`
   also used `{ body: ... }` in option translations for the PUT path, meaning answer option
   text was never persisted correctly on update either.

## Fix Applied

**`frontend/src/api/questions.ts`**:
- `QuestionCreatePayload`: removed flat `stem`/`explanation` fields; added
  `translations: Record<string, { stem: string; explanation?: string }>`.
- `AnswerOptionCreatePayload`: removed `body: string`; added
  `translations: Record<string, { text: string }>`.
  (`AnswerOptionUpdatePayload extends AnswerOptionCreatePayload` inherits the fix automatically.)

**`frontend/src/pages/admin/questions/QuestionEditorPage.tsx`**:
- `handleSaveDraft` (create mode): build `translations` map from all non-empty locales in
  `form.translations`; always include `default_locale` entry. Build answer option
  `translations` with `text` key from `o.translations[locale].body`.
- `buildUpdatePayload`: removed extra top-level `body` field on options; changed option
  translations from `{ body }` to `{ text }` to match `answerTranslationReq`.

## Files Changed

| File | Change |
|------|--------|
| `frontend/src/api/questions.ts` | Updated `QuestionCreatePayload` and `AnswerOptionCreatePayload` interfaces |
| `frontend/src/pages/admin/questions/QuestionEditorPage.tsx` | Fixed `handleSaveDraft` create payload and `buildUpdatePayload` option translations |

## Regression Test

`frontend/src/pages/admin/questions/QuestionEditorPage.test.tsx` — existing tests verify the editor
renders correctly. A new test verifies that `handleSaveDraft` in create mode posts the correct
`translations` structure to the API.

## Resolution Results

- Tests: 197 passed, 0 failed (frontend); all backend packages pass
- Migration applied: no
- Build clean: yes (no TypeScript errors)

## Recurrence Log

| Date | Trigger | Action Taken |
|------|---------|--------------|
| 2026-05-18 | Bug report — 422 on question create | Investigate + fix payload mismatch |

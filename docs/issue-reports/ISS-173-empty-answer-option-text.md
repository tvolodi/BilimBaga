---
id: ISS-173
title: Question create/update accepts answer options with empty text (seed sent {"body"} instead of {"text"})
status: resolved
severity: medium
layer: backend
module: questions
tags: [answer_options, translations, text, INVALID_OPTION_TEXT, ERR_VALIDATION, optionTextFieldErrors]
created: 2026-10-09
resolved: 2026-10-09
recurrence_count: 1
related_issues: []
regression_test: backend/internal/questions/option_validation_test.go
---

## Symptom
POST/PUT /api/v1/questions returned 201/200 for choice questions whose answer option translation text was empty
or whose JSON used an unknown key (e2e seed sent `{"body": ...}`), silently storing blank options.

## Root Cause
`answerOptionReq.Translations` decodes into `{Text string}`; unknown keys are ignored by `json.Decoder` and `Text`
stays "". Neither `validateCreateRequest`, the service, nor the import validator checked option text.

## Fix Applied
- New `option_validation.go`: `optionTextFieldErrors` (every option of single/multiple/truefalse/likert needs non-blank
  text for the question's default locale; other locales may be empty/absent; shorttext skipped), `OptionValidationError`
  (unwraps to `ErrInvalidOptionText`), `answerOptionInputsFromReq` (dedupes handler conversion).
- Create: `validateCreateRequest` -> 422 `ERR_VALIDATION` with `fields[].field = answer_options[i].translations.<locale>.text`.
  `service.CreateQuestionFull` also validates (defence in depth, mapped to the same 422).
- Update: `service.UpdateQuestion` validates against the stored question's type and default_locale; handler maps to 422.
- Import (CSV/JSON): `ValidateAndImport` reports the same message as a row error.
- AI generation: no server-side question creation path (drafts go through POST /questions), so covered by create.
- `DisallowUnknownFields` NOT applied: other request DTOs/clients may send extra keys (update payload carries fields
  the DTO does not declare); blank-text validation already catches the `body` mistake.
- Fixtures (`frontend/e2e/fixtures/seed.ts`, `scripts/seed-test-env.ts`) already send `text` (fixed earlier in a45b92b); verified, no change needed.
- True/false and likert options are client-supplied (not server-generated), so they follow the same rule.

## Files Changed
| File | Change |
|------|--------|
| backend/internal/questions/option_validation.go | new validation helper and error type |
| backend/internal/questions/handler.go | create validation, 422 mapping for create/update, shared option conversion |
| backend/internal/questions/service.go | validate in CreateQuestionFull and UpdateQuestion |
| backend/internal/questions/import_export_service.go | row-level option text validation |
| backend/internal/questions/service_test.go | existing valid-option fixtures now carry text |
| backend/internal/questions/option_validation_test.go | new tests (service, handler, import) |

## Regression Test
`backend/internal/questions/option_validation_test.go`: empty, whitespace, missing default locale, non-default locale
empty OK, true/false, likert, shorttext, create/update at service level, handler 422/201, import dry-run.

## Resolution Results
- Tests: go test -p 1 ./... all pass; go vet clean; staticcheck 0 findings
- Migration applied: no
- Build clean: yes

## Existing data
Dev DBs may hold blank options created before this fix. Optional, read-only first:
```sql
SELECT ao.id, ao.question_id FROM answer_options ao
JOIN questions q ON q.id = ao.question_id
LEFT JOIN answer_translations t ON t.option_id = ao.id AND t.locale = q.default_locale
WHERE q.type <> 'shorttext' AND (t.option_id IS NULL OR btrim(t.text) = '');
```
(Read-only; not applied automatically.) Existing blank rows
will make the next edit of such a question fail with 422 until the options are filled in.

## Recurrence Log
| Date | Trigger | Action Taken |
|------|---------|-------------- |

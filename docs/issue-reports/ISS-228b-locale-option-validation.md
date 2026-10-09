---
id: ISS-228b
title: Partially translated non-default locale accepted with blank option text (#228 item 1)
status: resolved
severity: medium
layer: backend
module: questions
tags: [ERR_VALIDATION, INVALID_OPTION_TEXT, partialLocaleFieldErrors, FR-BB24 AC-11]
created: 2026-10-09
resolved: 2026-10-09
recurrence_count: 1
related_issues: [ISS-173, ISS-228]
regression_test: backend/internal/questions/option_validation_test.go
---

## Symptom
PUT /questions/:id/translations/:locale and question create/update accepted a non-default locale
with some options blank, producing half-translated options.

## Root Cause
Only default-locale option text was validated; non-default locales were unchecked.

## Fix Applied
BA ruling FR-BB24 AC-11: a non-default locale is "present" if its stem or any option text is
non-blank; then every option needs non-blank trimmed text, else 422 ERR_VALIDATION with
`fields`/`details` listing `answer_options[i].translations.<loc>.text`. A locale with no option
text and no stem stays accepted (falls back). Implemented in `option_validation.go`
(`partialLocaleFieldErrors`, `partialLocaleErrors`, `validateOptionTextsWithStems`), wired into
`CreateQuestionFull`, `UpdateQuestion` and `translationService.Upsert`. `writeValidationErrors`
additionally emits `details` (same list as `fields`). Status transitions and bulk import are
unchanged (legacy rows not migrated; import follow-up noted).

## Files Changed
| File | Change |
|------|--------|
| backend/internal/questions/option_validation.go | partial-locale rule |
| backend/internal/questions/service.go | create/update use stems-aware validator |
| backend/internal/questions/translation_service.go | PUT non-default locale check |
| backend/internal/questions/handler.go | `details` in 422 body |
| backend/internal/questions/option_validation_test.go | new + adjusted tests |

## Regression Test
option_validation_test.go: TestTranslationUpsert_NonDefaultLocalePartial, TestCreateQuestionFull_*,
TestUpdateQuestion_PartialNonDefaultLocale, handler 422 tests for PUT and POST.

## Resolution Results
- Tests: `go test -p 2 ./...` all pass; go vet clean; schemaguard green
- Migration applied: no
- Build clean: yes

## Recurrence Log
| Date | Trigger | Action Taken |
|------|---------|-------------- |

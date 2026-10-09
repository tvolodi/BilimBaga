# Code Review: ISS-170 (commit de4ca3b)

Verdict: **PASS** (no blocker or major findings; read-only review, no tests were run per memory constraint)

## Verified
- Backend contract matches: `INSUFFICIENT_ADAPTIVE_QUESTIONS` details is a single object `{rule_id, difficulty, required, available}` (model.go `AdaptivePublishValidationError`); `INSUFFICIENT_QUESTIONS` has no details; `EXAM_RULES_UNSATISFIED` details is an array. `parseInsufficientDetails` handles object, array, string, null, malformed numbers without throwing.
- `ExamApiError.details` added as trailing optional arg; existing callers unaffected. `unsatisfiedRules` now only populated for arrays, and Step4Review guards with `Array.isArray`, which removes the prior object-in-list crash/`[object Object]` path.
- i18n: en/ru/kk have all four new keys with matching interpolation variables; `noQuestionsError` and `questionBank.difficulty.*` exist in all three locales. No hardcoded user strings.
- `role="alert"` added to error containers (accessibility gain, enables tests).
- Step2 `useCallback` deps updated with `t`.
- Tests cover object/array/string/missing/malformed details and the per-rule list path.

## Findings

### Low
1. `Step4Review.tsx` archive handler: `describeExamError(apiErr, t) || t('exam.archive.error')`. `describeExamError` always returns a non-empty string (falls back to `genericError`), so the `exam.archive.error` fallback is now dead code and archive failures with no message show the generic text instead of the archive-specific one. Suggest `safeMessage(apiErr.message) || t('exam.archive.error')` or a fallback-key parameter.
2. `describeExamError` for unpublish/archive uses the same path; fine functionally, but the backend `message` (English) is shown untranslated for other codes (pre-existing behaviour, not a regression).
3. `Step4Review.errors.test.tsx` locates the confirm button as the last "Publish" button; slightly brittle to dialog ordering changes. Tests also assert English text, relying on default i18n language in test env.
4. `difficulty` is looked up as `questionBank.difficulty.${value}` with `defaultValue`; an unexpected value is shown raw (acceptable, safe from injection since React escapes).
5. When `required`/`available` are present only for `array` details, they are taken from the first rule only; acceptable since the backend returns a single object.

### Blocker / Major
None.

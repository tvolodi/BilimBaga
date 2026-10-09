# Code Review: ISS-173b (blank default-locale option text in translation Upsert and activation)

Verdict: APPROVE (read-only review; no tests or builds were run because of host memory limits)

## Scope
backend/internal/questions: translation_service.go, translation_handler.go, service.go, handler.go, option_validation_test.go, service_test.go (mock only).

## Findings
- Correctness (OK): Upsert validates only when `locale == q.DefaultLocale`. It reuses `validateOptionTexts`, so short-text and blank non-default locales stay allowed. The check runs after the MissingOptions check and before `LoadLocaleTranslation` and the repo upsert, so nothing is persisted on rejection (test asserts `upsertCalls == 0`).
- Index ordering (OK): `ListAnswerOptionIDs` is documented as sorted by sort_order, and `GetWithDetails` loads options with `ORDER BY sort_order`. Field indexes `answer_options[i]` therefore match the order that the create/update validation uses. The activation path rebuilds inputs from `detail.AnswerOptions`, with a missing default-locale entry treated as blank (tested).
- Activation (OK): the check runs only for `newStatus == "active"`, after the transition-table and stem checks. Other targets (review, archived, draft) never call `GetWithDetails` (tested). A `GetWithDetails` failure is wrapped and returned, not swallowed.
- Error mapping (OK):
  - Upsert: `errors.As(*OptionValidationError)` is matched first and goes to `writeValidationErrors`, which returns 422 ERR_VALIDATION with `fields`.
  - TransitionStatus: `errors.Is(ErrInvalidOptionText)` returns 422 ERR_VALIDATION with `fields`, using `errors.As` for the details. The envelope is `{data:null,error:{...}}`.
  - Wrapped errors (`%w`) are unwrapped correctly in both paths.
- Cleanup (OK): removing the `_ = q` placeholder is correct, since `q` is now used.
- SQL (OK): no SQL or repository files are changed. The only repo-side change is the test mock.
- Tests (OK): table-driven coverage for single, multiple, truefalse, likert, shorttext, whitespace, non-default locale, activation rejection (status unchanged), non-activation transitions, and both handler mappings. The mock `detailOptions` field is additive, so existing tests are unaffected.

## Minor notes (non-blocking)
1. The activation 422 message differs from the generic "validation failed" used elsewhere. This is acceptable because it is more specific, and the `fields` payload is consistent.
2. The "active -> archived with blank options" case is tested. Blank legacy rows can still stay active until the next activation, which is the intended scope per the issue.
3. Activation costs one extra `GetWithDetails` query, which is negligible.
4. The new tests were not executed in this review. The Test Runner must run `go test ./internal/questions/...` before release.

## Regression risk
Low. Behavior changes only for (a) default-locale translation PUT with blank option text and (b) activation of choice questions with blank default-locale option text.

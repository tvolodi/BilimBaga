# Code Review: ISS-173 (empty answer option text)

Run ID: iss-173
Result: PASS

## Scope
backend/internal/questions/{option_validation.go, option_validation_test.go, handler.go, service.go, service_test.go, import_export_service.go}, docs/issue-reports/ISS-173-empty-answer-option-text.md

## Findings
- [Medium] service.go UpdateQuestion: an update with zero answer options passes validation (nothing to check). This is consistent with the existing behaviour and out of scope for this issue, but a choice question updated with an empty options list is not rejected here.
- [Medium] Existing blank options in dev data will make the next edit of such a question return 422. Documented in the issue report with a read-only SQL query; acceptable.
- [Low] handler.go Create: validation runs in both `validateCreateRequest` and `CreateQuestionFull` (defence in depth). The duplication is intentional and the 422 mapping is consistent.

No Critical or High findings.

## Checklist notes
- Handlers stay thin. The validator lives in its own file and the service enforces it.
- Errors are wrapped with `fmt.Errorf("...: %w")`. `OptionValidationError` unwraps to `ErrInvalidOptionText`, and the handler uses `errors.As`.
- Create and update return 422 `ERR_VALIDATION` through the existing `writeValidationErrors`, with fields shaped as `answer_options[i].translations.<locale>.text`.
- Update uses the stored `q.Type` and `q.DefaultLocale`, so the client cannot bypass validation through the request payload.
- Import reports row errors using the same helper, and shorttext is skipped.
- Other locales may be empty, which is correct. A nil map lookup returns the zero value, so there is no panic.
- No SQL, secrets, or env access in the diff. No migration is needed. No debug output.
- Not applying `DisallowUnknownFields` is justified in the issue report.
- The service_test.go fixture updates are legitimate: the options now carry text.
- Tests cover empty, whitespace, missing default locale, nil translations, per-type cases, service create and update, handler 422 and 201, and import. The caller reports go vet, staticcheck and `go test -p 1 ./...` all green. I did not re-run them.

## AC Coverage
No requirement doc (Pipeline B). The issue report's fix list is fully implemented: create, update, import, and shorttext/other-locale exemptions.

Summary: A correct, well-tested shared validator is applied on all three write paths with the right 422 contract; only minor notes remain.

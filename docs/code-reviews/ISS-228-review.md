# Code Review: ISS-228

Result: PASS

Scope: uncommitted diff. Files: backend/internal/questions/service.go, option_validation_test.go, QuestionBankPage.tsx, e2e specs (ai-assist, question-bank, question-management).

## Findings
- [Medium] service.go TransitionStatus: the stem check and the option-text check now use identical `review || active` gates. Merge the two blocks. Not required.
- [Low] option_validation_test.go TestTransitionStatus_ReviewBlankOptionText: `wantIdx` values are never compared, only its length. Either assert the field names or use a count.
- [Low] The review gate runs the stem check before the option check, so a question with a blank stem and blank options gets ErrStemRequired. This is acceptable and consistent with active.
- Info: ai-assist.spec.ts still uses name regexes for the in-dialog submit button (`^ai generate$`). These are anchored and scoped to the dialog, so there is no strict-mode risk.

## Checks
- Errors are wrapped with context. The validator and the 422 ERR_VALIDATION envelope are reused (same path as active), and the handler test asserts 422 and `data:null`.
- No SQL, secrets or RBAC changes. The existing transition table is unchanged. Audit behavior is unchanged.
- Tests: `go test ./internal/questions/` passes and `go vet` is clean. The review-path cases cover blank-default rejection (status stays draft), a valid move and short-text unaffected. draft->review was correctly removed from the "ignored targets" test.
- Frontend: `data-testid="ai-generate-button"` sits on the page-level button only, so the dialog submit button is unaffected. All three specs now use getByTestId, and no stale locators remain. No user-visible strings were added, so no i18n impact.
- E2E was not run, as instructed.

AC Coverage: (2) ->review option validation implemented and tested. (3) locator fixed. (1) deliberately left out pending a BA decision (documented in the issue report).

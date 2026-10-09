# Code Review ISS-228b (run swarm-228b)

Result: PASS

Verification: `go vet ./internal/questions/` clean; `go test ./internal/questions/` ok.

Findings:
- [Medium] backend/internal/questions/handler.go:~88 - `details` duplicates `fields` in the 422 body (kept for backward compatibility). Acceptable; consider retiring `fields` later.
- [Low] option_validation.go - On create/update, only the payload is checked; stored non-default-locale stems not in the payload are not considered. Matches spec (legacy rows handled by activation check, #228 item 2).
- [Low] translation_service.go Upsert - indexes follow ListAnswerOptionIDs order; assumed to be sort_order (matches create/update indexes).

AC Coverage:
- AC-11: covered. Present locale (stem or any option non-blank) with any blank option gives 422 ERR_VALIDATION listing indexes; fully empty locale accepted; applies to PUT translations and create/update; shorttext exempt; import and status transitions unchanged (they still use validateOptionTexts).
- Default-locale rule (ISS-173b): unchanged.

Summary: Implementation matches AC-11 with good table-driven service and handler tests; no Critical/High issues.

# Code Review: ISS-158 (audit user_id validation)

Run ID: ISS-158 (Pipeline B)

Result: PASS

Files reviewed: backend/internal/audit/handler.go, backend/internal/audit/handler_test.go,
docs/issue-reports/ISS-158-audit-user-id-validation.md, docs/issue-reports/README.md

Verification: `go test -p 1 ./internal/audit/` ok; `go vet ./internal/audit/` clean.

## Findings
- [Medium] handler.go parseFilters — if `actor_id` and `user_id` are both given, `actor_id` silently wins; the conflicting `user_id` is validated but ignored. Documented in the issue report; acceptable.
- [Low] handler_test.go — no test for precedence when both params are present, or for `actor_id` valid + `user_id` invalid (expected 422). Optional.
- [Low] Frontend/API docs do not mention the `user_id` alias; optional.

No Critical or High findings. SQL remains parameterized, the filter reaches the existing allowlisted ActorID path, the error uses the standard 422 VALIDATION_ERROR envelope via api.UUIDQuery, List and Export share parseFilters so both are covered, and no migration or audit-write changes are needed (read-only endpoint).

## Regression coverage
- user_id=bad -> 422 VALIDATION_ERROR on both List and Export, service not called.
- valid user_id -> mapped to AuditFilters.ActorID.

## Summary
Minimal, correct fix with regression tests; PASS.

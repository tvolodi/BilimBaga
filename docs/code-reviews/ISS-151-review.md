# Code Review: ISS-151 (Run iss-151, Pipeline B)

Result: PASS

Files reviewed: backend/internal/exams/service.go, service_test.go, handler_test.go, docs/issue-reports/ISS-151-adaptive-empty-publish.md

## Findings
- No Critical or High findings.
- [Low] service_test.go - TestPublish_Adaptive_WithValidRule_Succeeds sets countAvailableForRuleFn to 15 and per-difficulty to 5; fine, covers the boundary (>=5).
- [Low] handler_test.go - the new handler test uses a mocked service, so it mostly duplicates the existing non-adaptive wrapped-error mapping; harmless, confirms errors.Is mapping (handler.go:349) works through wrapping.

## Notes
- Fix removes the `!e.Adaptive` guard; the check precedes the adaptive per-difficulty loop, so adaptive exams with zero rules now return a wrapped ErrNoQuestionRules (422 INSUFFICIENT_QUESTIONS). Error is wrapped with context, no behavior change for non-adaptive exams.
- No SQL, config, route, or migration changes; no frontend changes needed (shared code path).
- Regression tests cover: adaptive+zero rules refused with status staying draft; adaptive+valid rule publishes; handler 422 mapping.

## AC Coverage
- Adaptive exam with zero rules refused on publish: covered
- Same error/status as non-adaptive: covered
- Adaptive exam with valid rules still publishes: covered

Summary: Minimal, correct fix with adequate tests; PASS.

# Code Review: ISS-195 (analytics nits, auto_submitted consistency)

Result: PASS

Scope: staged changes: backend/internal/reports/repository.go, backend/internal/reports/repository_autosubmitted_test.go, docs/issue-reports/ISS-195-analytics-nits.md. No go test run (per instruction).

## Findings

- [Medium] repository.go GetAvgScoreByTrack (~line 396): the predicate changed from `= 'submitted'` to `IN ('submitted','auto_submitted','grading_pending')`. Adding auto_submitted is the intent. Adding grading_pending is a behavior change beyond the issue: those sessions may have partial scores, which can pull the 90-day track average down until grading finishes. It matches the other queries and the ISS-195 report documents it, so it is acceptable. A maintainer should confirm it is intended.
- [Medium] repository_autosubmitted_test.go: the tests only assert SQL text through a fake driver. They do not check semantics against Postgres. The issue report already flags this as needs-live-db. Acceptable.
- [Low] repository_autosubmitted_test.go ~line 15: the `var (...)` block has extra alignment spaces, so it is not gofmt-clean. `gofmt -l` also lists most other files in the package, which suggests CRLF noise rather than real drift. Run gofmt on the new file anyway.
- [Low] The test uses `_, _ =` to discard errors from the repository calls. This is fine for SQL-capture tests. The `require.NotEmpty(f.queries)` check guards against a silent no-op.

## Checklist notes

- SQL: parameterized, with only literal status lists changed. The `@SCOPE@` deptscope placeholders are untouched, and the test asserts they are expanded.
- No remaining legacy `('submitted','grading_pending')` or `= 'submitted'` completed predicates in repository.go. A grep shows every status list in the file now includes auto_submitted, including the previously updated PR #194 queries.
- Enum values match migrations 014/015 (`in_progress`, `submitted`, `auto_submitted`, `grading_pending`). No migration is needed, and no existing migration was edited. No tenant column was added, as required.
- No secrets, no `os.Getenv`, no handler or service changes, and no new endpoints. Audit is not applicable because there are no state changes.
- The test helpers `newFakeDB`, `fakeDB.queue`, `fakeDB.queries`, `mockRepo` and `getCompletionFn` all exist in the package (fakedb_test.go, service_test.go).

## AC Coverage (ISS-195)

- Item 1 (completed-definition consistency): covered. All 7 queries are updated and there is a regression test for each.
- Item 2 (`graded` in doc): handled as a documentation finding left to the BA. It is not a code change.
- Item 3 (tenant filter): not applicable, and the rationale is sound.
- Item 4 (unknown user): already done in PR #182, with existing tests.

Summary: The change is minimal, correct and consistent across all completed-session predicates, with no Critical or High findings.

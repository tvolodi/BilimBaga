# Code Review: ISS-124 staticcheck cleanup (swarm/124-staticcheck)

Result: PASS

## Scope
`git diff origin/main...HEAD -- backend`: 23 files, +31/-118. Behaviour-preserving U1000 (dead code) and S1016 (struct conversion) cleanup.

## Verification
- Removed identifiers: `okHandler` (auth test), `ptrTime/ptrStr/ptrFloat` (portal test), `retryAfterSeconds` (ratelimit), `queryResult` (reports), `parseTagIDs` (sessions), `decodeResponse` (tenant test), `mockRepo.tokens` (users test), `qWithType.typ` (sessions). Repo-wide grep (incl. tests) shows no remaining references. The `okHandler` matches in ratelimit/rbac are separate package-local declarations.
- S1016 conversions: all compile (Go requires identical field names/types/order, tags ignored), so no field is dropped or reordered. Per-field copies were already total. `qWithType.typ` was never assigned (always ""), so the conversion from `resolvedWithRules` elements is equivalent.
- fakeConnector -> fakeConn conversions (6 test files): both are `struct{ f *fakeDB }`. Equivalent.
- No db/json-tag-carrying field removed; only the unexported local `qWithType.typ` and the test mock field `tokens`.
- No unrelated changes. Only 23 backend files changed.
- CRLF preserved in all changed files (CRLF line count equals total line count; diffs are minimal, no whole-file rewrites).
- `go build ./...`, `go vet ./...`, `go test ./...` all pass; `staticcheck ./...` reports nothing.

## Findings
- Critical: none
- High: none
- Medium/Low: none

Note: `docs/issue-reports/ISS-124-staticcheck-cleanup.md` is untracked in the worktree (not part of the backend diff); remember to commit it with the release.

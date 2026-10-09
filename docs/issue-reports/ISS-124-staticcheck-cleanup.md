# ISS-124 — staticcheck findings cleanup

Issue: #124. Branch: `swarm/124-staticcheck`. No behaviour change; no SQL, migration or config touched.

## Before / after (staticcheck ./... in backend/)

| Check | Before | After |
|-------|--------|-------|
| S1016 (struct literal -> conversion) | 30 | 0 |
| U1000 (unused code) | 10 | 0 |
| Total | 40 | 0 |

(One additional S1016 in `sessions/service.go` surfaced after removing the dead `typ` field; fixed too.)

## Per package

| Package | Check | Before | After | Notes |
|---------|-------|--------|-------|-------|
| ai | S1016 | 1 | 0 | test fake connector |
| audit | S1016 | 1 | 0 | test fake connector |
| auth | U1000 | 1 | 0 | unused test var `okHandler` |
| db | S1016 | 1 | 0 | test fake connector |
| email | S1016 | 4 (+1 test) | 0 | repository.go row to DTO conversions |
| exams | S1016 | 6 | 0 | handler request to input conversions |
| portal | S1016 / U1000 | 1 / 3 | 0 | test helpers `ptrTime/ptrStr/ptrFloat` removed |
| questions | S1016 | 10 | 0 | handlers + repositories |
| ratelimit | U1000 | 1 | 0 | const `retryAfterSeconds` removed |
| rbac | S1016 | 1 | 0 | test fake connector |
| reports | S1016 / U1000 | 2 / 1 | 0 | unused type `queryResult` removed |
| sessions | S1016 / U1000 | 3 / 2 | 0 | unused `parseTagIDs` and local field `typ` removed |
| tenant | U1000 | 1 | 0 | unused test helper `decodeResponse` |
| users | U1000 | 1 | 0 | unused `mockRepo.tokens` (test only) |

## Dead-code verification

Every removed identifier was grepped repo-wide (including `_test.go`); none is referenced, none carries `db:`/`json:` tags, none is used by reflection or build tags. The removed `typ` is a field of a function-local struct without tags. `mockRepo.tokens` is a test mock field with no tag.

## Skipped / not fixed

- Initially planned skips for PR #104 files (auth recovery_*, users, email service/templates, migration 031): PR #104 and #116 were merged to main before the first push; after rebase no open PR remained, so the one finding in `users/service_test.go` was fixed. No finding was left unfixed.
- No `staticcheck.conf` added: the noise was small (40 findings) and all fixable.
- Migrations untouched. No SQL changed (schemaguard green).

## Verification

`go build ./...`, `go vet ./...`, `go test -p 2 ./...` all green after rebase on origin/main; `staticcheck ./...` exits 0 with no findings.

# ISS-233: CI tests workflow

Added `.github/workflows/tests.yml` (jobs `backend-tests`, `frontend-tests`) and a README "Continuous integration" section.

- Backend: go vet, staticcheck 2025.1.1 (pin may need a bump if the run complains), `go test -p 2 ./...` with postgres:16 service; `TEST_DATABASE_URL` is read by `backend/internal/db/unique_email_index_test.go`.
- Frontend: tsc, lint, check:i18n, vitest (CI=true). Playwright e2e excluded.
- No tests were run locally (host memory); the first CI run decides.

## First run results

(pending)
Runs 37920845696 (node 20) and 37921104964 (node 22, go test forced to run).

backend-tests: go vet OK; go test -p 2 ./... OK (with postgres service); staticcheck FAILS:
- backend/internal/users/privilege_escalation_test.go:311:2 `svc` assigned and never used (SA4006) in TestD1_CustomRoleWithSensitivePermsNotAssignable. Real lint finding in test code on main (from ISS-217/FR-BB117 work); not a workflow problem. Fix: use `_` for svc or use it. Filed as follow-up, not fixed in this PR.

frontend-tests: tsc, lint, check:i18n OK. vitest: 13 failed / 608 passed, all `TypeError: object.stream is not a function` at `new Response(new Blob([...]))` in the tests' blobResponse helpers:
- src/api/download.test.ts (8 tests), src/components/results/ResultActions.test.tsx (1), src/components/analytics/__tests__/ExportCSVButton.test.tsx (4).
Same on Node 20 and 22, so not the Node version. Cause: jsdom's Blob (jsdom ^29) lacks `.stream()`, which Node's undici Response needs; probably resolved differently on dev machines (different node/jsdom/undici combination or a polyfill). Test/environment issue, pre-existing on main (never ran in CI); fix belongs in test code (build the Response from a string/ArrayBuffer, or use Node's Blob) or vitest setup.

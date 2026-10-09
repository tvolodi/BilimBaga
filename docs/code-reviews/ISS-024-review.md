# Code Review: ISS-024 (backend/internal/ai test coverage)

Verdict: APPROVE (PASS)

Scope: staged tests-only changes in backend/internal/ai (client_test.go, handler_insights_loyalty_test.go, repository_more_test.go, service_branches_test.go) and docs/issue-reports/ISS-024-ai-package-test-coverage.md. No production code changed.

## Verification run
- `go test -p 2 -count=3 -race ./internal/ai/`: ok, no flakes or races.
- `go vet ./internal/ai/`: clean.
- Coverage: 94.5% (up from 53.6%).

## Checks
- Network: client tests use an httptest server and a RoundTripper that rewrites the host. Nothing reaches api.anthropic.com. Repository tests use the in-package fakedb driver, so no real DB.
- Determinism: no sleeps, no random values, no env reads, no t.Parallel shared state. The server is closed via t.Cleanup.
- Time windows: the stale-cache test uses `time.Now().Add(-25h)` against a 24h TTL, which leaves a 1h margin and is not flaky. Other `time.Now()` uses are plain fixture values.
- Assertions: the success path checks the request body, headers, path and method, and the response parsing. Error paths check wrapping, status mapping and the `{data,error}` envelope.
- Secrets: only the dummy key "test-key".

## Findings
- [Low] client_test.go: `url.Parse` error is ignored in newTestClient (`u, _ :=`). Harmless since the input is httptest's URL.
- [Low] client_test.go: `rewriteTransport` uses http.DefaultTransport, so a keep-alive connection might linger. srv.Close handles it, so this is not a problem.
- [Low] The issue doc says "go test -p 2 ./... all green". I only re-ran the ai package. It passes.

No Critical or High findings. Production code is untouched, so the migration, audit and RBAC checks do not apply.

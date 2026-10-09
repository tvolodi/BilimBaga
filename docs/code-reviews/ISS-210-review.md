# Code Review: ISS-210 results CSV export row cap

Verdict: **APPROVED** (one Medium and two Low non-blocking findings)

Scope: uncommitted diff on `swarm/210-csv-export-cap` (service, handler, config, main wiring, .env.example, router_test, config test) plus the untracked `export_cap_test.go` and `ISS-210-csv-export-cap.md`.

Verification: `go vet` on reports, config, router and cmd is clean. `go test -p 2` on reports, config and router passes.

## Checklist results
- Secrets, SQL, `os.Getenv`: pass. `EXPORT_MAX_ROWS` is read once in `config.Load` through `getEnvInt`, validated >= 1, and passed to the service through a functional option. Handlers do not read env vars.
- Layering: pass. The cap lives in the service. The handler only maps the sentinel error with `errors.Is`, so wrapped errors are also caught (the handler test wraps it).
- Error contract: pass. 422 `EXPORT_TOO_LARGE` uses the `{data:null,error:{code,message}}` shape through `api.WriteError`. The buffered writer guarantees no CSV headers or partial body are sent on error, and the test asserts this.
- Cap semantics: pass. The check runs before the scan, once `len(sessions) >= max`. Exactly N rows are allowed and N+1 fails. Both boundaries are tested. `defer rows.Close()` still runs on early return.
- Backward compatibility: pass. `NewService(repo, opts...)` is variadic, so existing callers and tests compile. `WithMaxExportRows(<=0)` falls back to the default.
- Config: pass. `.env.example` is documented, and the test covers default, override, and the rejected values 0, -1 and "abc".
- G2: pass. The non-UUID exam id route is added to the router 404 table test.
- Stale comments: fixed in `StreamExamResultsCSV` and `StreamUserRecordCSV`.

## Findings
1. **Medium (non-blocking), misleading user message.** `handler.go` tells the user to "narrow the export (e.g. filter by date range)". `GET /admin/exams/{id}/results/export` has no date, department or other filter. Only the dashboard report takes `from` and `to`. The hint cannot be acted on. Suggest rewording to something like "too many results to export in one file (limit N)", or open a follow-up to add a filter. The test asserts `Contains "date"`, so update it together with the message.
2. **Low, residual memory profile.** The cap bounds memory at about 200k session rows plus the handler's `csvBuffer` copy, which is large but finite. This matches the issue scope. The default could be reconsidered in a later issue if memory on QA is tight.
3. **Low, stale docs.** `docs/issue-reports/FR-BB54-INNER-REPORT.md` still says "CSV streamed directly, no buffering". It is a historical report, so this is optional. Consider noting the cap in the FR-BB54 AC-8 requirement doc.
4. **Info.** `ISS-210-csv-export-cap.md` lists "Tests: go test ./... all pass". I re-ran only the three affected packages, and they pass. There is no migration or frontend change. The frontend does not specifically handle `EXPORT_TOO_LARGE`, so it falls through to its generic error path. Out of scope here, but a candidate follow-up.

No Critical or High issues. No source files were edited by the reviewer.

## Review follow-up
Finding 1 (message promised a date filter that does not exist) fixed: 422 message now states the row limit and EXPORT_MAX_ROWS.

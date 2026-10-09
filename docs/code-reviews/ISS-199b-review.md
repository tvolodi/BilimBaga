# Code Review: ISS-199b (ragged CSV export rows)

Result: **PASS** (static read-only review; no tests, builds or vet were run because the host memory is low)

Scope: backend/internal/questions/{import_export_handler.go, import_export_service.go, model.go, csv_injection_test.go, import_export_ragged_test.go}; issue report docs/issue-reports/ISS-199b-ragged-export-rows.md.

## Goal check
| Goal | Status |
|------|--------|
| Header = union of locales (translations and option translations), sorted, plus max option count over all rows | Met (`csvExportScan.add/columns`, `header()`) |
| Every row padded to header width | Met. `record()` always emits 4 + 2*L + maxOpts*L + 2 cells. Missing locale, option or explanation gives an empty cell. |
| Formula guard kept | Met. `api.CSVSafeRecord(rec)` is still applied to the whole record. Padding is empty so it is never prefixed, and the test asserts this. |
| Importer `FieldsPerRecord = -1` | Met |
| Extra non-empty cell is a row error | Met. `ParseErrors` is filled in `parseCSVImport` and prepended to `errs` in `ValidateAndImport`. Extra empty cells are ignored. Short rows are already handled by `get()`. |
| Empty export behaviour unchanged | Met (`scan.rows > 0` guard) |

## Findings

### Critical
None.

### High
None.

### Medium
- import_export_handler.go ~L185-195: the second `StreamExport` runs after the first with no shared snapshot. Rows that change between passes are handled safely, because `record()` drops anything outside the scanned layout and so never misaligns. The cost is that data written in that window can be silently omitted from the export. This is acceptable given the documented trade-off. Suggestion: if exact consistency is ever needed, run both passes in one read-only repeatable-read transaction inside the service.
- import_export_handler.go ~L185-195: a failure in the second pass, after the header and some rows are written, leaves a truncated 200 CSV. The audit event is skipped, but the client cannot tell. This is the same as the prior behaviour and not a regression. Suggestion: log it.
- Tests were not executed in this review, and the author reports only a targeted run with no full suite. CI must run the full `go test ./...`. The static read found nothing suspicious. I traced the test indexes: `recs[3]` is the shorttext row, and "column 10" matches a 9-column header with a 10-cell record. The helpers `exportCSVBytes` and `importDryRun` exist in import_csv_header_test.go.

### Low
- model.go / import_export_handler.go: `ParseErrors` is only populated for CSV. JSON import does not use it, which is fine. The field comment could say it is CSV-only for now.
- `record()`: the `n < c.maxOpts` guard on correct indices can only be false if rows change between passes. It is harmless, defensive logic.
- Locales found only in option translations create `stem_<loc>` and `explanation_<loc>` columns that are empty for every row. This is harmless, and the importer skips blank stems (import_csv_header.go buildRow).
- The two passes double the read load on large exports, with memory still O(1). The report explains this. Consider noting it in the architecture docs.
- No test covers the `Export` handler end to end (`Accept: text/csv`) with a stubbed service. The tests go through `exportCSVBytes`, which I assume wraps the handler path, and the rectangular and round-trip assertions are strong.

## Checklist notes (Go)
- No secrets, SQL, `os.Getenv` or new routes. No new handler SQL (it reuses `StreamExport`). No migration needed.
- Errors are wrapped or propagated. `cw.Write` errors are now checked, where before they were ignored (an improvement).
- The removed `buildCSVHeader`, `rowToCSV` and `sortedLocales` have no remaining references (grep clean). The `sort` and `strconv` imports are both used.
- No debug output. Naming is conventional.

## AC coverage (from the issue report)
- Export is rectangular for mixed shapes: covered ✓ (TestExport_MixedTypesAreRectangular_AndRoundTrip, strict FieldsPerRecord=0)
- Export then import round-trips with 0 error rows and the same count: covered ✓
- Formula guard survives the round trip: covered ✓
- A ragged legacy file imports: covered ✓
- An extra non-empty cell is a row error rather than a file abort: covered ✓

Summary: the implementation matches the stated goal with no Critical or High findings. Confirm the full suite in CI.

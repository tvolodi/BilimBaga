---
id: ISS-199b
title: Questions CSV export is ragged (header from first row only); re-import fails with "wrong number of fields"
status: resolved
severity: high
layer: backend
module: questions
tags: [csv, export, import, FieldsPerRecord, "wrong number of fields", buildCSVHeader, rowToCSV]
created: 2026-10-09
resolved: 2026-10-09
recurrence_count: 1
related_issues: [ISS-199, ISS-191, ISS-173b]
regression_test: backend/internal/questions/import_export_ragged_test.go
---

## Symptom
GET /api/v1/questions/export (text/csv) then POST /questions/import: 400 ERR_INVALID_BODY
"CSV row N: parse error: record on line N: wrong number of fields" (15 of 19 rows on a mixed fixture). Issues #199 (reopened), #219.

## Root Cause
buildCSVHeader derived locales and option count from the FIRST row only, while rowToCSV wrote one cell per
actual option / locale of each row. Rows with other locales, fewer or more options, or none (short text) had a
different width. The importer used the default strict csv.Reader (FieldsPerRecord = 0).

## Fix Applied
- Exporter: two passes over the existing StreamExport (no new SQL, no migration). Pass 1 (csvExportScan) keeps
  only the union of locales (translations and option translations) and max option count; pass 2 writes the
  header (type, difficulty, category_path, default_locale, stem_<loc>.., explanation_<loc>.., option_N_<loc>..
  for N=1..max, correct, tags; locales sorted alphabetically) and every row padded with empty cells to that width.
  Formula guard applied to all text cells; padding stays empty. Empty export still writes nothing.
- Importer safety net: csv.Reader FieldsPerRecord = -1. Missing trailing cells read as empty (already via get()).
  Extra EMPTY cells beyond the header are ignored; an extra NON-EMPTY cell makes that row a row error
  (ImportRow.ParseErrors, reported in error_rows by the service) instead of aborting the file or dropping data.

## Files Changed
| File | Change |
|------|--------|
| backend/internal/questions/import_export_handler.go | two-pass CSV export, csvExportScan/csvExportColumns, lenient reader |
| backend/internal/questions/import_export_service.go | ParseErrors surfaced as row errors |
| backend/internal/questions/model.go | ImportRow.ParseErrors |
| backend/internal/questions/csv_injection_test.go | adapt to new helper |
| backend/internal/questions/import_export_ragged_test.go | new regression tests |

## Regression Test
import_export_ragged_test.go: mixed-type export (2-option, true/false, short text, 5-option Likert, multi-correct,
en/ru/kk with missing locales) is strictly rectangular, round-trips via dry run with 0 error rows and same count,
guard cells (= + - @) survive; ragged legacy file imports; extra non-empty cell is a row error.

## Resolution Results
- Tests: targeted `go test -p 1 ./internal/questions/ -run 'Export|Import|CSV|Csv|Ragged|RowToCSV|Normalise'` ok; go vet ./internal/questions/ clean
- Full suite not run (host memory alert); left to GitHub CI
- Migration applied: no
- Build clean: package vet only; no live export/import run

## Recurrence Log
| Date | Trigger | Action Taken |
|------|---------|--------------|
| 2026-10-09 | UAT on #216 merge showed ragged rows | This fix |

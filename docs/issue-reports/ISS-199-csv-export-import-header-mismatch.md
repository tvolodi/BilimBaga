---
id: ISS-199
title: Questions CSV export cannot be re-imported (stem_<locale>/option_N_<locale> vs required 'stem')
status: resolved
severity: medium
layer: backend
module: questions
tags: [parseCSVImport, buildCSVHeader, 'CSV: missing required column "stem"', stem_en, option_1_en, CSVUnsafe, ISS-191]
created: 2026-10-09
resolved: 2026-10-09
recurrence_count: 1
related_issues: [ISS-191, ISS-173]
regression_test: backend/internal/questions/import_csv_header_test.go
---

## Symptom
POST /api/v1/questions/import (also ?dry_run=true) on a file produced by GET /api/v1/questions/export answers
400 `CSV: missing required column "stem"`. Export header: `type,difficulty,category_path,default_locale,stem_en,explanation_en,option_1_en,...,correct,tags`.

## Root Cause
`parseCSVImport` only knew the legacy single-locale columns (`stem`, `explanation`, `option_N`); the exporter writes per-locale columns.
The ISS-191 formula-guard reversal (`api.CSVUnsafe`) already ran on every cell read, so it was not the blocker.

## Fix Applied
New pure function `normaliseCSVHeader` (import_csv_header.go) classifies header cells (case/space/BOM tolerant) into
stem / explanation / option_N columns per locale, with `""` = legacy column. `csvLayout.buildRow` maps a record to an `ImportRow`:
- legacy columns bind to `default_locale`; an explicit `<name>_<locale>` cell for the same locale wins when non-blank;
- every locale in the header becomes a translation (the import model already supports per-locale translations); a non-default locale with a blank stem creates no translation;
- blank non-default-locale option text is skipped; blank/missing default-locale option text stays a row error from the service (#173), unchanged;
- missing default-locale stem yields the existing row error `stem: required for default_locale "xx"`;
- header errors list ALL missing required columns (`type`, `difficulty`, `category_path`, `default_locale`, and `stem` or `stem_<locale>`).
Formula guard: `CSVUnsafe` is applied in the single cell getter, so it covers all new columns; `'Tis` style cells (apostrophe + non `= + - @ TAB CR`) are untouched.
No SQL/migration/service change.

## Files Changed
| File | Change |
|------|--------|
| backend/internal/questions/import_csv_header.go | new: header normalisation + row mapping |
| backend/internal/questions/import_export_handler.go | parseCSVImport delegates to the above |
| backend/internal/questions/import_csv_header_test.go | new: table + round-trip tests |

## Regression Test
`import_csv_header_test.go`: header table tests; legacy layout; multi-locale; guard strip (new + legacy columns, legit apostrophe preserved);
export -> import dry_run round trip (= + - @ cells, 2 rows, 0 error rows, values equal); missing default-locale stem; blank default-locale option (#173); accurate missing-column messages.

## Resolution Results
- Tests: targeted `go test -p 1 ./internal/questions/ -run 'Header|ParseCSV|RoundTrip|ImportDryRun|ImportHandler|ExportHandler|Import_|CSV'` passed; `go vet ./internal/questions/` clean
- Full suite, `go build ./...`, staticcheck: not run locally (host memory alert); left to CI
- Migration applied: no
- Build clean: package-level vet only

## Known limitation (pre-existing, not changed)
The exporter derives the header (locales, max option count) from the FIRST row only; later rows with other locales/more options
are misaligned in the exported file. Out of scope for #199.

## Recurrence Log
| Date | Trigger | Action Taken |
|------|---------|-------------- |

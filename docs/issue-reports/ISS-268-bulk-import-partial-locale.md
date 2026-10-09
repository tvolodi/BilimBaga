# ISS-268 — Bulk import applies the partial-locale rule

Follow-up of #228 item 1 / PR #267. Rule: FR-BB24 AC-11 (BA ruling, #228).

## Decision
- Import (CSV/JSON, dry-run and commit): a non-default locale that is present (stem or any option
  text non-blank) with any blank option text is a per-row error in the existing error report
  (`error_rows[].errors`), naming the option index, e.g.
  `answer_options[1].translations.kk.text: ...`. On commit any row error aborts the batch (existing behaviour).
  A locale with no option text stays valid (untranslated, falls back at render). Short-text exempt.
- Export round-trip: the export fixture's partial `kk` row was an invalid state; the fixture now has
  full `kk` option text (test not weakened). A new test pins that exporting a genuinely partial legacy
  row and re-importing reports a row error (not silently accepted).
- Frontend editor did send stem-only locales with blank options (all locales' option text always sent as
  `''`). Fixed: `findPartialLocales` guards create, save-draft and autosave; shows `questionEditor.error.partialLocale`.

## Tests
- `import_service_test.go` TestImport_PartialLocale_RowError; `import_export_ragged_test.go` round-trip fixture + reimport test.
- vitest: `partialLocales.test.ts`, editor guard test in `QuestionEditorPage.test.tsx`.
- `go test -p 2 ./...`, `go vet ./...`, vitest (658), tsc, lint all green.

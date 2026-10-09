# Code Review: ISS-199 (CSV import accepts export layout)

Scope: staged diff; import_csv_header.go, import_csv_header_test.go, import_export_handler.go. Read-only review, nothing was built or run (host memory constraint).

Verdict: CHANGES REQUESTED (one medium finding, small fix)

## Checked and OK
- Backward compat: legacy `stem/explanation/option_N` bind to default_locale; header matching is a strict superset of the old (case/space/BOM tolerant). Required-column check is unchanged apart from `stem` also being satisfied by `stem_<loc>`.
- #173: default-locale blank option text is not filled in the parser. An option that has only non-default text is kept without a default-locale translation, so the service raises "option N must have non-empty text for the default locale". A fully blank option ends the list (legacy behaviour). Covered by a test.
- Formula guard: `api.CSVUnsafe` is applied in the single `get` closure, so it covers every text column (stem, explanation, option, tags, locale columns). It strips only `'` followed by a dangerous char, so a legitimate leading apostrophe is preserved. Tests cover this.
- Map iteration: `texts()` iterates a map, but the result is order-independent. Explicit locale wins over legacy when non-blank, a blank cell never overwrites, and legacy never overwrites a non-blank value. `translations` and `options` are maps/slices indexed by n, so output is deterministic. Tests use ElementsMatch.
- Huge `option_99999999` header: the loop stops at the first missing or blank option, and Atoi overflow is handled.

## Findings
1. MEDIUM, locale case mismatch. Header names are lower-cased (`stem_pt-BR` becomes key `pt-br`), but the `default_locale` cell value is not. For a regional locale, `default_locale=pt-BR` gives translations keyed `pt-br`, so the service reports "stem: required for default_locale pt-BR" and the export-to-import round trip breaks. The test `regional locale` locks in the lowercased key. Fix: preserve the original case of the locale suffix (lower-case only for prefix matching), or compare locales case-insensitively and normalise `default_locale` the same way. Add a round-trip test with a mixed-case locale. Seeded locales (en/ru/kk) are unaffected, so this is a latent bug.
2. LOW, a blank middle option silently truncates later options (`break`). This matches legacy behaviour but is now easier to hit with multi-locale exports. Optionally continue past a fully blank option, or report a row error.
3. LOW, an explanation or option present only for a locale with a blank stem is dropped silently (documented in the code comment). Acceptable.
4. INFO, out of scope: `buildCSVHeader(sample)` derives locale columns from the first export row only. Rows with other locales would lose data on export. Consider a separate issue.
5. Test quality: good coverage (table test for the header, legacy, multi-locale, guard, round trip through the real handlers, error naming). Gaps: no test for explicit-vs-legacy precedence when both are present, and no test for duplicate or case-variant locale columns. The tests were not run in this review; the implementer or CI must confirm they pass.

## Resolution of findings (Issue Resolution agent)
- Finding 1 (medium, locale case): fixed. Locale suffixes keep their original spelling; keywords stay case-insensitive; a column locale equal (EqualFold) to default_locale binds to the default_locale spelling. Tests added (TestParseCSVImport_LocaleCaseAndPrecedence, header table updated). Targeted tests pass.
- Findings 2-3: accepted (documented, legacy-compatible). Finding 4: pre-existing exporter header-from-first-row limitation, out of scope.

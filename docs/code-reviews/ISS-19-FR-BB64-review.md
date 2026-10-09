# Code Review: Issue #19 / FR-BB64 Security hardening (uncommitted changes)

Result: PASS

Verification: `go vet` and `go test` for questions, users and upload pass (GOTMPDIR/GOCACHE redirected to scratchpad because `C:\Temp` is missing). `npm audit --omit=dev --audit-level=high` reports 0 vulnerabilities. Full `npm audit --audit-level=high` still reports 7 (1 high, 3 critical, 3 moderate), all dev toolchain.

## Findings

- [Medium] Makefile:23 and docs/requirements/FR-BB64.Security-hardening.md AC-7: AC-7 states `npm audit --audit-level=high` reports zero high/critical and runs as a required CI step. The implementation narrows the blocking gate to `--omit=dev` and runs the full audit as informational (`continue-on-error: true`) in security.yml. This is a defensible, documented deviation (the remaining findings are in vitest 2.x dev tooling, and a fix needs `--force`/a major bump), but it is not what the AC says. Update the AC text or open a follow-up issue for the vitest upgrade, and do not tick AC-7 as written without that note.
- [Medium] backend/internal/questions/import_export_handler.go:23: `ParseMultipartForm(32<<20)` is unchanged, so a large body is still buffered or spilled to disk before the 10 MB check. The 413 is correct, but it is not a pre-parse cap. Optionally wrap `r.Body` in `http.MaxBytesReader` (about 11 MB). This is pre-existing and the users CSV handler behaves the same way.
- [Low] deploy/nginx.conf:4: HSTS trusts the client-supplied `X-Forwarded-Proto`. This is harmless (it only adds a header to the client's own response) and matches the requirement. If the TLS proxy always overwrites the header, no change is needed.
- [Low] backend/internal/questions/import_export_handler.go:~60: `ValidateCSVFile` uses `http.DetectContentType` on the first 512 bytes. A UTF-16 BOM file would be rejected with 415. This is acceptable and consistent with the users handler.
- [Low] .github/workflows/security.yml: actions are pinned to major tags rather than SHAs. This is acceptable. `permissions: contents: read` is set correctly. `make` is available on ubuntu-latest, and `go mod verify` runs against `backend/go.mod`.

Checked and clean:
- Import: size and type validation runs before parsing. The `LimitReader(MaxCSVBytes+1)` approach correctly triggers `ErrFileTooLarge` for oversize files. Errors use the envelope via `api.WriteError`. The 413 and 415 codes are correct. The parsers are fed from the validated bytes, so there is no TOCTOU. The `upload` import creates no cycle, and the build is clean.
- Tests: the oversize (413) and binary-disguised-as-CSV (415) cases are covered. The 2000-iteration `ValidateComplexity` test for `generateTempPassword` covers AC-8 with no production change.
- nginx: the `map` sits in the http context (the file is mounted at `conf.d/default.conf` in both the Dockerfile and docker-compose). An empty `$hsts` makes nginx omit the header, as the requirement specifies. No `location` block defines its own `add_header`, so the server-level headers are inherited.
- package-lock.json: lock-only change. package.json is untouched. `npm audit --omit=dev` is clean.

## AC Coverage
- AC-2: covered (HSTS via map). The tick is pending verification of the live header.
- AC-3: covered (questions Import: magic bytes, 413 over 10 MB, tests). Logo and users CSV were already covered.
- AC-7: covered with a deviation (see the Medium finding above). A CI workflow exists and `make security-check` is wired in.
- AC-8: covered (test added).
- AC-1, AC-4, AC-5, AC-6: unchanged and already satisfied. AC-1 is still missing ratelimit tests, which are out of scope for these changes.

Summary: No Critical or High findings; the one notable item is the AC-7 audit-scope deviation, which needs documenting.

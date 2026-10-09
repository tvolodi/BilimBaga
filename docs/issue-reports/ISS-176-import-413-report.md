# ISS-176 round 2: oversize import upload

- Defect (UAT on #197): over-cap POST /questions/import reported as 400 ERR_INVALID_BODY / dropped connection.
- Handler-level repro on current code returned 413 for 10-30 MiB in-process; hardened anyway:
  `upload.ParseImportMultipart` now also consults the error recorded by the capped body reader
  (and `multipart.ErrMessageTooLarge`), so a wrapped/replaced MaxBytesError still maps to ErrFileTooLarge;
  `Import` maps a MaxBytesError from `FormFile` to 413 ERR_FILE_TOO_LARGE.
- Tests: over-HTTP (httptest.Server) 11/12/30 MiB -> 413 envelope; non-multipart -> 400.
- Unrelated blocker fixed: internal/auth tests hard-coded 2026-10-09 12:00 UTC and failed once real time passed token exp; now use `testClock`.
- `go test -p 2 ./...` green (run twice).

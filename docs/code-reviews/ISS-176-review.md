# Code Review ISS-176 (branch swarm/176-api-inconsistencies)

Result: PASS

## Focus checks
1. 400->422 regressions: none. Frontend clients (api/ai.ts, api/recovery.ts, ResetPasswordPage, ChangePasswordForm, errorMessages) branch on error.code, not HTTP status. e2e `account-recovery.spec.ts:116` and `password-reset.spec.ts:78` concern INVALID_TOKEN, which correctly stays 400. Only VALIDATION_ERROR sites changed (ai generate, forgot/reset handler decode + service). Tests updated.
2. Rate limiter: `disabled()` true only for "true"/"1"; unset = ON. AnswerSaveLimiter bypass only under that flag (noop). Router wiring unchanged (router.go:242). Tests cover block-after-60 and disabled pass-through. .env.example entry is commented out.
3. Upload cap: ParseImportMultipart wraps body in MaxBytesReader(10 MiB + 1 MiB) and ParseMultipartForm(10 MiB memory; excess spills to disk). Files of 10-11 MiB fit the body cap, are parsed, and reach ValidateCSVFile (users: ReadAll; questions: LimitReader MaxCSVBytes+1) which returns 413. errors.As(*http.MaxBytesError) works on Go 1.25 (wrapped errors propagate). 413 codes consistent per module (questions ERR_FILE_TOO_LARGE, users FILE_TOO_LARGE), same as pre-existing post-read codes. Non-multipart still 400. users/questions/upload/ratelimit/ai/auth tests pass.

## Findings
- Critical: none. High: none.
- [Low] upload/import_multipart_test.go: no test for a file in the 10-11 MiB window (should return nil and later 413 from ValidateCSVFile). Logic is correct by inspection; optional test.
- [Low] A multipart body of ~11 MiB with a 10.5 MiB file reaches the content check only after buffering to temp disk; acceptable.
- [Low] ai/handler.go: INVALID_BODY (malformed JSON) remains 400 while semantic validation is 422; consistent with convention (malformed vs invalid).

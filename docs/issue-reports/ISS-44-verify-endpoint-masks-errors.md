---
id: ISS-44
title: Verify endpoint returns 200 valid:false on internal errors (should be 5xx)
status: resolved
severity: medium
layer: backend
module: certificates
tags: [HandleVerifyCertificate, valid:false, INTERNAL_ERROR, PUBLIC_APP_URL, FR-BB48]
created: 2026-10-09
resolved: 2026-10-09
recurrence_count: 1
related_issues: []
regression_test: backend/internal/certificates/handler_test.go
---

## Symptom
GET /api/v1/verify/:code returned 200 `{valid:false}` for any service error (e.g. DB outage), so the public page showed "invalid certificate" instead of the AC-5 "temporarily unavailable" state.

## Root Cause
`HandleVerifyCertificate` swallowed every error from `svc.GetByVerificationCode` and mapped it to 200 valid:false. The service already maps `ErrNotFound` to valid:false, so only genuine internal errors reached the handler's error branch.

## Fix Applied
- Handler: `ErrNotFound` -> 200 `{valid:false}`; any other error -> `slog.Error` with the wrapped error, then 500 `INTERNAL_ERROR` with a generic message via `api.WriteError`.
- Config: `PublicAppURLDefaulted` set when `PUBLIC_APP_URL` is unset. `main.go` logs a startup warning (zerolog) when it is. Config has no environment marker (no APP_ENV), so the warning fires whenever the default is used.
- Frontend: no change. `VerifyCertificatePage` + tests already map HTTP 500 / network failure to "Verification temporarily unavailable" with Retry.
- No SQL touched; schemaguard test remains green.

## Files Changed
| File | Change |
|------|--------|
| backend/internal/certificates/handler.go | 500 + logging for internal errors |
| backend/internal/certificates/handler_test.go | 500 logged/no-leak test, ErrNotFound test |
| backend/internal/config/config.go | PublicAppURLDefaulted |
| backend/internal/config/config_test.go | asserts for defaulted flag |
| backend/cmd/api/main.go | startup warning |

## Regression Test
handler_test.go: TestHandleVerifyCertificate_500_OnInternalError_LoggedNoLeak, _200_ValidFalse_OnErrNotFound (plus existing valid true / unknown code).

## Resolution Results
- Tests: `go test -p 2 ./...` all pass; `go vet ./...` clean
- Migration applied: no
- Build clean: yes

## Recurrence Log
| Date | Trigger | Action Taken |
|------|---------|--------------|

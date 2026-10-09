---
id: ISS-176
title: API inconsistencies I-1/I-15, I-10, I-11 (VALIDATION_ERROR status, import parse cap, DISABLE_RATE_LIMIT)
status: resolved
severity: medium
layer: backend
module: auth
tags: [VALIDATION_ERROR, DISABLE_RATE_LIMIT, AnswerSaveLimiter, ParseMultipartForm, MaxBytesReader]
created: 2026-10-09
resolved: 2026-10-09
recurrence_count: 1
related_issues: []
regression_test: backend/internal/upload/import_multipart_test.go
---

## Symptom
- VALIDATION_ERROR returned 400 by auth recovery and ai handlers, 422 elsewhere.
- Questions import parsed up to 32 MiB although the content cap is 10 MiB; users import was uncapped on body.
- DISABLE_RATE_LIMIT did not disable AnswerSaveLimiter (blocks FR-BB65 k6 plan #32).

## Root Cause
Per-handler ad hoc status codes; ParseMultipartForm(maxMemory) is not a body cap; AnswerSaveLimiter lacked the disabled() check.

## Fix Applied
- ratelimit: AnswerSaveLimiter honours DISABLE_RATE_LIMIT; .env.example documents it as test-only (default ON).
- upload.ParseImportMultipart: http.MaxBytesReader at 10 MiB + 1 MiB overhead, ParseMultipartForm memory 10 MiB; over cap returns 413 (ERR_FILE_TOO_LARGE / FILE_TOO_LARGE, same codes as the existing content check). Used by questions and users import.
- VALIDATION_ERROR is 422 in auth/recovery_handler.go, auth/recovery_service.go, ai/handler.go (api/uuid.go and users already 422). I-15 (wrong password 401 vs 400) is a different code and was not changed.

## Files Changed
| File | Change |
|------|--------|
| backend/internal/ratelimit/middleware.go(+_test) | answer-save bypass |
| backend/.env.example | document DISABLE_RATE_LIMIT |
| backend/internal/upload/validate.go, import_multipart_test.go | capped parse helper |
| backend/internal/questions/import_export_handler.go, users/handler.go (+tests) | use helper |
| backend/internal/auth/recovery_*.go, ai/handler.go (+tests) | 422 |
| frontend/src/pages/auth/recovery.test.tsx | mock status 422 |
| docs/requirements/api-conventions.md | sections 3, 8, 13 updated |

## Regression Test
upload/import_multipart_test.go, ratelimit answer cases, questions/users over-cap 413 tests, recovery/ai 422 tests.

## Resolution Results
- Tests: go test -p 1 ./... all pass; vitest src/pages/auth 12 passed
- Migration applied: no
- Build clean: yes

## Recurrence Log
| Date | Trigger | Action Taken |
|------|---------|-------------- |

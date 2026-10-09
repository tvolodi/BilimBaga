---
id: ISS-125
title: Empty SMTP_FROM makes every outgoing mail fail with 550 Invalid syntax in MAIL command
status: resolved
severity: medium
layer: backend
module: email
tags: [SMTP_FROM, extractEmailAddress, 550, "Invalid syntax in MAIL command"]
created: 2026-10-09
resolved: 2026-10-09
recurrence_count: 1
related_issues: []
regression_test: backend/internal/config/config_test.go
---

## Symptom
With SMTP_FROM unset every send fails: `smtp sendmail: 550 "Invalid syntax in MAIL command"`; forgot-password still returns neutral 200.

## Root Cause
`config.SMTPFrom` defaulted to ""; `extractEmailAddress("")` failed and `send`/`TestSend` fell back to the raw (empty) string as envelope sender. No startup validation.

## Fix Applied
- `config.validate`: when SMTP_HOST set, empty SMTP_FROM -> `DefaultSMTPFrom` (`BilimBaga <noreply@localhost>`, `SMTPFromDefaulted=true`, warning logged in main.go); unparseable -> startup error.
- `email.extractEmailAddress` rejects empty; `send` and `TestSend` return the error instead of falling back.
- `.env.example` and FR-BB61 doc updated.

## Files Changed
| File | Change |
|------|--------|
| backend/internal/config/config.go | validation + default |
| backend/internal/email/service.go | explicit error |
| backend/cmd/api/main.go | startup warning |
| backend/.env.example, docs/requirements/FR-BB61.Email-notifications.md | docs |

## Regression Test
config_test.go TestLoad_SMTPFrom; email/service_more_test.go TestSend_InvalidFromReturnsError (send + TestSend).

## Resolution Results
- Tests: all backend packages pass; go build/vet clean
- Migration applied: no
- Build clean: yes

## Recurrence Log
| Date | Trigger | Action Taken |
|------|---------|-------------- |

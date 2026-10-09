---
id: ISS-061
title: Overdue-reminder "Send Reminder" is a no-op that reports success
status: resolved
severity: medium
layer: backend
module: users
tags: [RemindEmployee, exam_reminders, overdue_reminder, FR-BB510, OverdueEmployee.exam_id]
created: 2026-10-09
resolved: 2026-10-09
recurrence_count: 1
related_issues: []
regression_test: backend/internal/users/reminder_test.go
---

## Symptom
`POST /api/v1/admin/users/{userId}/remind` returned `200 {data:null,error:null}` without doing anything;
the UI toasted "Reminder sent successfully." No email was sent, and `overdue_employees[]` had no `exam_id`
so the frontend posted an empty one.

## Root Cause
`users.Handler.RemindEmployee` was a deliberate FR-BB56 stub that ignored the request. `reports.OverdueEmployee`
did not select/expose `exam_id`. The notification system (FR-BB61/#55) it waited for now exists.

## Fix Applied
Implemented FR-BB510: migration 036 `exam_reminders`; `reports.OverdueEmployee.exam_id`;
`email.EmailService.SendOverdueReminder` + `overdue_reminder` template (en/ru/kk) + repo `GetOverdueReminderData`;
`users.Service.RemindEmployee` (404/409/429/502, 24h rate limit, row inserted only after successful send);
handler audits `users.remind` (metadata exam_id/exam_title, no email body/address); frontend OverdueTable maps
error codes to i18n messages and keeps sent rows disabled.

## Files Changed
| File | Change |
|------|--------|
| backend/migrations/036_exam_reminders.{up,down}.sql | new table + index |
| backend/internal/users/reminder.go | service, handler, repository methods |
| backend/internal/users/{service,repository,handler}.go | wiring, stub removed |
| backend/internal/email/{service,repository}.go, locales/*.json | overdue_reminder |
| backend/internal/reports/{model,repository}.go | exam_id |
| frontend/src/api/dashboard.ts, components/dashboard/OverdueTable.tsx, locales/*.json | UI |

## Regression Test
`backend/internal/users/reminder_test.go`, `backend/internal/email/overdue_reminder_test.go`,
`backend/internal/reports/overdue_exam_id_test.go`, `frontend/src/components/dashboard/OverdueTable.test.tsx`.

## Resolution Results
- Tests: backend `go test -p 2 ./...` all pass; `go vet ./...` clean; frontend vitest 630 passed; `tsc --noEmit` clean
- Migration applied: no (SQL verified only via schemaguard and fake drivers; needs live DB, label needs-live-db)
- Build clean: yes
- Not done: live E2E (Mailhog) test from AC-9; department-scope check on the remind endpoint (not in spec).

## Recurrence Log
| Date | Trigger | Action Taken |
|------|---------|-------------- |

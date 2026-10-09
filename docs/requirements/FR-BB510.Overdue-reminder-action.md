# FR-BB510 — Overdue Employee Reminder: Real "Send Reminder" Action

## Metadata
| Field | Value |
|-------|-------|
| ID | FR-BB510 |
| Phase | 5 — Analytics & Reporting (gap closure; completes roadmap 5.6 "Overdue employees table with quick-assign-reminder action") |
| Priority | 2 |
| Status | Validated |
| Depends On | FR-BB51, FR-BB56, FR-BB61, FR-BB19, FR-BB16, FR-BB62 |

## Description
Roadmap 5.6 requires the dashboard overdue-employees table to offer a quick reminder action. FR-BB56 AC-7 shipped it as a deliberate stub: `POST /api/v1/admin/users/{userId}/remind` (`users.Handler.RemindEmployee`) ignores the request and always returns `200 { data: null, error: null }`, and the UI then toasts "Reminder sent successfully." No email is sent. Worse, `reports.OverdueEmployee` does not expose `exam_id`, so the frontend sends an empty `exam_id` for every row. The notification system the stub was waiting for now exists (FR-BB61), so the stub must become real: an admin clicking "Send Reminder" must cause the overdue employee to receive a localized email for that exam, the attempt must be audited and rate-limited, and failures must be reported honestly instead of as success.

## Acceptance Criteria
- [ ] AC-1: `reports.OverdueEmployee` (`GET /api/v1/admin/dashboard`, `overdue_employees[]`) additionally exposes `exam_id` (UUID v4); the repository query selects it. Existing fields are unchanged and the 20-row limit and ordering are preserved.
- [ ] AC-2: `POST /api/v1/admin/users/{userId}/remind` (permission `reports:read`, as today) accepts `{ "exam_id": "<uuid>" }`. A missing, empty or non-UUID `exam_id` returns 400 `VALIDATION_ERROR`; an unknown user returns 404 `NOT_FOUND`; an unknown exam returns 404 `NOT_FOUND`. The stub behaviour (200 with no work done) no longer exists.
- [ ] AC-3: The endpoint only sends when the user is genuinely a reminder target: the user is `active`, the exam has an assignment resolving to that user (user, department-tree or `all`, same resolution as `GetOverdueEmployees`), and the user has no passing session for the exam. Otherwise it returns 409 `NOT_OVERDUE` (not assigned / already passed) or 409 `USER_INACTIVE` and sends nothing.
- [ ] AC-4: On a valid target the server sends a new email template `overdue_reminder` (plain text + HTML multipart, subject/body keys in `backend/internal/email/locales/{en,ru,kk}.json`, rendered in the recipient's `preferred_locale` with tenant-default fallback) containing the exam title, the deadline (UTC, human formatted) and a portal link to the exam. Delivery is synchronous in this endpoint; success returns 200 `{ "data": { "sent_at": "<UTC ISO 8601>" }, "error": null }`. An SMTP failure returns 502 `EMAIL_SEND_FAILED` (no success toast possible). Every attempt, success or failure, is recorded in `email_log` via the existing `logAttempt` path.
- [ ] AC-5: Rate limiting: at most one reminder per (user, exam) per 24 hours, tracked in a new table `exam_reminders (id UUID PK, user_id UUID FK users, exam_id UUID FK exams, sent_by UUID FK users, sent_at TIMESTAMPTZ NOT NULL DEFAULT now())` with index `(user_id, exam_id, sent_at DESC)`, created by a new numbered migration with `.down.sql`. A second request inside the window returns 429 `REMINDER_RATE_LIMITED` with `retry_after_seconds` in `error.details` and sends nothing. A row is inserted only after a successful send.
- [ ] AC-6: Each successful send writes an `audit_log` entry `users.remind` (entity_type `user`, entity_id = target user id, metadata `{ exam_id, exam_title }`, actor = admin); the audit metadata never contains the email body. Rejected requests (409/429) write no entry.
- [ ] AC-7: Authorization: employee roles receive 403 (existing RBAC middleware); the response and logs never reveal the recipient's email address to the caller.
- [ ] AC-8: Frontend: `OverdueTable` sends the row's real `exam_id`; the Send Reminder button is disabled while the mutation is pending; success shows `dashboard.reminder_sent`; 429 shows the new `dashboard.reminder_rate_limited` message, 409 shows `dashboard.reminder_not_overdue`, any other failure shows `dashboard.reminder_error`; after success the button for that row stays disabled with a "Reminder sent" label for the rest of the session. All new strings exist in `en`, `ru`, `kk`.
- [ ] AC-9: Tests: backend `service_test.go` + `handler_test.go` cover validation (400), unknown user/exam (404), not-overdue and inactive (409), success path (email sent, `exam_reminders` row, audit entry), SMTP failure (502, no `exam_reminders` row), rate limit (429), RBAC 403; email `templates_test.go` covers `overdue_reminder` rendering in all three locales with no unresolved `{{`; frontend Vitest + RTL covers `OverdueTable` (real `exam_id` posted, pending/disabled, each error code mapped to its message); a live E2E asserts that clicking Send Reminder on an overdue row delivers a message to Mailhog.

## Technical Specification

### Database Schema
New migration (next free number after the highest in `backend/migrations/`, currently 030; never edit existing files):
```sql
CREATE TABLE IF NOT EXISTS exam_reminders (
    id        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id   UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    exam_id   UUID NOT NULL REFERENCES exams(id) ON DELETE CASCADE,
    sent_by   UUID NOT NULL REFERENCES users(id),
    sent_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_exam_reminders_user_exam ON exam_reminders (user_id, exam_id, sent_at DESC);
```
Down migration drops the index and table.

### API Contract
| Method | Path | Auth | Request | Success | Errors |
|--------|------|------|---------|---------|--------|
| POST | `/api/v1/admin/users/{userId}/remind` | `reports:read` | `{ "exam_id": "uuid" }` | 200 `{ "data": { "sent_at": "..." }, "error": null }` | 400 `VALIDATION_ERROR`, 403, 404 `NOT_FOUND`, 409 `NOT_OVERDUE` / `USER_INACTIVE`, 429 `REMINDER_RATE_LIMITED`, 502 `EMAIL_SEND_FAILED` |
| GET | `/api/v1/admin/dashboard` | `reports:read` | - | `overdue_employees[]` gains `exam_id` | unchanged |

All responses use the `{ data, error }` envelope; IDs UUID v4; timestamps UTC ISO 8601.

### Go Implementation Notes
- Package `users`: `Service.RemindEmployee(ctx, actorID, userID, examID)` holds all business rules (target check, rate limit, send, record, audit); `Handler.RemindEmployee` only binds, calls the service, and maps typed errors to status codes. The service depends on a small `Reminder` interface (`SendOverdueReminder(ctx, userID, examID string) (*ReminderResult, error)`) satisfied by `email.EmailService`, preserving the rule that `internal/email` imports no domain packages.
- Package `email`: add template `overdue_reminder` to `templates.go` and the locale files, plus a synchronous `SendOverdueReminder` (reusing `send` + `logAttempt`); a repository method fetches the reminder context (email, locale, exam title, effective deadline) with all SQL in `repository.go`.
- Package `users` repository: `IsOverdueTarget(ctx, userID, examID)` (department-tree resolution equivalent to `reports.GetOverdueEmployees`, implemented locally, no import of `reports`), `LastReminderAt`, `InsertReminder`.
- Package `reports`: add `ExamID string json:"exam_id"` to `OverdueEmployee` and select it.
- The portal link base comes from the typed `Config` (no `os.Getenv` in handlers).

### Frontend Implementation Notes
- Files: `src/api/dashboard.ts` (`useRemindEmployee` maps the error `code` to message keys; `OverdueEmployee.exam_id` becomes required), `src/components/dashboard/OverdueTable.tsx` (per-row pending/sent state keyed by `user_id + exam_id`).
- i18n keys under `dashboard.*`: `reminder_rate_limited`, `reminder_not_overdue`, `reminder_already_sent`; existing `send_reminder`, `reminder_sent`, `reminder_error` retained.
- Use the shared Bearer-aware API helper; no raw `fetch`.

## Out of Scope
- Bulk "remind all overdue" and scheduled escalation emails to managers.
- Reminder history UI (rows are queryable in `exam_reminders` and `audit_log`).
- Reminders for non-overdue (upcoming-deadline) assignments; those remain handled by the FR-BB61 daily scheduler.
- SMS/push channels.

## Notes
- Supersedes the stub clause in FR-BB56 AC-7 ("the endpoint must exist (stub if not yet implemented)") and the FR-BB56 note that the endpoint may return an empty 200.
- Evidence of the gap: `backend/internal/users/handler.go` `RemindEmployee` returns `{data:nil,error:nil}` unconditionally; `reports.OverdueEmployee` has no `exam_id`; `OverdueTable.tsx` posts `employee.exam_id ?? ''`.
- Severity is a UX-honesty/ops issue (false "sent" confirmation), not a security or data-loss defect.

# FR-BB61 — Email Notifications

## Metadata
| Field | Value |
|-------|-------|
| ID | FR-BB61 |
| Phase | 6 — Polish & Hardening |
| Priority | 1 |
| Status | implemented |
| Depends On | FR-BB33 |

## Description
Implements a transactional email service for the platform using Go's `net/smtp` package. Emails are sent asynchronously in background goroutines and cover all key lifecycle events: exam assignment, results, deadline reminders, and password resets. A background scheduler runs daily to send 48-hour deadline reminders. All email content is locale-aware, using the recipient's preferred locale.

## Scope

| Layer | Affected files / artefacts |
|---|---|
| Database | `migrations/022_email_log.up.sql`, `022_email_log.down.sql`, `023_users_preferred_locale.up.sql`, `023_users_preferred_locale.down.sql` |
| Backend package | `internal/email/` (service.go, repository.go, templates.go, scheduler.go, locales/*.json) |
| Config | `internal/config/config.go` — add SMTP + TZ fields; `.env.example` |
| API | `POST /api/v1/admin/notifications/test` — new endpoint in `internal/email/handler.go` |
| Router | `internal/router/router.go` — wire new route |
| Background | Deadline-reminder goroutine started from `cmd/api/main.go` |
| Integration | `internal/exams` service, `internal/sessions` service, `internal/auth` service — call `EmailService.Send` |
| Docker | `docker-compose.yml` — add `mailhog` service |

## Out of Scope
- Email open/click tracking
- Bounce and unsubscribe handling
- Inbox delivery guarantees (SPF, DKIM, DMARC)
- Token-based password reset links (planned for a future FR)
- Email log retention/purge job

## Acceptance Criteria
- [ ] AC-1: All five email templates (`new_exam_assigned`, `exam_passed`, `exam_failed`, `deadline_reminder`, `password_reset`) are implemented as multipart/alternative MIME messages containing both a plain-text part and an HTML part; a table-driven unit test covering all five template names verifies that every Go template placeholder (`{{.VariableName}}`) resolves to a non-empty string and that no `{{` remains in the rendered output.
- [ ] AC-2: Emails are dispatched in a background goroutine (fire-and-forget); errors are logged at error level using `slog.Error` and never returned to or observed by the calling HTTP handler.
- [ ] AC-3: SMTP connection is configured exclusively via environment variables (`SMTP_HOST`, `SMTP_PORT`, `SMTP_USER`, `SMTP_PASS`, `SMTP_TLS`, `SMTP_FROM`); no SMTP credentials appear in source code or embedded files.
- [ ] AC-4: Background deadline-reminder goroutine starts at API boot, runs daily at 08:00 in the configured `TENANT_TIMEZONE`; it queries `exam_assignments` (expanding `user`, `department`, and `all` assignment types to individual users) for deadlines in the `[now+24h, now+48h]` window and sends a `deadline_reminder` email only to users for whom no `exam_sessions` row with `passed = TRUE` exists for that exam.
- [ ] AC-5: Every sent or failed email attempt is recorded in the `email_log` table with `recipient_email`, `template`, `sent_at`, and `error` (NULL on success); a unit test verifies a row is written for both success and failure scenarios.
- [ ] AC-6: `POST /api/v1/admin/notifications/test` sends a test email to the authenticated super admin's email address and returns `200 { "data": { "sent_to": "<email>" }, "error": null }` on success, or `503 { "data": null, "error": { "code": "EMAIL_UNAVAILABLE", "message": "..." } }` when the SMTP connection fails.
- [ ] AC-7: Email body language is chosen by the following resolution order: (1) user `preferred_locale`, (2) tenant `default_locale` from `tenant_config` table, (3) `en` as the final hard fallback; a unit test exercises all three fallback levels in isolation.
- [ ] AC-8: Templates contain no hardcoded natural-language strings; all copy is loaded from embedded JSON locale files under the `email.*` key namespace.
- [ ] AC-9: All user-controlled data placed into SMTP headers (e.g. `Subject`, `To`) is sanitised to strip `\r` and `\n` characters before being written to the header, preventing SMTP header injection (CWE-93).

## Technical Specification

### Configuration / Infrastructure

**New environment variables** (add to `.env.example`):
```
SMTP_HOST=smtp.example.com
SMTP_PORT=587
SMTP_USER=noreply@example.com
SMTP_PASS=secret
SMTP_TLS=true          # true = STARTTLS, false = plain
SMTP_FROM="BilimBaga <noreply@example.com>"
TENANT_TIMEZONE=Asia/Almaty
```

**New migrations**:

`migrations/022_email_log.up.sql`:
```sql
CREATE TABLE email_log (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  recipient_email TEXT        NOT NULL,
  template        TEXT        NOT NULL,
  sent_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  error           TEXT
);

CREATE INDEX idx_email_log_sent_at ON email_log (sent_at);
```

`migrations/022_email_log.down.sql`:
```sql
DROP TABLE IF EXISTS email_log;
```

`migrations/023_users_preferred_locale.up.sql`:
```sql
ALTER TABLE users ADD COLUMN preferred_locale TEXT;
```

`migrations/023_users_preferred_locale.down.sql`:
```sql
ALTER TABLE users DROP COLUMN IF EXISTS preferred_locale;
```

### API Endpoints

#### POST /api/v1/admin/notifications/test
- **Auth**: super_admin role only
- **Request body**: none (uses authenticated admin's email)
- **Success response** `200`:
  ```json
  { "data": { "sent_to": "admin@example.com" }, "error": null }
  ```
- **Error response** `503`:
  ```json
  { "data": null, "error": { "code": "EMAIL_UNAVAILABLE", "message": "SMTP connection failed" } }
  ```

### Implementation Details

**Package structure**: `internal/email/`
- `service.go` — `EmailService` struct with `Send(to, tmplName string, data map[string]any, locale string)` method
- `repository.go` — `FetchDeadlineReminderTargets(ctx, db, from, to time.Time) ([]ReminderRow, error)` — all SQL confined here
- `templates.go` — template registry mapping template name → `text/template` (plain-text part) + `html/template` (HTML part) pairs; locale copy loaded from embedded JSON files; `html/template` is used for the HTML MIME part to ensure automatic HTML escaping of all data values
- `scheduler.go` — deadline reminder goroutine; calls `repository.FetchDeadlineReminderTargets`, delegates to `EmailService.Send`
- `handler.go` — `HandleTestNotification` HTTP handler
- `locales/en.json`, `locales/kk.json`, `locales/ru.json` — email copy keyed under `email.*` namespace; embedded via `//go:embed locales/*.json`

**Email service** (uses `log/slog`, consistent with project logger pattern):
```go
type EmailService struct {
    cfg    Config
    db     *sqlx.DB
    logger *slog.Logger
}

func (s *EmailService) Send(to, tmplName string, data map[string]any, locale string) {
    go func() {
        err := s.send(to, tmplName, data, locale)
        s.logAttempt(to, tmplName, err)
        if err != nil {
            s.logger.Error("email send failed", "template", tmplName, "to", to, "error", err)
        }
    }()
}
```

**SMTP header sanitisation** — required by AC-9:
- A `sanitiseHeader(s string) string` helper in `service.go` replaces any `\r` or `\n` characters with a space before the value is written to any SMTP header field (`Subject`, `To`, `From`).

**Backend locale files** (embedded JSON, keyed `email.<template>.<key>`):
- Location: `internal/email/locales/{en,kk,ru}.json`
- Format: flat JSON map, e.g. `{ "email.new_exam_assigned.subject": "New exam assigned: {{.ExamTitle}}" }`
- Loaded at `EmailService` construction via `embed.FS`; cached as `map[string]map[string]string` (outer key = locale code)
- Locale resolution order: (1) user `preferred_locale`, (2) tenant `default_locale` from `tenant_config` table (key `default_locale`), (3) `en` as final fallback

**Template data contracts**:
| Template | Required data keys |
|---|---|
| `new_exam_assigned` | `ExamTitle`, `Deadline` (ISO8601), `PortalLink` |
| `exam_passed` | `ExamTitle`, `ScorePct`, `CertDownloadLink` |
| `exam_failed` | `ExamTitle`, `ScorePct`, `PassingScorePct`, `AttemptsRemaining` |
| `deadline_reminder` | `ExamTitle`, `Deadline` (human-formatted in locale) |
| `password_reset` | `TempPassword` |

**Deadline reminder scheduler**:
- Accepts a `context.Context`; exits cleanly when the context is cancelled (graceful API shutdown).
- Uses `time.AfterFunc` to fire at the next 08:00 in `TENANT_TIMEZONE`; inside the callback it sends reminders then schedules the next `time.AfterFunc` for the following 08:00 — no long-lived ticker is used.
- All SQL is in `repository.FetchDeadlineReminderTargets`; scheduler contains no raw SQL.
- Expands all three assignment types (`user`, `department`, `all`) to individual users before querying.
- Repository query (`repository.go`):
  ```sql
  WITH assigned_users AS (
      SELECT ea.exam_id, ea.assignee_id AS user_id, ea.deadline
      FROM exam_assignments ea
      WHERE ea.assignee_type = 'user'
        AND ea.deadline BETWEEN $1 AND $2
      UNION
      SELECT ea.exam_id, u.id AS user_id, ea.deadline
      FROM exam_assignments ea
      JOIN users u ON u.department_id = ea.assignee_id
      WHERE ea.assignee_type = 'department'
        AND ea.deadline BETWEEN $1 AND $2
      UNION
      SELECT ea.exam_id, u.id AS user_id, ea.deadline
      FROM exam_assignments ea
      CROSS JOIN users u
      WHERE ea.assignee_type = 'all'
        AND ea.deadline BETWEEN $1 AND $2
  )
  SELECT au.user_id, au.deadline, u.email, COALESCE(u.preferred_locale, '') AS preferred_locale, e.title
  FROM assigned_users au
  JOIN users u ON u.id = au.user_id
  JOIN exams e ON e.id = au.exam_id
  WHERE NOT EXISTS (
      SELECT 1 FROM exam_sessions es
      WHERE es.exam_id = au.exam_id AND es.user_id = au.user_id AND es.passed = TRUE
  )
  ```
  (`$1` = `NOW() + INTERVAL '24 hours'`, `$2` = `NOW() + INTERVAL '48 hours'`)

**SMTP TLS**:
- `SMTP_TLS=true` → connect via `smtp.Dial`, then call `StartTLS` with `tls.Config{ServerName: host}`
- `SMTP_TLS=false` → plain `smtp.Dial` (for local dev mail catchers like MailHog)

**Integration points** — `EmailService.Send` called from:
- `exams` service (`internal/exams`): on assignment creation → `new_exam_assigned`
- `sessions` service (`internal/sessions`): after auto-grading on session submission → `exam_passed` or `exam_failed`; after manual grading completes → same
- `auth` service (`internal/auth`): on temp-password creation → `password_reset`
- Scheduler: daily → `deadline_reminder`

## Test Strategy
- **Unit — template rendering** (AC-1): table-driven test in `templates_test.go` iterating all five template names with minimal valid data maps; asserts both `text/plain` and `text/html` parts are present and no `{{` remains.
- **Unit — locale fallback** (AC-7): three sub-tests — (a) user locale present, (b) user locale missing → tenant default used, (c) both missing → `en` used.
- **Unit — email_log write** (AC-5): mock `db`; assert `INSERT INTO email_log` called for both success and error paths.
- **Unit — header sanitisation** (AC-9): pass values containing `\r\n` to `sanitiseHeader`; assert output contains no CRLF.
- **Integration — POST /admin/notifications/test** (AC-6): spin up httptest server with bad SMTP config; assert HTTP 503 response with `EMAIL_UNAVAILABLE` error code.
- **Unit — scheduler timing**: inject a fake clock; verify `time.AfterFunc` is called with the correct duration to the next 08:00.

## Notes
- MailHog is recommended for local development; add `mailhog` service to `docker-compose.yml` alongside `api`.
- The `email_log` table is append-only; no records are deleted. Retention policy can be added as a future maintenance job.
- HTML templates must not load external resources (images, fonts) to comply with the Content-Security-Policy in FR-BB64.
- If `AttemptsRemaining` is 0, the `exam_failed` template body should omit the retake section.
- **Intentional deviation from roadmap §6.1**: The roadmap specifies a "password reset link" email. This requirement instead sends a temporary cleartext password, consistent with the existing `force_password_change` mechanism in the `users` table. Token-based password reset links are deferred to a future FR. The `auth` service MUST set `force_password_change = true` when issuing a temporary password so the user is forced to change it on first login.

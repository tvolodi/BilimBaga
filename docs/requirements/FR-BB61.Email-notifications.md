# FR-BB61 — Email Notifications

## Metadata
| Field | Value |
|-------|-------|
| ID | FR-BB61 |
| Phase | 6 — Polish & Hardening |
| Priority | 1 |
| Status | Draft |
| Depends On | FR-BB33 |

## Description
Implements a transactional email service for the platform using Go's `net/smtp` package. Emails are sent asynchronously in background goroutines and cover all key lifecycle events: exam assignment, results, deadline reminders, and password resets. A background scheduler runs daily to send 48-hour deadline reminders. All email content is locale-aware, using the recipient's preferred locale.

## Acceptance Criteria
- [ ] AC-1: All five email templates (new_exam_assigned, exam_passed, exam_failed, deadline_reminder, password_reset) are implemented as multipart plain-text + HTML emails and render correctly with all required variables substituted.
- [ ] AC-2: Emails are dispatched in a background goroutine (fire-and-forget); errors are logged at error level and never bubble up to the calling HTTP handler.
- [ ] AC-3: SMTP connection is configured exclusively via environment variables (`SMTP_HOST`, `SMTP_PORT`, `SMTP_USER`, `SMTP_PASS`, `SMTP_TLS`, `SMTP_FROM`); no SMTP credentials appear in source code.
- [ ] AC-4: Background deadline-reminder goroutine starts at API boot, runs daily at 08:00 in the configured `TENANT_TIMEZONE`; it scans `exam_assignments` for deadlines in the [now+24h, now+48h] window and sends a reminder only to users who have not yet passed the exam.
- [ ] AC-5: Every sent or failed email attempt is recorded in the `email_log` table with recipient_email, template name, sent_at, and any error text.
- [ ] AC-6: `POST /api/v1/admin/notifications/test` sends a test email to the authenticated super admin's email address and returns 200 on success or 503 with error code `EMAIL_UNAVAILABLE` on SMTP failure.
- [ ] AC-7: Email body language matches the recipient user's `preferred_locale`; falls back to tenant `default_locale` if preferred_locale is unset or unsupported.
- [ ] AC-8: Templates contain no hardcoded natural-language strings; all copy is loaded from locale files keyed under `email.*` namespace.

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

**New migration** — `migrations/NNN_email_log.sql`:
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
- `service.go` — `EmailService` struct with `Send(ctx, to, template, data, locale)` method
- `templates.go` — template registry mapping template name → `text/template` + `html/template` pairs
- `scheduler.go` — deadline reminder goroutine

**Email service**:
```go
type EmailService struct {
    cfg    Config
    db     *sqlx.DB
    logger zerolog.Logger
}

func (s *EmailService) Send(to, template string, data map[string]any, locale string) {
    go func() {
        err := s.send(to, template, data, locale)
        s.logAttempt(to, template, err)
        if err != nil {
            s.logger.Error().Err(err).Str("template", template).Str("to", to).Msg("email send failed")
        }
    }()
}
```

**Template data contracts**:
| Template | Required data keys |
|---|---|
| `new_exam_assigned` | `exam_title`, `deadline` (ISO8601), `portal_link` |
| `exam_passed` | `exam_title`, `score_pct`, `cert_download_link` |
| `exam_failed` | `exam_title`, `score_pct`, `passing_score_pct`, `attempts_remaining` |
| `deadline_reminder` | `exam_title`, `deadline` (human-formatted in locale) |
| `password_reset` | `temp_password` |

**Deadline reminder scheduler**:
- Uses `time.AfterFunc` with a calculated duration to next 08:00 in `TENANT_TIMEZONE`, then tickers daily.
- Query:
  ```sql
  SELECT ea.user_id, u.email, u.preferred_locale, e.title
  FROM exam_assignments ea
  JOIN exams e ON e.id = ea.exam_id
  JOIN users u ON u.id = ea.user_id
  WHERE ea.deadline BETWEEN NOW() + INTERVAL '24 hours' AND NOW() + INTERVAL '48 hours'
    AND ea.status != 'passed'
  ```

**SMTP TLS**:
- `SMTP_TLS=true` → use `smtp.STARTTLS` via `tls.Config{ServerName: host}`
- `SMTP_TLS=false` → plain `smtp.Dial` (for local dev mail catchers like MailHog)

**Integration points** — `EmailService.Send` called from:
- `exam_assignment` service: on assignment creation → `new_exam_assigned`
- `grading` service: post-grading → `exam_passed` or `exam_failed`
- `auth` service: on temp-password creation → `password_reset`
- Scheduler: daily → `deadline_reminder`

## Notes
- MailHog is recommended for local development; add `mailhog` service to `docker-compose.yml` alongside `api`.
- The `email_log` table is append-only; no records are deleted. Retention policy (e.g. DELETE rows older than 90 days) can be added as a future maintenance job.
- HTML templates must not load external resources (images, fonts) to comply with the Content-Security-Policy in FR-BB64.
- If `attempts_remaining` is 0, the `exam_failed` template body should omit the retake section.

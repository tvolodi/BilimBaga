-- Migration 022: FR-BB61 — Email Notification Service
-- Creates the email_log table for append-only delivery audit.
CREATE TABLE email_log (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  recipient_email TEXT        NOT NULL,
  template        TEXT        NOT NULL,
  sent_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  error           TEXT
);

CREATE INDEX idx_email_log_sent_at ON email_log (sent_at);

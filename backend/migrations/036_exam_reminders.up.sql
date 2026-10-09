-- Migration 036: exam_reminders (FR-BB510 / ISS-061).
-- Records every successfully sent manual overdue reminder; used for the
-- one-reminder-per-(user, exam)-per-24h rate limit.
CREATE TABLE IF NOT EXISTS exam_reminders (
    id        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id   UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    exam_id   UUID NOT NULL REFERENCES exams(id) ON DELETE CASCADE,
    sent_by   UUID NOT NULL REFERENCES users(id),
    sent_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_exam_reminders_user_exam ON exam_reminders (user_id, exam_id, sent_at DESC);

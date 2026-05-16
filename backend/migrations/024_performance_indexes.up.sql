-- FR-BB65: Performance indexes for high-traffic query patterns.

-- session_answers: fetch answers by session
CREATE INDEX IF NOT EXISTS idx_session_answers_session_id
  ON session_answers (session_id);

-- audit_log: time-range + actor queries
CREATE INDEX IF NOT EXISTS idx_audit_log_created_at_actor
  ON audit_log (created_at DESC, actor_id);

-- exam_sessions: per-user + per-exam queries
CREATE INDEX IF NOT EXISTS idx_exam_sessions_user_exam_status
  ON exam_sessions (user_id, exam_id, status);

-- question_translations: locale lookups
CREATE INDEX IF NOT EXISTS idx_question_translations_question_locale
  ON question_translations (question_id, locale);

-- exam_assignments: assignment queries
CREATE INDEX IF NOT EXISTS idx_exam_assignments_exam_id
  ON exam_assignments (exam_id);

-- certificates: session lookup + verification
CREATE INDEX IF NOT EXISTS idx_certificates_session_id
  ON certificates (session_id);

CREATE INDEX IF NOT EXISTS idx_certificates_verification_code
  ON certificates (verification_code);

-- FR-BB65: Drop performance indexes.

DROP INDEX IF EXISTS idx_session_answers_session_id;
DROP INDEX IF EXISTS idx_audit_log_created_at_actor;
DROP INDEX IF EXISTS idx_exam_sessions_user_exam_status;
DROP INDEX IF EXISTS idx_question_translations_question_locale;
DROP INDEX IF EXISTS idx_exam_assignments_exam_id;
DROP INDEX IF EXISTS idx_certificates_session_id;
DROP INDEX IF EXISTS idx_certificates_verification_code;

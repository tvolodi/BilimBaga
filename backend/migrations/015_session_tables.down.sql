-- FR-BB36: Session Tables (rollback)

DROP TABLE IF EXISTS tab_switch_events;
DROP TABLE IF EXISTS session_answers;

ALTER TABLE session_questions
    DROP CONSTRAINT IF EXISTS session_questions_question_id_fkey;

ALTER TABLE session_questions
    ADD CONSTRAINT session_questions_question_id_fkey
        FOREIGN KEY (question_id) REFERENCES questions(id) ON DELETE CASCADE;

-- Restore surrogate PK on session_questions
ALTER TABLE session_questions DROP CONSTRAINT IF EXISTS session_questions_pkey;
ALTER TABLE session_questions ADD COLUMN IF NOT EXISTS id UUID NOT NULL DEFAULT gen_random_uuid();
ALTER TABLE session_questions ADD CONSTRAINT session_questions_pkey PRIMARY KEY (id);

ALTER TABLE session_questions
    DROP COLUMN IF EXISTS question_version_id;

DROP INDEX IF EXISTS idx_exam_sessions_exam_id;
DROP INDEX IF EXISTS idx_exam_sessions_expires_status;
DROP INDEX IF EXISTS idx_exam_sessions_user_exam_status;

ALTER TABLE exam_sessions
    DROP CONSTRAINT IF EXISTS exam_sessions_expires_after_start;

ALTER TABLE exam_sessions
    DROP CONSTRAINT IF EXISTS exam_sessions_score_range;

-- Note: PostgreSQL does not support removing enum values; grading_pending remains in session_status.

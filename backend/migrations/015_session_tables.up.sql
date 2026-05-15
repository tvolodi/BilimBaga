-- FR-BB36: Session Tables (schema completions)

-- AC-1: add grading_pending to session_status enum
ALTER TYPE session_status ADD VALUE IF NOT EXISTS 'grading_pending';

-- AC-1: add missing constraints to exam_sessions
ALTER TABLE exam_sessions
    ADD CONSTRAINT exam_sessions_score_range
        CHECK (score_pct IS NULL OR score_pct BETWEEN 0 AND 100);

ALTER TABLE exam_sessions
    ADD CONSTRAINT exam_sessions_expires_after_start
        CHECK (expires_at > started_at);

-- AC-8: composite index for portal/admin queries
CREATE INDEX IF NOT EXISTS idx_exam_sessions_user_exam_status
    ON exam_sessions(user_id, exam_id, status);

-- AC-8: partial index for auto-submit background job (FR-BB310)
CREATE INDEX IF NOT EXISTS idx_exam_sessions_expires_status
    ON exam_sessions(expires_at, status)
    WHERE status = 'in_progress';

-- AC-8: index for exam-scoped queries
CREATE INDEX IF NOT EXISTS idx_exam_sessions_exam_id
    ON exam_sessions(exam_id);

-- AC-2: rework session_questions to match FR-BB36 spec
--   - drop surrogate PK, add composite PK (session_id, question_id)
--   - drop ON DELETE CASCADE on question_id FK → RESTRICT
--   - add question_version_id column
--   - options_order column retained for backward-compat with FR-BB35 code
ALTER TABLE session_questions
    ADD COLUMN IF NOT EXISTS question_version_id UUID;

-- Change question_id FK from CASCADE to RESTRICT (AC-2)
ALTER TABLE session_questions
    DROP CONSTRAINT IF EXISTS session_questions_question_id_fkey;

ALTER TABLE session_questions
    ADD CONSTRAINT session_questions_question_id_fkey
        FOREIGN KEY (question_id) REFERENCES questions(id) ON DELETE RESTRICT;

-- AC-2: composite PK (session_id, question_id) — requires no data in table
ALTER TABLE session_questions DROP CONSTRAINT IF EXISTS session_questions_pkey;
ALTER TABLE session_questions DROP COLUMN IF EXISTS id;
ALTER TABLE session_questions ADD CONSTRAINT session_questions_pkey PRIMARY KEY (session_id, question_id);

-- AC-5: session_answers table
CREATE TABLE IF NOT EXISTS session_answers (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id          UUID NOT NULL REFERENCES exam_sessions(id) ON DELETE CASCADE,
    question_id         UUID NOT NULL REFERENCES questions(id) ON DELETE RESTRICT,
    selected_option_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
    text_answer         TEXT,
    saved_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    time_spent_seconds  INT NOT NULL DEFAULT 0 CHECK (time_spent_seconds >= 0),
    UNIQUE (session_id, question_id)
);

CREATE INDEX IF NOT EXISTS idx_session_answers_session_id
    ON session_answers(session_id);

-- AC-4: tab_switch_events table
CREATE TABLE IF NOT EXISTS tab_switch_events (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id   UUID NOT NULL REFERENCES exam_sessions(id) ON DELETE CASCADE,
    occurred_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    action_taken TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_tab_switch_events_session_id
    ON tab_switch_events(session_id);

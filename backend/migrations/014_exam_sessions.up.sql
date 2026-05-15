-- FR-BB35: Exam Session Creation

CREATE TYPE session_status AS ENUM ('in_progress', 'submitted', 'auto_submitted');

CREATE TABLE exam_sessions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    exam_id     UUID NOT NULL REFERENCES exams(id) ON DELETE CASCADE,
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status      session_status NOT NULL DEFAULT 'in_progress',
    seed        BIGINT NOT NULL,
    started_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at  TIMESTAMPTZ NOT NULL,
    submitted_at TIMESTAMPTZ,
    score_pct   DECIMAL(5, 2),
    passed      BOOL NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_exam_sessions_exam_user ON exam_sessions(exam_id, user_id);
CREATE INDEX idx_exam_sessions_user_id ON exam_sessions(user_id);
CREATE INDEX idx_exam_sessions_status ON exam_sessions(status);

-- Resolved question set for a session (one row per question, ordered by sort_order).
-- options_order stores a JSONB array of option UUIDs in the shuffled order.
CREATE TABLE session_questions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id      UUID NOT NULL REFERENCES exam_sessions(id) ON DELETE CASCADE,
    question_id     UUID NOT NULL REFERENCES questions(id) ON DELETE CASCADE,
    sort_order      INT NOT NULL CHECK (sort_order >= 0),
    options_order   JSONB NOT NULL DEFAULT '[]'::jsonb,
    UNIQUE (session_id, sort_order)
);

CREATE INDEX idx_session_questions_session_id ON session_questions(session_id);

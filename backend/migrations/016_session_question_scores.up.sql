-- FR-BB311: Grading Engine — per-question score store

CREATE TYPE grading_status AS ENUM ('graded', 'pending_manual', 'ai_graded');

CREATE TABLE session_question_scores (
    session_id      UUID NOT NULL REFERENCES exam_sessions(id) ON DELETE CASCADE,
    question_id     UUID NOT NULL REFERENCES questions(id)     ON DELETE RESTRICT,
    score           DECIMAL(6, 4) NOT NULL DEFAULT 0,
    max_score       DECIMAL(6, 4) NOT NULL DEFAULT 1,
    grading_status  grading_status NOT NULL DEFAULT 'graded',
    PRIMARY KEY (session_id, question_id)
);

CREATE INDEX idx_session_question_scores_session_id
    ON session_question_scores(session_id);

CREATE INDEX idx_session_question_scores_pending
    ON session_question_scores(session_id)
    WHERE grading_status = 'pending_manual';

-- #546 (FR-BB73 revision 2): the per-exam AI grading mode, and the suggestion columns of a short-text answer.
-- AC-9: existing exams get 'suggest', so a host that ran 'auto' for every exam moves to 'suggest' on deploy.
ALTER TABLE exams
    ADD COLUMN ai_grading_mode TEXT NOT NULL DEFAULT 'suggest'
        CHECK (ai_grading_mode IN ('off', 'suggest', 'auto'));

-- AC-10: the suggestion of the AI, its time, and the flags that routed it (AC-15). None of these is read by any score,
-- grading status, session status, pass/fail, mail or certificate computation (AC-13).
ALTER TABLE session_question_scores
    ADD COLUMN ai_suggested_score_pct NUMERIC(5, 2) NULL
        CHECK (ai_suggested_score_pct BETWEEN 0 AND 100),
    ADD COLUMN ai_suggested_at TIMESTAMPTZ NULL,
    ADD COLUMN ai_flags TEXT[] NULL;

-- Reverses 040_ai_grading_mode: the suggestion columns first, then the exam column with its check.
ALTER TABLE session_question_scores
    DROP COLUMN IF EXISTS ai_flags,
    DROP COLUMN IF EXISTS ai_suggested_at,
    DROP COLUMN IF EXISTS ai_suggested_score_pct;

ALTER TABLE exams
    DROP COLUMN IF EXISTS ai_grading_mode;

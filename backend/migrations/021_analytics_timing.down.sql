-- Migration 021 rollback
ALTER TABLE session_question_scores
    DROP COLUMN IF EXISTS time_taken_seconds;

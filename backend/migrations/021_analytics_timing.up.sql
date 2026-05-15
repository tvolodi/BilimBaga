-- Migration 021: FR-BB52 — Per-Exam Analytics API
-- Adds grader-verified precision timing to session_question_scores.
ALTER TABLE session_question_scores
    ADD COLUMN IF NOT EXISTS time_taken_seconds DECIMAL(8,2);

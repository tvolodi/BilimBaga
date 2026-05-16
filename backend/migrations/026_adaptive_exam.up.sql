-- FR-BB72: Adaptive Difficulty
-- Add adaptive flag to exams and adaptive_state JSONB to exam_sessions

ALTER TABLE exams
  ADD COLUMN adaptive BOOL NOT NULL DEFAULT false;

-- Session state: track served questions and current difficulty
ALTER TABLE exam_sessions
  ADD COLUMN adaptive_state JSONB;

-- adaptive_state schema (for reference):
-- {
--   "current_difficulty": "easy|medium|hard",
--   "served_question_ids": ["uuid", ...],
--   "recent_results": [true, false, true],   -- last N answers, most recent last
--   "current_question_id": "uuid | null"      -- pending unanswered question
-- }

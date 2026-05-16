-- Reverse FR-BB72: Adaptive Difficulty

ALTER TABLE exam_sessions
  DROP COLUMN IF EXISTS adaptive_state;

ALTER TABLE exams
  DROP COLUMN IF EXISTS adaptive;

-- FR-BB73: Short-Text Auto-Grading
-- Adds auto_grade/model_answer to questions and ai_reasoning to session_question_scores.

ALTER TABLE questions
  ADD COLUMN auto_grade   BOOL NOT NULL DEFAULT false,
  ADD COLUMN model_answer TEXT;

ALTER TABLE session_question_scores
  ADD COLUMN ai_reasoning TEXT;

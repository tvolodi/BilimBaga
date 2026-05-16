-- FR-BB73: Short-Text Auto-Grading (down)
ALTER TABLE session_question_scores DROP COLUMN IF EXISTS ai_reasoning;
ALTER TABLE questions DROP COLUMN IF EXISTS model_answer;
ALTER TABLE questions DROP COLUMN IF EXISTS auto_grade;

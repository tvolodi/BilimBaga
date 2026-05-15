-- FR-BB31: Exam Configuration Model (rollback)

DROP TRIGGER IF EXISTS exams_updated_at ON exams;

DROP TABLE IF EXISTS exam_manual_questions;
DROP TABLE IF EXISTS exam_question_rules;
DROP TABLE IF EXISTS exam_sections;
DROP TABLE IF EXISTS exams;

DROP TYPE IF EXISTS question_selection_mode;
DROP TYPE IF EXISTS tab_switch_action;
DROP TYPE IF EXISTS show_answers_policy;
DROP TYPE IF EXISTS exam_status;

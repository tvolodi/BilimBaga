-- FR-BB41: Result Retrieval API — link session_questions to the rule that produced them
-- Nullable: rows created before this migration (or without a rule) will remain NULL.
-- section_scores queries skip NULL rule_id rows via "AND sq.rule_id IS NOT NULL".
ALTER TABLE session_questions
  ADD COLUMN rule_id UUID REFERENCES exam_question_rules(id);

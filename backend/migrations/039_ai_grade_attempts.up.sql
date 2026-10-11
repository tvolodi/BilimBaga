-- #528: the AI grading job counts its attempts per answer. A row is picked again only while this is below 2.
ALTER TABLE session_question_scores
    ADD COLUMN ai_grade_attempts SMALLINT NOT NULL DEFAULT 0;

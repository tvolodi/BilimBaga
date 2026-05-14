-- Rollback for migration 009: FR-BB22 — Question Model
-- Drop tables in reverse dependency order.

DROP TRIGGER IF EXISTS question_translations_updated_at ON question_translations;
DROP TRIGGER IF EXISTS questions_updated_at ON questions;

DROP TABLE IF EXISTS question_tags;
DROP TABLE IF EXISTS answer_translations;
DROP TABLE IF EXISTS answer_options;
DROP TABLE IF EXISTS question_translations;
DROP TABLE IF EXISTS questions;

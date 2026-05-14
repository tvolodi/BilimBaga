-- Rollback migration 011.
DROP INDEX IF EXISTS idx_question_translations_stem_trgm_en;
DROP INDEX IF EXISTS idx_question_translations_stem_trgm_ru;
DROP INDEX IF EXISTS idx_question_translations_stem_trgm_kk;
-- Note: pg_trgm extension is shared and not dropped to avoid breaking other consumers.

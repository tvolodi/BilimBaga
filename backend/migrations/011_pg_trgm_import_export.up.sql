-- Migration 011: FR-BB25 — enable pg_trgm for duplicate detection during import.
-- The extension and index are created idempotently so re-runs are safe.

CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- GIN trigram indexes on question_translations.stem per locale for fast similarity search.
CREATE INDEX IF NOT EXISTS idx_question_translations_stem_trgm_kk
    ON question_translations
    USING GIN (stem gin_trgm_ops)
    WHERE locale = 'kk';

CREATE INDEX IF NOT EXISTS idx_question_translations_stem_trgm_ru
    ON question_translations
    USING GIN (stem gin_trgm_ops)
    WHERE locale = 'ru';

CREATE INDEX IF NOT EXISTS idx_question_translations_stem_trgm_en
    ON question_translations
    USING GIN (stem gin_trgm_ops)
    WHERE locale = 'en';

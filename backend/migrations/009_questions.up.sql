-- Migration 009: FR-BB22 — Question Model
-- Creates 5 tables for the multilingual question bank.

CREATE TABLE IF NOT EXISTS questions (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    category_id    UUID NOT NULL REFERENCES categories(id),
    difficulty     TEXT NOT NULL CHECK (difficulty IN ('easy','medium','hard')),
    type           TEXT NOT NULL CHECK (type IN ('single','multiple','truefalse','likert','shorttext')),
    default_locale TEXT NOT NULL DEFAULT 'kk',
    status         TEXT NOT NULL DEFAULT 'draft'
                        CHECK (status IN ('draft','review','active','archived')),
    created_by     UUID NOT NULL REFERENCES users(id),
    version        INT  NOT NULL DEFAULT 1,
    parent_id      UUID REFERENCES questions(id),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_questions_category_id ON questions(category_id);
CREATE INDEX IF NOT EXISTS idx_questions_status      ON questions(status);
CREATE INDEX IF NOT EXISTS idx_questions_parent_id   ON questions(parent_id);
CREATE INDEX IF NOT EXISTS idx_questions_created_by  ON questions(created_by);

CREATE TABLE IF NOT EXISTS question_translations (
    question_id  UUID NOT NULL REFERENCES questions(id) ON DELETE CASCADE,
    locale       TEXT NOT NULL,
    stem         TEXT NOT NULL,
    explanation  TEXT,
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (question_id, locale)
);

CREATE TABLE IF NOT EXISTS answer_options (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    question_id     UUID NOT NULL REFERENCES questions(id) ON DELETE CASCADE,
    sort_order      INT  NOT NULL DEFAULT 0,
    is_correct      BOOL NOT NULL DEFAULT FALSE,
    likert_weight   DECIMAL(5,2),
    likert_polarity TEXT CHECK (likert_polarity IN ('positive','negative')),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_answer_options_question_id ON answer_options(question_id);

CREATE TABLE IF NOT EXISTS answer_translations (
    option_id  UUID NOT NULL REFERENCES answer_options(id) ON DELETE CASCADE,
    locale     TEXT NOT NULL,
    text       TEXT NOT NULL,
    PRIMARY KEY (option_id, locale)
);

CREATE TABLE IF NOT EXISTS question_tags (
    question_id UUID NOT NULL REFERENCES questions(id) ON DELETE CASCADE,
    tag_id      UUID NOT NULL REFERENCES tags(id)      ON DELETE CASCADE,
    PRIMARY KEY (question_id, tag_id)
);

-- Trigger function: auto-update updated_at on row changes.
CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER questions_updated_at
    BEFORE UPDATE ON questions
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER question_translations_updated_at
    BEFORE UPDATE ON question_translations
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

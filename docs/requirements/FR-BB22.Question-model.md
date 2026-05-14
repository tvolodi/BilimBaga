# FR-BB22 — Question Model

## Metadata
| Field | Value |
|-------|-------|
| ID | FR-BB22 |
| Phase | 2 — Content Management |
| Priority | 1 |
| Status | Implemented |
| Depends On | FR-BB21 |

## Scope

| Layer | Items |
|-------|-------|
| Database | 5 new tables: `questions`, `question_translations`, `answer_options`, `answer_translations`, `question_tags` |
| Go package | `internal/questions` — repository and model types for these tables are owned here |
| Frontend | N/A for this FR |
| i18n keys | N/A for this FR |

## Description
Defines the core data model for the multilingual question bank. A question record stores type, difficulty, status, and authorship metadata; all user-visible text lives in separate translation rows keyed by locale. Answer options are similarly split into a structure table and a translation table. Versioning is achieved by chaining questions via `parent_id` so every edit to an active question produces a new version while the old one is archived automatically.

## Acceptance Criteria
- [ ] AC-1: The `questions` table exists with all specified columns; `type` is constrained to `('single','multiple','truefalse','likert','shorttext')`; `difficulty` is constrained to `('easy','medium','hard')`; `status` is constrained to `('draft','review','active','archived')`; the database rejects out-of-range enum values.
- [ ] AC-2: The `question_translations` table enforces a composite PK on `(question_id, locale)` and a FK to `questions(id) ON DELETE CASCADE`; inserting a translation for a non-existent question is rejected by the DB.
- [ ] AC-3: The `answer_options` table enforces a FK to `questions(id) ON DELETE CASCADE`; `sort_order` defaults to 0 (the DB column default); the caller is responsible for setting explicit `sort_order` values when display order matters; `likert_weight` and `likert_polarity` are NULL for non-Likert question types and are validated at the application layer.
- [ ] AC-4: The `answer_translations` table enforces a composite PK on `(option_id, locale)` and a FK to `answer_options(id) ON DELETE CASCADE`.
- [ ] AC-5: The `question_tags` join table enforces a composite PK on `(question_id, tag_id)`, FKs to both parent tables with `ON DELETE CASCADE`, preventing orphan associations.
- [ ] AC-6: When a new version of an active question is created, `parent_id` on the new row references the previous question's `id`; the previous question's `status` is set to `archived` atomically in the same transaction; only one question per parent chain may have `status = 'active'` at any time.
- [ ] AC-7: `version` starts at 1 for new questions and increments by 1 with each new version row; the application sets this value (not a DB trigger) and must be verified by unit tests.
- [ ] AC-8: `default_locale` must be one of the tenant's configured `available_locales` (validated at the application layer); a `question_translations` row for `default_locale` must exist before status can advance beyond `draft`.
- [ ] AC-9: All timestamp columns (`created_at`, `updated_at`) store UTC values; `updated_at` is refreshed on every UPDATE via either a DB trigger or explicit application-layer assignment.
- [ ] AC-10: A database migration file creates all five tables in dependency order; the migration is idempotent when run via `golang-migrate` and does not break existing Phase 1 tables.

## Technical Specification

### Database Schema

```sql
-- Questions (core metadata, no user-visible text)
CREATE TABLE questions (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    category_id    UUID NOT NULL REFERENCES categories(id),
    difficulty     TEXT NOT NULL CHECK (difficulty IN ('easy','medium','hard')),
    type           TEXT NOT NULL CHECK (type IN ('single','multiple','truefalse','likert','shorttext')),
    default_locale TEXT NOT NULL DEFAULT 'kk',
    status         TEXT NOT NULL DEFAULT 'draft'
                        CHECK (status IN ('draft','review','active','archived')),
    created_by     UUID NOT NULL REFERENCES users(id),
    version        INT  NOT NULL DEFAULT 1,
    parent_id      UUID REFERENCES questions(id),   -- links to previous version
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_questions_category_id ON questions(category_id);
CREATE INDEX idx_questions_status      ON questions(status);
CREATE INDEX idx_questions_parent_id   ON questions(parent_id);
CREATE INDEX idx_questions_created_by  ON questions(created_by);

-- Question translations (user-visible text per locale)
CREATE TABLE question_translations (
    question_id  UUID NOT NULL REFERENCES questions(id) ON DELETE CASCADE,
    locale       TEXT NOT NULL,
    stem         TEXT NOT NULL,          -- the question text
    explanation  TEXT,                   -- shown after answering
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (question_id, locale)
);

-- Answer options (structure only, no text)
CREATE TABLE answer_options (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    question_id     UUID NOT NULL REFERENCES questions(id) ON DELETE CASCADE,
    sort_order      INT  NOT NULL DEFAULT 0,
    is_correct      BOOL NOT NULL DEFAULT FALSE,
    likert_weight   DECIMAL(5,2),        -- NULL for non-Likert types
    likert_polarity TEXT CHECK (likert_polarity IN ('positive','negative')),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_answer_options_question_id ON answer_options(question_id);

-- Answer option translations
CREATE TABLE answer_translations (
    option_id  UUID NOT NULL REFERENCES answer_options(id) ON DELETE CASCADE,
    locale     TEXT NOT NULL,
    text       TEXT NOT NULL,
    PRIMARY KEY (option_id, locale)
);

-- Question ↔ Tag association
CREATE TABLE question_tags (
    question_id UUID NOT NULL REFERENCES questions(id) ON DELETE CASCADE,
    tag_id      UUID NOT NULL REFERENCES tags(id)      ON DELETE CASCADE,
    PRIMARY KEY (question_id, tag_id)
);

-- Trigger: auto-update updated_at on questions
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
```

## Go Domain Package

Go domain package: `internal/questions` — repository and model types for these tables are owned here. No other package may import question repository types directly; access goes through the `questions` service interface.

## Out of Scope

- REST API endpoints for question CRUD (FR-BB23)
- Translation upsert API (FR-BB24)
- RBAC permission rows for questions (covered separately)
- Bulk import/export of question banks (FR-BB25)

## Test Strategy

1. **Migration integration test**: verify all five tables (`questions`, `question_translations`, `answer_options`, `answer_translations`, `question_tags`) are created and foreign key constraints are enforced (e.g., inserting a child row with a non-existent parent UUID is rejected).
2. **DB constraint tests**: verify that inserting invalid `type`, `difficulty`, or `status` values is rejected by the database CHECK constraints.
3. **Versioning transaction unit test** (AC-6): verify that creating a new version atomically sets `parent_id`, sets the previous question's `status` to `'archived'`, and that no two rows with the same parent chain have `status = 'active'`.

## Notes
- The split between `questions` + `answer_options` (structure) and their `_translations` counterparts means grading logic never needs to parse locale-specific text — it only reads `is_correct` and `likert_weight` from the structure tables.
- `shorttext` questions have no answer options rows; grading is handled by keyword matching or manual review (Phase 4).
- `truefalse` questions must have exactly two answer options seeded at creation time; the service layer enforces this.
- The `parent_id` chain forms a singly-linked list. To retrieve full version history, walk the chain in application code; there is no recursive CTE required for normal reads.
- `likert_polarity` supports reverse-scored items (where a high-weighted response is "negative"), enabling competency scale scoring in Phase 5 analytics.

# FR-BB31 — Exam Configuration Model

## Metadata
| Field | Value |
|-------|-------|
| ID | FR-BB31 |
| Phase | 3 — Exam Engine |
| Priority | 1 |
| Status | Draft |
| Depends On | FR-BB22 |

## Description
Defines the database schema for storing exam configurations, including top-level exam settings, optional sections for grouping questions, question selection rules (manual or random), and the explicit question lists for manual rules. This model drives all downstream session creation, grading, and reporting logic.

## Acceptance Criteria
- [ ] AC-1: `exams` table exists with all specified columns; `status` column accepts only `'draft'`, `'active'`, `'archived'` via a PostgreSQL CHECK or ENUM constraint.
- [ ] AC-2: `exam_sections` table exists and enforces FK to `exams.id` with ON DELETE CASCADE; `sort_order` is non-negative.
- [ ] AC-3: `exam_question_rules` table exists; `mode` accepts only `'manual'` or `'random'`; FK to `exams.id` with ON DELETE CASCADE; FK to `exam_sections.id` nullable with ON DELETE SET NULL.
- [ ] AC-4: `exam_manual_questions` table exists with composite PK `(rule_id, question_id)`; FKs to `exam_question_rules.id` and `questions.id` with ON DELETE CASCADE.
- [ ] AC-5: `show_answers` ENUM accepts only `'never'`, `'after_completion'`, `'after_all_attempts'`; `on_tab_switch` ENUM accepts only `'log'`, `'warn'`, `'submit'`.
- [ ] AC-6: All UUID primary keys are generated as UUID v4 at the database level via `gen_random_uuid()`.
- [ ] AC-7: `created_at` and `updated_at` columns default to `NOW()` UTC; `updated_at` is automatically maintained by a trigger.
- [ ] AC-8: `tag_ids` in `exam_question_rules` is stored as JSONB and accepts a JSON array of UUID strings (may be empty array).
- [ ] AC-9: Migration file is numbered sequentially and does not modify any existing migration files.
- [ ] AC-10: All tables have appropriate indexes on foreign key columns for join performance.

## Technical Specification

### Database Schema

```sql
-- Exam status type
CREATE TYPE exam_status AS ENUM ('draft', 'active', 'archived');

-- Show-answers policy type
CREATE TYPE show_answers_policy AS ENUM ('never', 'after_completion', 'after_all_attempts');

-- Tab-switch action type
CREATE TYPE tab_switch_action AS ENUM ('log', 'warn', 'submit');

-- Core exam configuration
CREATE TABLE exams (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title               TEXT NOT NULL,
    description         TEXT,
    status              exam_status NOT NULL DEFAULT 'draft',
    time_limit_minutes  INT NOT NULL CHECK (time_limit_minutes > 0),
    passing_score_pct   DECIMAL(5, 2) NOT NULL CHECK (passing_score_pct BETWEEN 0 AND 100),
    max_attempts        INT NOT NULL DEFAULT 1 CHECK (max_attempts > 0),
    available_from      TIMESTAMPTZ,
    available_until     TIMESTAMPTZ,
    shuffle_questions   BOOL NOT NULL DEFAULT FALSE,
    shuffle_options     BOOL NOT NULL DEFAULT FALSE,
    show_answers        show_answers_policy NOT NULL DEFAULT 'never',
    on_tab_switch       tab_switch_action NOT NULL DEFAULT 'log',
    certificate_enabled BOOL NOT NULL DEFAULT FALSE,
    created_by          UUID NOT NULL REFERENCES users(id),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT exams_availability_check CHECK (
        available_from IS NULL OR available_until IS NULL OR available_from < available_until
    )
);

CREATE INDEX idx_exams_status ON exams(status);
CREATE INDEX idx_exams_created_by ON exams(created_by);

-- Optional grouping sections within an exam
CREATE TABLE exam_sections (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    exam_id     UUID NOT NULL REFERENCES exams(id) ON DELETE CASCADE,
    title       TEXT,
    sort_order  INT NOT NULL DEFAULT 0 CHECK (sort_order >= 0),
    UNIQUE (exam_id, sort_order)
);

CREATE INDEX idx_exam_sections_exam_id ON exam_sections(exam_id);

-- Question selection rules (one rule = one pool or one manual list)
CREATE TYPE question_selection_mode AS ENUM ('manual', 'random');

CREATE TABLE exam_question_rules (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    exam_id     UUID NOT NULL REFERENCES exams(id) ON DELETE CASCADE,
    section_id  UUID REFERENCES exam_sections(id) ON DELETE SET NULL,
    mode        question_selection_mode NOT NULL DEFAULT 'random',
    category_id UUID REFERENCES categories(id) ON DELETE SET NULL,
    tag_ids     JSONB NOT NULL DEFAULT '[]'::jsonb,
    difficulty  TEXT,
    count       INT NOT NULL CHECK (count > 0),
    sort_order  INT NOT NULL DEFAULT 0 CHECK (sort_order >= 0),
    UNIQUE (exam_id, sort_order)
);

CREATE INDEX idx_exam_question_rules_exam_id ON exam_question_rules(exam_id);
CREATE INDEX idx_exam_question_rules_section_id ON exam_question_rules(section_id);

-- Explicit question list for manual-mode rules
CREATE TABLE exam_manual_questions (
    rule_id     UUID NOT NULL REFERENCES exam_question_rules(id) ON DELETE CASCADE,
    question_id UUID NOT NULL REFERENCES questions(id) ON DELETE CASCADE,
    sort_order  INT NOT NULL DEFAULT 0 CHECK (sort_order >= 0),
    PRIMARY KEY (rule_id, question_id)
);

CREATE INDEX idx_exam_manual_questions_rule_id ON exam_manual_questions(rule_id);

-- Auto-maintain updated_at on exams
CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER exams_updated_at
    BEFORE UPDATE ON exams
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
```

## Notes
- The `difficulty` column in `exam_question_rules` is stored as free-text to remain flexible across question bank extensions; a CHECK constraint may be added later once difficulty ENUM is finalised in the question model.
- `tag_ids` JSONB must be validated at the application layer to ensure it contains a valid array of UUID strings before persistence.
- `exam_sections` is optional per exam; an exam may have rules with `section_id = NULL` (un-sectioned exam).
- The `updated_at` trigger function `set_updated_at()` may already exist from an earlier migration; use `CREATE OR REPLACE` to be idempotent.

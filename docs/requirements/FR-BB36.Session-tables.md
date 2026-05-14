# FR-BB36 — Session Tables

## Metadata
| Field | Value |
|-------|-------|
| ID | FR-BB36 |
| Phase | 3 — Exam Engine |
| Priority | 1 |
| Status | Draft |
| Depends On | FR-BB35 |

## Description
Defines the database schema for all session-related tables: the core `exam_sessions` record, the `session_questions` snapshot of question assignment, `session_answers` for per-question responses, and `tab_switch_events` for anti-cheat tracking. These tables serve as the persistence layer for FR-BB35 (creation), FR-BB37 (answers), FR-BB38 (events), FR-BB39 (submission), and FR-BB311 (grading).

## Acceptance Criteria
- [ ] AC-1: `exam_sessions` table exists with all specified columns; `status` ENUM accepts only `'in_progress'`, `'submitted'`, `'auto_submitted'`, `'grading_pending'`.
- [ ] AC-2: `session_questions` has composite PK `(session_id, question_id)`; FK to `exam_sessions.id` ON DELETE CASCADE and to `questions.id` ON DELETE RESTRICT (questions must not be hard-deleted while sessions reference them).
- [ ] AC-3: `session_answers` upsert on `(session_id, question_id)` works correctly; the table has a unique index on `(session_id, question_id)`.
- [ ] AC-4: `tab_switch_events` records are insertable while the session is `in_progress`; `session_id` FK has ON DELETE CASCADE.
- [ ] AC-5: `session_answers.selected_option_ids` is stored as JSONB; an empty array `[]` is valid for unanswered states.
- [ ] AC-6: `exam_sessions.seed` is a `BIGINT NOT NULL`; it stores the unix nanosecond timestamp used as PRNG seed.
- [ ] AC-7: `session_questions.question_version_id` references a version snapshot of the question at time of session creation (FK to `question_versions.id` if that table exists, or nullable UUID if versioning is added in a later phase).
- [ ] AC-8: All index definitions are included in the migration for: `session_answers(session_id)`, `exam_sessions(user_id, exam_id, status)`, `tab_switch_events(session_id)`, `exam_sessions(expires_at, status)` (for the auto-submit background job query).
- [ ] AC-9: Migration is numbered sequentially after FR-BB35's migration; no existing migration files are modified.
- [ ] AC-10: `session_answers.time_spent_seconds` is NOT NULL with default 0; values must be non-negative (CHECK constraint).

## Technical Specification

### Database Schema

```sql
-- Session status type
CREATE TYPE session_status AS ENUM (
    'in_progress',
    'submitted',
    'auto_submitted',
    'grading_pending'
);

-- Core session record
CREATE TABLE exam_sessions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    exam_id         UUID NOT NULL REFERENCES exams(id),
    user_id         UUID NOT NULL REFERENCES users(id),
    status          session_status NOT NULL DEFAULT 'in_progress',
    started_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at      TIMESTAMPTZ NOT NULL,
    submitted_at    TIMESTAMPTZ,
    score_pct       DECIMAL(5, 2),
    passed          BOOL,
    seed            BIGINT NOT NULL,
    CONSTRAINT exam_sessions_score_range CHECK (score_pct IS NULL OR score_pct BETWEEN 0 AND 100),
    CONSTRAINT exam_sessions_expires_after_start CHECK (expires_at > started_at)
);

CREATE INDEX idx_exam_sessions_user_exam_status
    ON exam_sessions(user_id, exam_id, status);

CREATE INDEX idx_exam_sessions_expires_status
    ON exam_sessions(expires_at, status)
    WHERE status = 'in_progress';

CREATE INDEX idx_exam_sessions_exam_id
    ON exam_sessions(exam_id);

-- Snapshot of which questions were assigned to this session
CREATE TABLE session_questions (
    session_id          UUID NOT NULL REFERENCES exam_sessions(id) ON DELETE CASCADE,
    question_id         UUID NOT NULL REFERENCES questions(id) ON DELETE RESTRICT,
    question_version_id UUID,  -- nullable until question versioning is implemented
    sort_order          INT NOT NULL CHECK (sort_order >= 0),
    PRIMARY KEY (session_id, question_id)
);

CREATE INDEX idx_session_questions_session_id
    ON session_questions(session_id);

-- User's answers per question within a session
CREATE TABLE session_answers (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id          UUID NOT NULL REFERENCES exam_sessions(id) ON DELETE CASCADE,
    question_id         UUID NOT NULL REFERENCES questions(id) ON DELETE RESTRICT,
    selected_option_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
    text_answer         TEXT,
    saved_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    time_spent_seconds  INT NOT NULL DEFAULT 0 CHECK (time_spent_seconds >= 0),
    UNIQUE (session_id, question_id)
);

CREATE INDEX idx_session_answers_session_id
    ON session_answers(session_id);

-- Anti-cheat: tab-switch and focus-loss events
CREATE TABLE tab_switch_events (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id   UUID NOT NULL REFERENCES exam_sessions(id) ON DELETE CASCADE,
    occurred_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    action_taken TEXT NOT NULL  -- e.g. 'log', 'warn', 'submit'
);

CREATE INDEX idx_tab_switch_events_session_id
    ON tab_switch_events(session_id);
```

## Notes
- `session_questions.question_version_id` is left nullable for now. When question versioning (a future phase) is implemented, a migration will populate historical values and add the FK constraint.
- The partial index `idx_exam_sessions_expires_status WHERE status = 'in_progress'` is specifically optimised for the auto-submit background job query pattern (FR-BB310).
- `session_answers` uses `UNIQUE (session_id, question_id)` to enable efficient `INSERT ... ON CONFLICT DO UPDATE` (upsert) in the answer-saving endpoint (FR-BB37).
- `action_taken` in `tab_switch_events` stores the actual action taken at event time (`log`, `warn`, or `submit`) so the audit trail reflects the configuration in effect when the event occurred.
- The `exam_sessions` table intentionally does not cascade-delete from `exams` (no ON DELETE CASCADE on the `exam_id` FK) to prevent accidental data loss when archiving exams.

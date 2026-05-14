# FR-BB23 — Question CRUD API

## Metadata
| Field | Value |
|-------|-------|
| ID | FR-BB23 |
| Phase | 2 — Content Management |
| Priority | 1 |
| Status | implemented |
| Depends On | FR-BB22 |

## Description
Exposes a complete REST API for creating, reading, updating, and managing the lifecycle of questions in the question bank. Endpoints cover single-question operations, paginated/filtered listing, status workflow transitions, hard deletion of drafts, version history retrieval, and tag association management. All mutating operations are restricted to the `examiner` role and above; the response always includes a `locale_coverage` computed field indicating which locales have translations.

## Scope

| Layer | Items |
|-------|-------|
| Backend packages | `internal/questions` — new `handler.go`; service interface extensions (see Technical Notes); repository query additions |
| Migrations | New `010_question_tags_tag_id_index.up.sql` / `010_question_tags_tag_id_index.down.sql` |
| Frontend | None — frontend question editor is FR-BB26/FR-BB27 |
| i18n keys | None added in this requirement |

## Acceptance Criteria
- [ ] AC-1: `GET /api/v1/questions` returns a paginated response with an `items` array and a `meta` object containing `page`, `per_page`, and `total` (matching the existing `users`/`audit` API convention); it accepts query parameters `category_id`, `tag_id`, `difficulty`, `type`, `status`, `locale`, and `locale_missing` (returns questions lacking translation for a given locale); unauthorized requests receive `401`.
- [ ] AC-2: `POST /api/v1/questions` creates a question in `draft` status; it requires `default_locale` stem and at least 2 answer options for `single`, `multiple`, and `truefalse` types; `likert` type requires at least 2 answer options and `likert_weight` must be non-null per option; creating `shorttext` with zero options is valid; validation failures return `422` with a per-field error array.
- [ ] AC-3: `GET /api/v1/questions/:id` returns the full question including all `question_translations`, all `answer_options` with their `answer_translations`, associated tag IDs, and the computed `locale_coverage` array; returns `404` if the question does not exist.
- [ ] AC-4: `PUT /api/v1/questions/:id` — if the question's current status is `active`, the service atomically creates a new version (incremented `version`, `parent_id` pointing to the current row, status `draft`), archives the current row, and returns the new question ID in the response; if status is `draft` or `review`, it updates in place.
- [ ] AC-5: `POST /api/v1/questions/:id/status` enforces the workflow: `draft → review → active`; `active → archived`; any other transition returns `422` with error code `ERR_INVALID_TRANSITION`; transition to `review` is blocked (400) when the default locale stem translation (`stem` text) is missing or empty; transition to `active` is blocked (400) when the default locale stem translation is missing or empty. These rules align with FR-BB22 AC-8.
- [ ] AC-6: `DELETE /api/v1/questions/:id` succeeds only when status is `draft`; attempting to delete a non-draft question returns `409` with error code `ERR_NOT_DRAFT`; deletion is a hard delete (CASCADE removes translations and answer options).
- [ ] AC-7: `GET /api/v1/questions/:id/versions` returns the full version chain ordered oldest-first, each entry containing `id`, `version`, `status`, `created_at`, and `created_by`; response shape is `data: { versions: [...] }` with no pagination meta (see Technical Notes for rationale).
- [ ] AC-8: `POST /api/v1/questions/:id/tags` and `DELETE /api/v1/questions/:id/tags/:tagId` add/remove tag associations; adding a non-existent tag ID returns `404`; adding a duplicate is idempotent (no error).
- [ ] AC-9: Every mutating endpoint records an audit log entry with `entity_type = 'question'`, the question `id`, the `action` performed, the `actor_id`, and a JSON diff of changed fields serialised as `{ "field": { "old": <previous_value>, "new": <new_value> } }` stored in the `metadata` column of the `audit_log` table (see `internal/audit` metadata column convention).
- [ ] AC-10: The list query execution plan must use an index scan (not a sequential scan) on questions for all supported filter combinations. Validated via EXPLAIN ANALYZE in integration tests with a 10k-row fixture.
- [ ] AC-11: An authenticated request from a caller without the `examiner` role (e.g., `employee`) returns `403 Forbidden`.

## Technical Specification

### API Endpoints

| Method | Path | Auth | RBAC Action | Description |
|--------|------|------|-------------|-------------|
| GET | `/api/v1/questions` | examiner+ | `questions:read` | Paginated filtered question list |
| POST | `/api/v1/questions` | examiner+ | `questions:write` | Create question in draft |
| GET | `/api/v1/questions/:id` | examiner+ | `questions:read` | Full question detail |
| PUT | `/api/v1/questions/:id` | examiner+ | `questions:write` | Update; auto-versions active questions |
| POST | `/api/v1/questions/:id/status` | examiner+ | `questions:write` | Transition status |
| DELETE | `/api/v1/questions/:id` | examiner+ | `questions:write` | Hard delete draft |
| GET | `/api/v1/questions/:id/versions` | examiner+ | `questions:read` | Version history |
| POST | `/api/v1/questions/:id/tags` | examiner+ | `questions:write` | Add tag association |
| DELETE | `/api/v1/questions/:id/tags/:tagId` | examiner+ | `questions:write` | Remove tag association |

#### Request / Response Shapes

**GET /api/v1/questions?page=1&per_page=20&status=active&difficulty=hard**
```json
{
  "data": {
    "items": [
      {
        "id": "uuid-q1",
        "type": "single",
        "difficulty": "hard",
        "status": "active",
        "category_id": "uuid-cat",
        "default_locale": "kk",
        "version": 2,
        "locale_coverage": ["kk", "ru"],
        "stem_preview": "Қандай хаттама...",
        "tags": ["network-security"], // tag names (not IDs) — resolved by backend JOIN
        "created_by": "uuid-user",
        "created_at": "2026-05-14T10:00:00Z",
        "updated_at": "2026-05-14T10:00:00Z"
      }
    ],
    "meta": {
      "page": 1,
      "per_page": 20,
      "total": 1
    }
  },
  "error": null
}
```

**POST /api/v1/questions — Request**
```json
{
  "category_id": "uuid-cat",
  "difficulty": "medium",
  "type": "single",
  "default_locale": "kk",
  "translations": {
    "kk": {
      "stem": "Вирустық бағдарлама дегеніміз не?",
      "explanation": "Вирус — зиянды код."
    }
  },
  "answer_options": [
    { "sort_order": 1, "is_correct": true,  "translations": { "kk": { "text": "Зиянды бағдарлама" } } },
    { "sort_order": 2, "is_correct": false, "translations": { "kk": { "text": "Жүйелік файл" } } }
  ],
  "tag_ids": ["uuid-tag1"]
}
```

**POST /api/v1/questions — Response (201)**
```json
{
  "data": {
    "id": "new-uuid",
    "type": "single",
    "difficulty": "medium",
    "status": "draft",
    "category_id": "uuid-cat",
    "default_locale": "kk",
    "version": 1,
    "parent_id": null,
    "locale_coverage": ["kk"],
    "created_by": "uuid-user",
    "created_at": "2026-05-14T10:00:00Z",
    "updated_at": "2026-05-14T10:00:00Z"
  },
  "error": null
}
```

**GET /api/v1/questions/:id — Response (200)**
```json
{
  "data": {
    "id": "uuid-q1",
    "type": "single",
    "difficulty": "medium",
    "status": "active",
    "category_id": "uuid-cat",
    "default_locale": "kk",
    "version": 1,
    "parent_id": null,
    "locale_coverage": ["kk", "ru"],
    "translations": {
      "kk": { "stem": "Вирустық бағдарлама...", "explanation": "Вирус — зиянды код." },
      "ru": { "stem": "Что такое вирус?", "explanation": "Вирус — вредоносный код." }
    },
    "answer_options": [
      {
        "id": "uuid-opt1",
        "sort_order": 1,
        "is_correct": true,
        "likert_weight": null,
        "likert_polarity": null,
        "translations": {
          "kk": { "text": "Зиянды бағдарлама" },
          "ru": { "text": "Вредоносная программа" }
        }
      }
    ],
    "tags": ["uuid-tag1"],
    "created_by": "uuid-user",
    "created_at": "2026-05-14T10:00:00Z",
    "updated_at": "2026-05-14T10:00:00Z"
  },
  "error": null
}
```

**POST /api/v1/questions/:id/status — Request**
```json
{ "status": "active" }
```

**POST /api/v1/questions/:id/status — Response (200)**
```json
{
  "data": { "id": "uuid-q1", "status": "active" },
  "error": null
}
```

**POST /api/v1/questions/:id/status — Invalid Transition (422)**
```json
{
  "data": null,
  "error": {
    "code": "ERR_INVALID_TRANSITION",
    "message": "Cannot transition from 'archived' to 'active'"
  }
}
```

**PUT /api/v1/questions/:id (draft or review status) — Response (200)**
```json
{
  "data": {
    "id": "uuid-q1",
    "status": "draft",
    "type": "single",
    "difficulty": "medium",
    "category_id": "uuid-cat",
    "default_locale": "kk",
    "version": 1,
    "parent_id": null,
    "locale_coverage": ["kk"],
    "updated_at": "2026-05-14T11:00:00Z"
  },
  "error": null
}
```

**PUT /api/v1/questions/:id (active question) — Response (200)**
```json
{
  "data": {
    "id": "new-version-uuid",
    "version": 2,
    "parent_id": "uuid-q1",
    "status": "draft",
    "note": "New version created; previous version archived"
  },
  "error": null
}
```

**GET /api/v1/questions/:id/versions — Response (200)**
```json
{
  "data": {
    "versions": [
      { "id": "uuid-q1", "version": 1, "status": "archived", "created_at": "2026-04-01T00:00:00Z", "created_by": "uuid-user" },
      { "id": "new-version-uuid", "version": 2, "status": "draft", "created_at": "2026-05-14T10:00:00Z", "created_by": "uuid-user" }
    ]
  },
  "error": null
}
```

**POST /api/v1/questions/:id/tags — Request**
```json
{ "tag_id": "uuid-tag2" }
```

**POST /api/v1/questions/:id/tags — Response (200)**
```json
{
  "data": { "question_id": "uuid-q1", "tag_ids": ["uuid-tag1", "uuid-tag2"] },
  "error": null
}
```

## Notes
- The `locale_coverage` field is computed by a single aggregation query (`SELECT locale FROM question_translations WHERE question_id = $1`) and appended by the service layer — no additional round-trips.
- The auto-versioning logic in `PUT` must run inside a serializable transaction to prevent two concurrent editors from creating duplicate versions.
- `stem_preview` in list responses is limited to 120 characters of the default locale stem to avoid large payloads; the full stem is available in the detail endpoint.
- The `locale_missing` filter parameter (`?locale_missing=ru`) is implemented as `NOT EXISTS (SELECT 1 FROM question_translations WHERE question_id = q.id AND locale = $locale)` to identify untranslated questions efficiently.
- **Migration (HIGH-4)**: A new migration `010` must add `CREATE INDEX idx_question_tags_tag_id ON question_tags(tag_id)` as a prerequisite for AC-10 tag-filter performance. The corresponding down migration drops this index.
- **Version history pagination**: `GET /api/v1/questions/:id/versions` intentionally returns `data: { versions: [...] }` with no pagination meta. Version chains are bounded by application-level update frequency (a question accumulates versions only through explicit edits), keeping the response size predictable. Pagination may be added in a future FR if operational data shows chains exceeding a practical threshold.

### Service Interface Extensions

The following methods are added to / replace existing methods on the `QuestionService` interface in `internal/questions`:

```go
// ListFiltered replaces the previous ListQuestions(ctx, categoryID) method.
ListFiltered(ctx context.Context, filter ListQuestionsFilter) ([]QuestionSummary, int, error)

// GetQuestionWithDetails returns full question including translations, options, and tags.
GetQuestionWithDetails(ctx context.Context, id uuid.UUID) (*QuestionDetail, error)

// UpdateQuestion applies an in-place update (draft/review) or triggers auto-versioning (active).
UpdateQuestion(ctx context.Context, id uuid.UUID, req UpdateQuestionRequest) error

// TransitionStatus enforces the status state machine and records the audit entry.
TransitionStatus(ctx context.Context, id uuid.UUID, targetStatus string, actorID uuid.UUID) error

// DeleteQuestion hard-deletes a draft question (returns ErrNotDraft if not in draft status).
DeleteQuestion(ctx context.Context, id uuid.UUID) error

// ListVersions returns the full version chain for a question ordered oldest-first.
ListVersions(ctx context.Context, id uuid.UUID) ([]QuestionVersion, error)
```

## Out of Scope

- Translation upsert for existing questions (FR-BB24).
- Bulk import/export of question banks (FR-BB25).
- Frontend question editor UI (FR-BB26/FR-BB27).
- AI-assisted question generation (Phase 7).

## Test Strategy

- **Unit tests** (`internal/questions`): table-driven tests covering AC-2 validation rules for every question type (`single`, `multiple`, `truefalse`, `likert`, `shorttext`); state-machine transition rules for AC-5 (all valid and invalid transitions).
- **Integration tests** (`internal/questions`): serializable transaction correctness for `PUT` auto-versioning under concurrent requests (AC-4); tag idempotency for AC-8.
- **Repository tests**: verify `ListFiltered` uses the `idx_question_tags_tag_id` index and produces no N+1 queries (AC-10).
- **Handler tests**: 401/403 response for unauthenticated/unauthorised callers; 422 response shape for validation failures.

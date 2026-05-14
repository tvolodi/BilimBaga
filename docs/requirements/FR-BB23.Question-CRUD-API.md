# FR-BB23 — Question CRUD API

## Metadata
| Field | Value |
|-------|-------|
| ID | FR-BB23 |
| Phase | 2 — Content Management |
| Priority | 1 |
| Status | Draft |
| Depends On | FR-BB22 |

## Description
Exposes a complete REST API for creating, reading, updating, and managing the lifecycle of questions in the question bank. Endpoints cover single-question operations, paginated/filtered listing, status workflow transitions, hard deletion of drafts, version history retrieval, and tag association management. All mutating operations are restricted to the `examiner` role and above; the response always includes a `locale_coverage` computed field indicating which locales have translations.

## Acceptance Criteria
- [ ] AC-1: `GET /api/v1/questions` returns a paginated response with `items`, `total`, `page`, and `per_page`; it accepts query parameters `category_id`, `tag_id`, `difficulty`, `type`, `status`, `locale`, and `locale_missing` (returns questions lacking translation for a given locale); unauthorized requests receive `401`.
- [ ] AC-2: `POST /api/v1/questions` creates a question in `draft` status; it requires `default_locale` stem and at least 2 answer options for `single`, `multiple`, and `truefalse` types; creating `shorttext` with zero options is valid; validation failures return `422` with a per-field error array.
- [ ] AC-3: `GET /api/v1/questions/:id` returns the full question including all `question_translations`, all `answer_options` with their `answer_translations`, associated tag IDs, and the computed `locale_coverage` array; returns `404` if the question does not exist.
- [ ] AC-4: `PUT /api/v1/questions/:id` — if the question's current status is `active`, the service atomically creates a new version (incremented `version`, `parent_id` pointing to the current row, status `draft`), archives the current row, and returns the new question ID in the response; if status is `draft` or `review`, it updates in place.
- [ ] AC-5: `POST /api/v1/questions/:id/status` enforces the workflow: `draft → review → active`; `active → archived`; any other transition returns `422` with error code `ERR_INVALID_TRANSITION`; transition to `active` is blocked if the default locale translation is missing.
- [ ] AC-6: `DELETE /api/v1/questions/:id` succeeds only when status is `draft`; attempting to delete a non-draft question returns `409` with error code `ERR_NOT_DRAFT`; deletion is a hard delete (CASCADE removes translations and answer options).
- [ ] AC-7: `GET /api/v1/questions/:id/versions` returns the full version chain ordered oldest-first, each entry containing `id`, `version`, `status`, `created_at`, and `created_by`.
- [ ] AC-8: `POST /api/v1/questions/:id/tags` and `DELETE /api/v1/questions/:id/tags/:tagId` add/remove tag associations; adding a non-existent tag ID returns `404`; adding a duplicate is idempotent (no error).
- [ ] AC-9: Every mutating endpoint records an audit log entry with `entity_type = 'question'`, the question `id`, the `action` performed, the `actor_id`, and a JSON diff of changed fields.
- [ ] AC-10: All list and detail endpoints respond within 500 ms at p95 under a question bank of 10,000 rows (query must use indexed columns and avoid N+1 queries for translations).

## Technical Specification

### API Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/v1/questions` | examiner+ | Paginated filtered question list |
| POST | `/api/v1/questions` | examiner+ | Create question in draft |
| GET | `/api/v1/questions/:id` | examiner+ | Full question detail |
| PUT | `/api/v1/questions/:id` | examiner+ | Update; auto-versions active questions |
| POST | `/api/v1/questions/:id/status` | examiner+ | Transition status |
| DELETE | `/api/v1/questions/:id` | examiner+ | Hard delete draft |
| GET | `/api/v1/questions/:id/versions` | examiner+ | Version history |
| POST | `/api/v1/questions/:id/tags` | examiner+ | Add tag association |
| DELETE | `/api/v1/questions/:id/tags/:tagId` | examiner+ | Remove tag association |

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
        "tags": ["uuid-tag1"],
        "created_by": "uuid-user",
        "created_at": "2026-05-14T10:00:00Z",
        "updated_at": "2026-05-14T10:00:00Z"
      }
    ],
    "total": 1,
    "page": 1,
    "per_page": 20
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
  "data": [
    { "id": "uuid-q1", "version": 1, "status": "archived", "created_at": "2026-04-01T00:00:00Z", "created_by": "uuid-user" },
    { "id": "new-version-uuid", "version": 2, "status": "draft", "created_at": "2026-05-14T10:00:00Z", "created_by": "uuid-user" }
  ],
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

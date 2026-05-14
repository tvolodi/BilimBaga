# FR-BB32 — Exam Configuration API

## Metadata
| Field | Value |
|-------|-------|
| ID | FR-BB32 |
| Phase | 3 — Exam Engine |
| Priority | 1 |
| Status | Draft |
| Depends On | FR-BB31 |

## Description
Provides the full REST API surface for creating, reading, updating, and publishing exam configurations. Includes lifecycle transitions (draft → active → archived), section management, question rule management, and manual question assignment. All mutating operations are restricted to draft-status exams except for archive and assignment endpoints.

## Acceptance Criteria
- [ ] AC-1: `GET /api/v1/exams` returns paginated exam list filterable by `status`; accessible to users with role `examiner` or higher.
- [ ] AC-2: `POST /api/v1/exams` creates a new exam with `status = 'draft'`; returns the created exam object with HTTP 201.
- [ ] AC-3: `PUT /api/v1/exams/:id` returns HTTP 409 if the exam `status` is not `'draft'`.
- [ ] AC-4: `POST /api/v1/exams/:id/publish` validates that every question rule can be satisfied (sufficient active questions matching category/tags/difficulty exist); returns HTTP 422 with a structured error listing unsatisfied rules if validation fails.
- [ ] AC-5: `POST /api/v1/exams/:id/archive` transitions status from `active` or `draft` to `archived`; subsequent `POST /sessions` against this exam returns HTTP 403.
- [ ] AC-6: `DELETE /api/v1/exams/:id` is only allowed on draft exams; returns HTTP 404 for non-existent, HTTP 409 for non-draft.
- [ ] AC-7: Section CRUD endpoints maintain `sort_order` uniqueness per exam and return 404 if section does not belong to the specified exam.
- [ ] AC-8: `PUT /api/v1/exams/:id/rules/:rId/questions` fully replaces the manual question list for a manual-mode rule; returns HTTP 400 if called on a random-mode rule.
- [ ] AC-9: All endpoints require a valid JWT; role checks return HTTP 403 (not 401) when authenticated but unauthorised.
- [ ] AC-10: Publish endpoint writes an audit log entry `{ action: "exam.publish", entity_type: "exam", entity_id }` on successful transition.

## Technical Specification

### API Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/v1/exams` | examiner+ | List exams with optional `?status=` filter |
| POST | `/api/v1/exams` | examiner+ | Create exam (draft) |
| GET | `/api/v1/exams/:id` | examiner+ | Full exam config with sections and rules |
| PUT | `/api/v1/exams/:id` | examiner+ | Update exam config (draft only) |
| POST | `/api/v1/exams/:id/publish` | examiner+ | Transition draft → active |
| POST | `/api/v1/exams/:id/archive` | examiner+ | Transition to archived |
| DELETE | `/api/v1/exams/:id` | examiner+ | Hard delete (draft only) |
| POST | `/api/v1/exams/:id/sections` | examiner+ | Add a section |
| PUT | `/api/v1/exams/:id/sections/:sId` | examiner+ | Update section title/sort_order |
| DELETE | `/api/v1/exams/:id/sections/:sId` | examiner+ | Remove section |
| POST | `/api/v1/exams/:id/rules` | examiner+ | Add a question rule |
| PUT | `/api/v1/exams/:id/rules/:rId` | examiner+ | Update question rule |
| DELETE | `/api/v1/exams/:id/rules/:rId` | examiner+ | Remove question rule |
| PUT | `/api/v1/exams/:id/rules/:rId/questions` | examiner+ | Replace manual question list |

#### Request / Response Shapes

**POST /api/v1/exams — Request**
```json
{
  "title": "Go Developer Certification",
  "description": "Tests core Go language proficiency.",
  "time_limit_minutes": 90,
  "passing_score_pct": 70.0,
  "max_attempts": 2,
  "available_from": "2026-06-01T00:00:00Z",
  "available_until": "2026-06-30T23:59:59Z",
  "shuffle_questions": true,
  "shuffle_options": true,
  "show_answers": "after_completion",
  "on_tab_switch": "warn",
  "certificate_enabled": true
}
```

**POST /api/v1/exams — Response (201)**
```json
{
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "title": "Go Developer Certification",
    "status": "draft",
    "time_limit_minutes": 90,
    "passing_score_pct": 70.0,
    "max_attempts": 2,
    "available_from": "2026-06-01T00:00:00Z",
    "available_until": "2026-06-30T23:59:59Z",
    "shuffle_questions": true,
    "shuffle_options": true,
    "show_answers": "after_completion",
    "on_tab_switch": "warn",
    "certificate_enabled": true,
    "created_by": "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
    "created_at": "2026-05-14T10:00:00Z",
    "updated_at": "2026-05-14T10:00:00Z"
  },
  "error": null
}
```

**GET /api/v1/exams/:id — Response (200)**
```json
{
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "title": "Go Developer Certification",
    "status": "draft",
    "time_limit_minutes": 90,
    "passing_score_pct": 70.0,
    "max_attempts": 2,
    "available_from": "2026-06-01T00:00:00Z",
    "available_until": "2026-06-30T23:59:59Z",
    "shuffle_questions": true,
    "shuffle_options": false,
    "show_answers": "after_completion",
    "on_tab_switch": "warn",
    "certificate_enabled": true,
    "sections": [
      { "id": "sec-uuid-1", "title": "Core Language", "sort_order": 0 }
    ],
    "rules": [
      {
        "id": "rule-uuid-1",
        "section_id": "sec-uuid-1",
        "mode": "random",
        "category_id": "cat-uuid-1",
        "tag_ids": ["tag-uuid-a"],
        "difficulty": "medium",
        "count": 10,
        "sort_order": 0
      }
    ]
  },
  "error": null
}
```

**POST /api/v1/exams/:id/publish — Response (200)**
```json
{
  "data": { "id": "550e8400-e29b-41d4-a716-446655440000", "status": "active" },
  "error": null
}
```

**POST /api/v1/exams/:id/publish — Validation Failure (422)**
```json
{
  "data": null,
  "error": {
    "code": "EXAM_RULES_UNSATISFIED",
    "message": "One or more question rules cannot be satisfied.",
    "details": [
      {
        "rule_id": "rule-uuid-1",
        "required": 10,
        "available": 3,
        "filter": { "category_id": "cat-uuid-1", "difficulty": "medium" }
      }
    ]
  }
}
```

**PUT /api/v1/exams/:id/rules/:rId/questions — Request**
```json
{
  "question_ids": [
    { "question_id": "q-uuid-1", "sort_order": 0 },
    { "question_id": "q-uuid-2", "sort_order": 1 }
  ]
}
```

**Error — Draft Required (409)**
```json
{
  "data": null,
  "error": {
    "code": "EXAM_NOT_DRAFT",
    "message": "This operation is only allowed on exams in draft status."
  }
}
```

## Notes
- The publish validation query should join `questions`, `question_options`, and apply category/tag/difficulty filters to count active questions; this query must run inside a transaction to avoid TOCTOU races.
- Archiving an exam does not affect existing in-progress sessions; they continue until submitted or auto-submitted.
- Section deletion cascades rule nullification of `section_id` (SET NULL) rather than deleting the rules.
- `GET /api/v1/exams` supports `?status=draft`, `?status=active`, `?status=archived`; multiple values may be supported via repeated query params.
- Pagination: `?page=1&page_size=20`; response includes `total_count` in the `data` envelope.

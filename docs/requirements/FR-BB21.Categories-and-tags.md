# FR-BB21 — Categories and Tags

## Metadata
| Field | Value |
|-------|-------|
| ID | FR-BB21 |
| Phase | 2 — Content Management |
| Priority | 1 |
| Status | uat-verified |
| Depends On | FR-BB16 |

## Description
Provides the taxonomy layer for the question bank: a hierarchical category tree (with track assignment) and a flat tag vocabulary. Categories map questions to built-in compliance tracks (security, safety, loyalty/values) and custom client-defined areas. Tags allow cross-cutting labels that span categories. Both are managed by examiners and above through a dedicated CRUD API, while the full tree is readable by any authenticated user.

## Acceptance Criteria
- [ ] AC-1: A `categories` table exists with columns `id` (UUID PK), `name` (TEXT NOT NULL), `parent_id` (UUID nullable FK → categories), `track` (TEXT), `sort_order` (INT DEFAULT 0), `created_at`, `updated_at`; cyclic parent references are rejected at the database level via a constraint or application check.
- [ ] AC-2: Three built-in root categories are seeded by migration: *Security Awareness*, *Workplace Safety*, and *Loyalty & Values*, each with the corresponding `track` values `security`, `safety`, `loyalty`; seeded rows are re-entrant (migration is idempotent).
- [ ] AC-3: `GET /api/v1/categories` returns the full category tree as nested JSON, requires a valid JWT, and responds in < 200 ms on a warm cache.
- [ ] AC-4: `POST`, `PUT`, and `DELETE` on `/api/v1/categories[/:id]` require role `examiner` or above; attempts by lower roles receive `403 Forbidden` with error code `ERR_FORBIDDEN`.
- [ ] AC-5: `DELETE /api/v1/categories/:id` is rejected with `409 Conflict` (error code `ERR_CATEGORY_IN_USE`) when the category has child categories or when any question references it.
- [ ] AC-6: A `tags` table exists with columns `id` (UUID PK), `name` (TEXT UNIQUE NOT NULL), `created_at`; `GET /api/v1/tags` returns a flat list sorted alphabetically and is accessible to `examiner` and above. Each row includes `usage_count` — the number of `question_tags` rows referencing the tag (zero when `question_tags` does not yet exist).
- [ ] AC-7: `POST /api/v1/tags` creates a tag (name trimmed, lowercased, max 64 chars); returns `409 Conflict` (error code `ERR_TAG_DUPLICATE`) if the name already exists.
- [ ] AC-8: `PUT /api/v1/tags/:id` renames a tag in place so existing `question_tags` rows continue to reference it; accepts `{ "name": "<new-name>" }`, applies the same trim/lowercase/length rules as create; returns `404 Not Found`, `400` on invalid name, and `409 Conflict` (`ERR_TAG_DUPLICATE`) on collision.
- [ ] AC-9: `DELETE /api/v1/tags/:id` is rejected with `409 Conflict` (error code `ERR_TAG_IN_USE`) when any question references the tag via the `question_tags` join table.
- [ ] AC-10: All mutating endpoints emit an audit log entry (table `audit_log`) with `entity_type`, `entity_id`, `action`, `actor_id`, and `diff` (JSON patch of changed fields). Tag mutations emit `tag.create`, `tag.update`, and `tag.delete` action codes.
- [ ] AC-11: All user-visible API error messages are keyed (not free-form strings) so the frontend can translate them.

## Technical Specification

### Database Schema

```sql
-- Categories (hierarchical)
CREATE TABLE categories (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT NOT NULL,
    parent_id   UUID REFERENCES categories(id) ON DELETE RESTRICT,
    track       TEXT,                     -- 'security' | 'safety' | 'loyalty' | NULL (custom)
    sort_order  INT  NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_categories_parent_id ON categories(parent_id);

-- Tags (flat vocabulary)
CREATE TABLE tags (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name       TEXT UNIQUE NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Seed: built-in root categories (idempotent)
INSERT INTO categories (id, name, track, sort_order)
VALUES
    ('00000000-0000-0000-0000-000000000001', 'Security Awareness', 'security', 1),
    ('00000000-0000-0000-0000-000000000002', 'Workplace Safety',   'safety',   2),
    ('00000000-0000-0000-0000-000000000003', 'Loyalty & Values',   'loyalty',  3)
ON CONFLICT (id) DO NOTHING;
```

### API Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/v1/categories` | Any authenticated | Full nested category tree |
| POST | `/api/v1/categories` | examiner+ | Create a new category |
| PUT | `/api/v1/categories/:id` | examiner+ | Update name, parent, track, or sort_order |
| DELETE | `/api/v1/categories/:id` | examiner+ | Delete leaf category with no questions |
| GET | `/api/v1/tags` | examiner+ | List all tags (alphabetical) with `usage_count` |
| POST | `/api/v1/tags` | examiner+ | Create a new tag |
| PUT | `/api/v1/tags/:id` | examiner+ | Rename a tag in place (preserves question links) |
| DELETE | `/api/v1/tags/:id` | examiner+ | Delete tag if not in use |

#### Request / Response Shapes

**GET /api/v1/categories**
```json
{
  "data": [
    {
      "id": "00000000-0000-0000-0000-000000000001",
      "name": "Security Awareness",
      "track": "security",
      "sort_order": 1,
      "children": [
        {
          "id": "uuid-child",
          "name": "Phishing",
          "track": "security",
          "sort_order": 1,
          "children": []
        }
      ]
    }
  ],
  "error": null
}
```

**POST /api/v1/categories — Request**
```json
{
  "name": "Password Management",
  "parent_id": "00000000-0000-0000-0000-000000000001",
  "track": "security",
  "sort_order": 2
}
```

**POST /api/v1/categories — Response (201)**
```json
{
  "data": {
    "id": "new-uuid",
    "name": "Password Management",
    "parent_id": "00000000-0000-0000-0000-000000000001",
    "track": "security",
    "sort_order": 2,
    "created_at": "2026-05-14T10:00:00Z",
    "updated_at": "2026-05-14T10:00:00Z"
  },
  "error": null
}
```

**DELETE /api/v1/categories/:id — Conflict Response (409)**
```json
{
  "data": null,
  "error": {
    "code": "ERR_CATEGORY_IN_USE",
    "message": "Category has child categories or referenced questions"
  }
}
```

**GET /api/v1/tags**
```json
{
  "data": [
    { "id": "uuid-1", "name": "gdpr",     "created_at": "2026-05-14T10:00:00Z", "usage_count": 12 },
    { "id": "uuid-2", "name": "iso27001", "created_at": "2026-05-14T10:00:00Z", "usage_count": 0  }
  ],
  "error": null
}
```

**POST /api/v1/tags — Request**
```json
{ "name": "GDPR" }
```

**POST /api/v1/tags — Response (201)**
```json
{
  "data": { "id": "new-uuid", "name": "gdpr", "created_at": "2026-05-14T10:00:00Z", "usage_count": 0 },
  "error": null
}
```

**PUT /api/v1/tags/:id — Request**
```json
{ "name": "GDPR-2024" }
```

**PUT /api/v1/tags/:id — Response (200)**
```json
{
  "data": { "id": "uuid-1", "name": "gdpr-2024", "created_at": "2026-05-14T10:00:00Z", "usage_count": 0 },
  "error": null
}
```

**PUT /api/v1/tags/:id — Conflict (409)**
```json
{
  "data": null,
  "error": { "code": "ERR_TAG_DUPLICATE", "message": "tag with this name already exists" }
}
```

## Notes
- `track` on categories is advisory metadata used by analytics (Phase 5) to group results by compliance domain; it does not drive access control.
- The frontend category picker (FR-BB26) expects the nested tree format from `GET /api/v1/categories` directly — no client-side tree reconstruction.
- Tag names are normalized to lowercase-trimmed on write (`tags.NormalizeName`); display capitalisation is handled by the frontend.
- `usage_count` on `GET /api/v1/tags` is computed via `LEFT JOIN (SELECT tag_id, COUNT(*) FROM question_tags GROUP BY tag_id)`; the handler probes `to_regclass('public.question_tags')` first and falls back to a constant `0` when the join table has not yet been provisioned (during early-phase deployments before FR-BB22 has run).
- Rename (`PUT /api/v1/tags/:id`) is preferred over delete-and-recreate because it preserves every `question_tags` link; the frontend (FR-BB29) exposes only Rename, never delete-then-recreate.
- Cyclic parent detection: before inserting/updating `parent_id`, walk ancestors in application code and reject if the new parent is a descendant of the current node.
- Pagination exemption — `GET /api/v1/categories`: This endpoint returns the full category tree as a nested structure and is intentionally exempt from the `meta: { page, per_page, total }` pagination envelope. The category tree is a bounded, administrator-maintained vocabulary (expected to remain in the tens-to-hundreds of nodes); returning it in full on every call avoids the complexity of paginating a recursive structure and matches the frontend's requirement (FR-BB26) to receive the complete tree in a single request.
- Pagination exemption — `GET /api/v1/tags`: This endpoint returns the full flat tag list and is intentionally exempt from the `meta: { page, per_page, total }` pagination envelope. Tags form a small, bounded vocabulary that is loaded once per session for autocomplete inputs; delivering the full list unbounded simplifies client-side filtering without meaningful performance impact.

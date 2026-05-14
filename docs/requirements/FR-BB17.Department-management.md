# FR-BB17 — Department Management

## Metadata
| Field | Value |
|-------|-------|
| ID | FR-BB17 |
| Phase | 1 — Foundation |
| Priority | 2 |
| Status | Draft |
| Depends On | FR-BB16 |

## Description
Provides the hierarchical department structure that organises users across the organisation. Departments can be nested (a department may have a `parent_id`) to reflect real org-chart relationships. CRUD endpoints let super admins maintain the department tree; the list endpoint returns the full tree rather than a flat list so the frontend can render it without a secondary aggregation step.

## Acceptance Criteria
- [ ] AC-1: Migration creates the `departments` table with `id UUID PK`, `name TEXT NOT NULL`, `parent_id UUID NULLABLE REFERENCES departments(id)`, and `created_at TIMESTAMPTZ NOT NULL DEFAULT now()`.
- [ ] AC-2: `GET /api/v1/departments` returns the full department tree as nested JSON (each node contains an optional `children` array); accessible to any `admin`-level role.
- [ ] AC-3: `POST /api/v1/departments` creates a new department; `parent_id` is optional (null = top-level); accessible to `super_admin` only.
- [ ] AC-4: `PUT /api/v1/departments/:id` updates only the department's `name`; accessible to `super_admin` only.
- [ ] AC-5: `DELETE /api/v1/departments/:id` returns `409 Conflict` if any user in the `users` table has `department_id = :id`; accessible to `super_admin` only.
- [ ] AC-6: `DELETE /api/v1/departments/:id` returns `409 Conflict` if the department has any child departments.
- [ ] AC-7: Creating a department with a non-existent `parent_id` returns `404 Not Found`.
- [ ] AC-8: Department names must be unique within the same parent (i.e. two departments under the same parent cannot share a name); violation returns `409 Conflict` with code `DUPLICATE_NAME`.
- [ ] AC-9: All mutating operations (create, rename, delete) are written to `audit_log`.

## Technical Specification

### Database Schema

```sql
-- Migration: 005_departments.up.sql

CREATE TABLE departments (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name       TEXT        NOT NULL,
    parent_id  UUID        REFERENCES departments(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (parent_id, name)   -- unique name per parent (NULLs are not equal, so top-level duplicates prevented by partial index below)
);

-- Unique names among top-level departments (where parent_id IS NULL)
CREATE UNIQUE INDEX idx_departments_top_level_name
    ON departments (name)
    WHERE parent_id IS NULL;

CREATE INDEX idx_departments_parent_id ON departments(parent_id);
```

```sql
-- Migration: 005_departments.down.sql
DROP TABLE IF EXISTS departments;
```

### API Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/v1/departments` | admin roles | Full tree |
| POST | `/api/v1/departments` | super_admin | Create department |
| PUT | `/api/v1/departments/:id` | super_admin | Rename department |
| DELETE | `/api/v1/departments/:id` | super_admin | Delete empty department |

#### Request / Response shapes

```json
// GET /api/v1/departments — 200 OK
{
  "data": [
    {
      "id": "aaa00000-0000-0000-0000-000000000001",
      "name": "Engineering",
      "parent_id": null,
      "created_at": "2026-01-01T00:00:00Z",
      "children": [
        {
          "id": "aaa00000-0000-0000-0000-000000000002",
          "name": "Backend",
          "parent_id": "aaa00000-0000-0000-0000-000000000001",
          "created_at": "2026-01-02T00:00:00Z",
          "children": []
        }
      ]
    }
  ],
  "error": null
}
```

```json
// POST /api/v1/departments — request body
{
  "name": "QA",
  "parent_id": "aaa00000-0000-0000-0000-000000000001"
}

// 201 Created
{
  "data": {
    "id": "aaa00000-0000-0000-0000-000000000003",
    "name": "QA",
    "parent_id": "aaa00000-0000-0000-0000-000000000001",
    "created_at": "2026-05-14T10:00:00Z",
    "children": []
  },
  "error": null
}

// 404 — parent not found
{
  "data": null,
  "error": { "code": "NOT_FOUND", "message": "parent department not found" }
}

// 409 — duplicate name
{
  "data": null,
  "error": { "code": "DUPLICATE_NAME", "message": "a department with this name already exists under the same parent" }
}
```

```json
// PUT /api/v1/departments/:id — request body
{
  "name": "Quality Assurance"
}

// 200 OK
{
  "data": {
    "id": "aaa00000-0000-0000-0000-000000000003",
    "name": "Quality Assurance",
    "parent_id": "aaa00000-0000-0000-0000-000000000001",
    "created_at": "2026-05-14T10:00:00Z",
    "children": []
  },
  "error": null
}
```

```json
// DELETE /api/v1/departments/:id

// 204 No Content — success

// 409 — has assigned users
{
  "data": null,
  "error": { "code": "DEPARTMENT_NOT_EMPTY", "message": "cannot delete department with assigned users" }
}

// 409 — has child departments
{
  "data": null,
  "error": { "code": "DEPARTMENT_HAS_CHILDREN", "message": "cannot delete department with child departments" }
}
```

## Notes
- The tree is built in the Go service layer using a recursive algorithm over a flat SQL result set (a single query fetching all departments), avoiding N+1 queries. For very large organisations (hundreds of departments) a CTE recursive query may be preferred.
- `ON DELETE RESTRICT` on `parent_id` prevents orphaning children via raw SQL; the API enforces the same constraint with a nicer error message before the FK is checked.
- Department renaming does not cascade any changes to users or exams — department names are display-only references.
- A future phase may add `manager_id` (FK to users) and `description` columns; those additions will be new migrations.

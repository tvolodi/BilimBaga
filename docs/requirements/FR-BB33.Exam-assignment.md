# FR-BB33 — Exam Assignment

## Metadata
| Field | Value |
|-------|-------|
| ID | FR-BB33 |
| Phase | 3 — Exam Engine |
| Priority | 1 |
| Status | Implemented |
| Depends On | FR-BB32 |

## Description
Enables super admins and department admins to assign active exams to individual users, entire departments (including sub-departments), or all active users in the organisation. Tracks deadlines per assignment and exposes completion statistics per assignee group.

## Acceptance Criteria
- [ ] AC-1: `POST /api/v1/exams/:id/assign` creates an `exam_assignments` record; returns HTTP 409 if an identical assignment (same exam, assignee_type, assignee_id) already exists.
- [ ] AC-2: Assignment to `assignee_type = 'department'` implicitly covers all users in that department and all sub-departments at session-start time (not materialised at assignment time).
- [ ] AC-3: Assignment to `assignee_type = 'all'` implicitly covers all users with `status = 'active'` at session-start time.
- [ ] AC-4: `GET /api/v1/exams/:id/assignments` returns each assignment with `total_users`, `completed_count`, and `passed_count` computed dynamically from `exam_sessions`.
- [ ] AC-5: `DELETE /api/v1/exams/:id/assign/:assignmentId` removes the assignment record; existing in-progress or completed sessions are not affected.
- [ ] AC-6: Only super admins may assign to `assignee_type = 'all'` or to departments outside their own; department admins may only assign within their managed department.
- [ ] AC-7: Assigning to a non-active exam returns HTTP 422 with code `EXAM_NOT_ACTIVE`.
- [ ] AC-8: `deadline` field is optional; if provided, it must be in the future at the time of assignment creation.
- [ ] AC-9: An audit log entry is written for every successful assignment and de-assignment.
- [ ] AC-10: The endpoint returns HTTP 404 if the exam does not exist, HTTP 403 if the caller lacks permission, and HTTP 400 for malformed request bodies.

## Technical Specification

### Database Schema

```sql
CREATE TYPE assignee_type AS ENUM ('user', 'department', 'all');

CREATE TABLE exam_assignments (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    exam_id         UUID NOT NULL REFERENCES exams(id) ON DELETE CASCADE,
    assignee_type   assignee_type NOT NULL,
    assignee_id     UUID,  -- NULL when assignee_type = 'all'
    deadline        TIMESTAMPTZ,
    assigned_by     UUID NOT NULL REFERENCES users(id),
    assigned_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT exam_assignments_unique UNIQUE (exam_id, assignee_type, assignee_id),
    CONSTRAINT exam_assignments_id_required CHECK (
        assignee_type = 'all' OR assignee_id IS NOT NULL
    )
);

CREATE INDEX idx_exam_assignments_exam_id ON exam_assignments(exam_id);
CREATE INDEX idx_exam_assignments_assignee ON exam_assignments(assignee_type, assignee_id);
```

### API Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/api/v1/exams/:id/assign` | admin+ | Create assignment |
| DELETE | `/api/v1/exams/:id/assign/:assignmentId` | admin+ | Remove assignment |
| GET | `/api/v1/exams/:id/assignments` | examiner+ | List assignments with stats |

#### Request / Response Shapes

**POST /api/v1/exams/:id/assign — Request**
```json
{
  "assignee_type": "department",
  "assignee_id": "dept-uuid-1",
  "deadline": "2026-07-01T23:59:59Z"
}
```

**POST /api/v1/exams/:id/assign — Response (201)**
```json
{
  "data": {
    "id": "assign-uuid-1",
    "exam_id": "exam-uuid-1",
    "assignee_type": "department",
    "assignee_id": "dept-uuid-1",
    "deadline": "2026-07-01T23:59:59Z",
    "assigned_by": "user-uuid-admin",
    "assigned_at": "2026-05-14T10:00:00Z"
  },
  "error": null
}
```

**GET /api/v1/exams/:id/assignments — Response (200)**
```json
{
  "data": [
    {
      "id": "assign-uuid-1",
      "assignee_type": "department",
      "assignee_id": "dept-uuid-1",
      "assignee_name": "Engineering",
      "deadline": "2026-07-01T23:59:59Z",
      "assigned_at": "2026-05-14T10:00:00Z",
      "stats": {
        "total_users": 42,
        "completed_count": 18,
        "passed_count": 15
      }
    },
    {
      "id": "assign-uuid-2",
      "assignee_type": "user",
      "assignee_id": "user-uuid-x",
      "assignee_name": "Alice Smith",
      "deadline": null,
      "assigned_at": "2026-05-14T11:00:00Z",
      "stats": {
        "total_users": 1,
        "completed_count": 1,
        "passed_count": 1
      }
    }
  ],
  "error": null
}
```

**Error — Exam Not Active (422)**
```json
{
  "data": null,
  "error": {
    "code": "EXAM_NOT_ACTIVE",
    "message": "Exams must be in active status before they can be assigned."
  }
}
```

**Error — Duplicate Assignment (409)**
```json
{
  "data": null,
  "error": {
    "code": "ASSIGNMENT_ALREADY_EXISTS",
    "message": "This exam is already assigned to the specified target."
  }
}
```

## Notes
- The `completed_count` stat counts sessions with `status IN ('submitted', 'auto_submitted')` where `score_pct IS NOT NULL`; `passed_count` counts those with `passed = TRUE`.
- Sub-department traversal for `assignee_type = 'department'` is performed using a recursive CTE on the `departments` table at session-start time (FR-BB35), not materialised here.
- `assignee_name` in the list response is resolved by joining `departments` or `users` based on `assignee_type`.
- Department admins are validated against their `managed_department_ids` stored in the user record or a separate admin-department mapping table.

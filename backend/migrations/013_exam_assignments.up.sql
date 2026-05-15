-- FR-BB33: Exam Assignment

CREATE TYPE assignee_type AS ENUM ('user', 'department', 'all');

CREATE TABLE exam_assignments (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    exam_id         UUID NOT NULL REFERENCES exams(id) ON DELETE CASCADE,
    assignee_type   assignee_type NOT NULL,
    assignee_id     UUID,
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

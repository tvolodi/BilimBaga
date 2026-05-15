-- FR-BB42: Manual Grading Queue
-- Add manual grading columns to session_question_scores and seed grading permissions.

-- AC-10: persist graded_by, graded_at, manual_feedback per answer
ALTER TABLE session_question_scores
  ADD COLUMN IF NOT EXISTS manual_feedback TEXT        NULL,
  ADD COLUMN IF NOT EXISTS graded_by       UUID        NULL REFERENCES users(id),
  ADD COLUMN IF NOT EXISTS graded_at       TIMESTAMPTZ NULL;

-- AC-9: seed grading:read and grading:write permissions
INSERT INTO permissions (id, resource, action) VALUES
  (gen_random_uuid(), 'grading', 'read'),
  (gen_random_uuid(), 'grading', 'write')
ON CONFLICT (resource, action) DO NOTHING;

-- AC-9: grant to examiner, department_admin, super_admin
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.name IN ('examiner', 'department_admin', 'super_admin')
  AND p.resource = 'grading'
ON CONFLICT DO NOTHING;

-- Migration 034: role management (FR-BB117, issue #137).
-- Adds roles.description / roles.is_system, protects the four built-in roles,
-- and seeds roles:read + roles:manage (super_admin only). Idempotent.

ALTER TABLE roles ADD COLUMN IF NOT EXISTS description TEXT    NOT NULL DEFAULT '';
ALTER TABLE roles ADD COLUMN IF NOT EXISTS is_system   BOOLEAN NOT NULL DEFAULT false;

UPDATE roles SET is_system = true
 WHERE name IN ('super_admin', 'department_admin', 'examiner', 'employee');

INSERT INTO permissions (resource, action) VALUES
    ('roles', 'read'),
    ('roles', 'manage')
ON CONFLICT (resource, action) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
  FROM roles r
  JOIN permissions p ON p.resource = 'roles' AND p.action IN ('read', 'manage')
 WHERE r.name = 'super_admin'
ON CONFLICT DO NOTHING;

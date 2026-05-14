-- Migration: 006_departments.up.sql
-- Adds parent_id and supporting constraints to the departments table (FR-BB17).
-- Also seeds RBAC permissions for the departments resource.

ALTER TABLE departments
    ADD COLUMN parent_id UUID REFERENCES departments(id) ON DELETE RESTRICT;

ALTER TABLE departments
    ADD CONSTRAINT departments_parent_id_name_unique UNIQUE (parent_id, name);

-- Unique names among top-level departments (where parent_id IS NULL).
CREATE UNIQUE INDEX idx_departments_top_level_name
    ON departments (name)
    WHERE parent_id IS NULL;

CREATE INDEX idx_departments_parent_id ON departments(parent_id);

-- Seed RBAC permissions for departments resource.
INSERT INTO permissions (resource, action) VALUES
    ('departments', 'read'),
    ('departments', 'manage')
ON CONFLICT (resource, action) DO NOTHING;

-- super_admin: already has all permissions via blanket insert in 005, but ensure via explicit seed.
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r, permissions p
WHERE r.name = 'super_admin'
  AND (p.resource, p.action) IN (('departments', 'read'), ('departments', 'manage'))
ON CONFLICT DO NOTHING;

-- department_admin: read only.
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON (p.resource, p.action) IN (('departments', 'read'))
WHERE r.name = 'department_admin'
ON CONFLICT DO NOTHING;

-- examiner: read only.
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON (p.resource, p.action) IN (('departments', 'read'))
WHERE r.name = 'examiner'
ON CONFLICT DO NOTHING;

-- Migration: 005_rbac.up.sql
-- Renames hr_admin → examiner, creates permissions & role_permissions, and seeds the full permission matrix.
-- NOTE: The roles table already exists (created by migration 003).

-- ── Rename hr_admin to examiner ───────────────────────────────────────────────
UPDATE roles SET name = 'examiner' WHERE name = 'hr_admin';

-- ── New tables ────────────────────────────────────────────────────────────────
CREATE TABLE permissions (
    id       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    resource TEXT NOT NULL,
    action   TEXT NOT NULL,
    UNIQUE (resource, action)
);

CREATE TABLE role_permissions (
    role_id       UUID NOT NULL REFERENCES roles(id)       ON DELETE CASCADE,
    permission_id UUID NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, permission_id)
);

-- ── Seed permissions ─────────────────────────────────────────────────────────
INSERT INTO permissions (resource, action) VALUES
    ('users',     'read'),
    ('users',     'manage'),
    ('questions', 'read'),
    ('questions', 'write'),
    ('exams',     'read'),
    ('exams',     'write'),
    ('exams',     'assign'),
    ('reports',   'read'),
    ('portal',    'read'),
    ('portal',    'submit'),
    ('tenant',    'manage'),
    ('audit',     'read')
ON CONFLICT (resource, action) DO NOTHING;

-- ── super_admin: all permissions ─────────────────────────────────────────────
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r, permissions p
WHERE r.name = 'super_admin'
ON CONFLICT DO NOTHING;

-- ── department_admin ─────────────────────────────────────────────────────────
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON (p.resource, p.action) IN (
    ('users',     'read'),
    ('users',     'manage'),
    ('questions', 'read'),
    ('exams',     'read'),
    ('exams',     'assign'),
    ('reports',   'read')
)
WHERE r.name = 'department_admin'
ON CONFLICT DO NOTHING;

-- ── examiner ─────────────────────────────────────────────────────────────────
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON (p.resource, p.action) IN (
    ('questions', 'read'),
    ('questions', 'write'),
    ('exams',     'read'),
    ('exams',     'write'),
    ('exams',     'assign'),
    ('reports',   'read')
)
WHERE r.name = 'examiner'
ON CONFLICT DO NOTHING;

-- ── employee ─────────────────────────────────────────────────────────────────
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON (p.resource, p.action) IN (
    ('portal', 'read'),
    ('portal', 'submit')
)
WHERE r.name = 'employee'
ON CONFLICT DO NOTHING;

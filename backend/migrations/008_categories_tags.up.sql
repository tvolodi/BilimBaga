-- Migration 008: FR-BB21 — Categories and Tags
-- Hierarchical category tree (with track) + flat tag vocabulary.

CREATE TABLE IF NOT EXISTS categories (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT NOT NULL,
    parent_id   UUID REFERENCES categories(id) ON DELETE RESTRICT,
    track       TEXT,
    sort_order  INT  NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_categories_parent_id ON categories(parent_id);

CREATE TABLE IF NOT EXISTS tags (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name       TEXT UNIQUE NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Seed: built-in root categories (idempotent on id).
INSERT INTO categories (id, name, track, sort_order)
VALUES
    ('00000000-0000-0000-0000-000000000001', 'Security Awareness', 'security', 1),
    ('00000000-0000-0000-0000-000000000002', 'Workplace Safety',   'safety',   2),
    ('00000000-0000-0000-0000-000000000003', 'Loyalty & Values',   'loyalty',  3)
ON CONFLICT (id) DO NOTHING;

-- Permissions for categories and tags management (examiner+ can manage,
-- any authenticated user can read the tree via the dedicated public-list
-- endpoint without a permission check).
INSERT INTO permissions (resource, action) VALUES
    ('categories', 'manage'),
    ('tags',       'read'),
    ('tags',       'manage')
ON CONFLICT (resource, action) DO NOTHING;

-- Grant the new permissions to roles that should have them.
-- super_admin already gets every permission via the seed in migration 005,
-- but that seed has already run, so we must explicitly grant the new rows.
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r, permissions p
WHERE r.name = 'super_admin'
  AND (p.resource, p.action) IN (
      ('categories', 'manage'),
      ('tags',       'read'),
      ('tags',       'manage')
  )
ON CONFLICT DO NOTHING;

-- examiner: full CRUD on categories and tags.
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON (p.resource, p.action) IN (
    ('categories', 'manage'),
    ('tags',       'read'),
    ('tags',       'manage')
)
WHERE r.name = 'examiner'
ON CONFLICT DO NOTHING;

-- department_admin: read-only access to tags (already gets read-only access
-- to questions; consistent shape).
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON (p.resource, p.action) IN (
    ('tags', 'read')
)
WHERE r.name = 'department_admin'
ON CONFLICT DO NOTHING;

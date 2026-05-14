-- Migration: 006_departments.down.sql
-- Reverses 006_departments.up.sql.

DROP INDEX IF EXISTS idx_departments_parent_id;
DROP INDEX IF EXISTS idx_departments_top_level_name;
ALTER TABLE departments DROP CONSTRAINT IF EXISTS departments_parent_id_name_unique;
ALTER TABLE departments DROP COLUMN IF EXISTS parent_id;

-- Remove departments RBAC permissions.
DELETE FROM role_permissions
WHERE permission_id IN (
    SELECT id FROM permissions WHERE resource = 'departments'
);
DELETE FROM permissions WHERE resource = 'departments';

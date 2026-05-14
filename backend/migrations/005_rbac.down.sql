-- Migration: 005_rbac.down.sql
-- Reverses migration 005_rbac.up.sql.
-- NOTE: Do NOT drop the roles table — it is owned by migration 003.

UPDATE roles SET name = 'hr_admin' WHERE name = 'examiner';

DELETE FROM role_permissions;
DELETE FROM permissions;

DROP TABLE IF EXISTS role_permissions;
DROP TABLE IF EXISTS permissions;

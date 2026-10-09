-- Migration 034 down: revert role management.
-- Fails loudly (and rolls back) if any user still holds a custom role.
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM users u JOIN roles r ON r.id = u.role_id WHERE r.is_system = false
    ) THEN
        RAISE EXCEPTION 'cannot roll back 034: users are still assigned to custom roles';
    END IF;
END $$;

DELETE FROM roles WHERE is_system = false;

DELETE FROM role_permissions
 WHERE permission_id IN (SELECT id FROM permissions WHERE resource = 'roles');
DELETE FROM permissions WHERE resource = 'roles';

ALTER TABLE roles DROP COLUMN IF EXISTS is_system;
ALTER TABLE roles DROP COLUMN IF EXISTS description;

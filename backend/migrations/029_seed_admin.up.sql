-- Migration 029: Seed initial super_admin user.
-- Inserts a default admin account only when no users exist (idempotent).
-- Credentials: admin@bilimbaga.local / Admin1234!
-- force_password_change = true — the user must set a new password on first login.
INSERT INTO users (email, password_hash, full_name, role_id, status, force_password_change)
SELECT
    'admin@bilimbaga.local',
    '$2b$12$.b2VsbiluRqUkJuT4gpfPOGpE4TLD3RTNZtXjHFUcqrIgFNVxyKj2',
    'Super Admin',
    r.id,
    'active',
    true
FROM roles r
WHERE r.name = 'super_admin'
  AND NOT EXISTS (SELECT 1 FROM users);

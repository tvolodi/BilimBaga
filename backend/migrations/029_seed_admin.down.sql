-- Rollback 029: remove the seeded admin user.
DELETE FROM users WHERE email = 'admin@bilimbaga.local';

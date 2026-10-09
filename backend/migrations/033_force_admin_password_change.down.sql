-- Migration 033 down: restore the migration 030 state (force_password_change = false)
-- for the seeded admin, but only while it still has the default password hash.
UPDATE users
SET
    force_password_change = false,
    updated_at            = now()
WHERE email = 'admin@bilimbaga.local'
  AND password_hash = '$2a$12$v7BCGG1mNmqtooAWqav4LuoB5i4n5hWA06iMyYLT.jPRMA1B9Mbo6';

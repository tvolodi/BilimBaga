-- Migration 030: Reset super_admin password and unlock account.
-- Resets the admin@bilimbaga.local password to Admin1234! (bcrypt cost 12)
-- and clears any lockout state. Safe to run multiple times (idempotent).
UPDATE users
SET
    password_hash       = '$2a$12$v7BCGG1mNmqtooAWqav4LuoB5i4n5hWA06iMyYLT.jPRMA1B9Mbo6',
    force_password_change = false,
    failed_attempts     = 0,
    locked_until        = NULL,
    updated_at          = now()
WHERE email = 'admin@bilimbaga.local';

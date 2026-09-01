-- Migration 030 down: restore previous password hash (original migration 029 value).
-- After rolling back, admin@bilimbaga.local reverts to the migration 029 hash
-- (Admin1234! with force_password_change = true).
UPDATE users
SET
    password_hash       = '$2b$12$.b2VsbiluRqUkJuT4gpfPOGpE4TLD3RTNZtXjHFUcqrIgFNVxyKj2',
    force_password_change = true,
    failed_attempts     = 0,
    locked_until        = NULL,
    updated_at          = now()
WHERE email = 'admin@bilimbaga.local';

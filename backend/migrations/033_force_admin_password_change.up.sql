-- Migration 033: Force a password change for the seeded super_admin while it still has
-- the publicly documented default password (ISS-150, ISS-152).
-- Migration 030 reset the admin to the default hash with force_password_change = false,
-- so the default credential admin@bilimbaga.local / Admin1234! worked indefinitely.
-- Only rows whose password_hash is still exactly the 030 hash (or the 029 hash) are
-- touched: an admin who has already changed the password is left alone. Idempotent.
-- password_changed_at is deliberately NOT stamped (the password has not changed).
UPDATE users
SET
    force_password_change = true,
    updated_at            = now()
WHERE email = 'admin@bilimbaga.local'
  AND password_hash IN (
      '$2a$12$v7BCGG1mNmqtooAWqav4LuoB5i4n5hWA06iMyYLT.jPRMA1B9Mbo6',
      '$2b$12$.b2VsbiluRqUkJuT4gpfPOGpE4TLD3RTNZtXjHFUcqrIgFNVxyKj2'
  );

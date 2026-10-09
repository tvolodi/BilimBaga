-- ISS-105: access tokens issued before a password reset/admin reset must stop working.
-- NULL = password never changed since this column was added (no token is rejected).
ALTER TABLE users ADD COLUMN IF NOT EXISTS password_changed_at TIMESTAMPTZ;

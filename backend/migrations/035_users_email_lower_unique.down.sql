-- Reverts 035: drop the case-insensitive unique email index (no-op if it was never created).
DROP INDEX IF EXISTS idx_users_email_lower_unique;

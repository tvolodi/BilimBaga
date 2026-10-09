-- Migration 035: case-insensitive uniqueness for users.email (ISS-181, follow-up of ISS-164).
--
-- users.email carries a case-sensitive UNIQUE constraint, so "John@X.com" and "john@x.com"
-- can coexist in legacy data. The API self-migrates at startup and exits if a migration
-- fails, so a plain CREATE UNIQUE INDEX would take a deployed instance DOWN on such data.
-- This migration therefore NEVER fails because of duplicates: it creates the index only
-- when no case-insensitive duplicates exist; otherwise it raises a NOTICE and skips.
-- It never deletes, merges or modifies users. The API reports skipped-index duplicates at
-- startup (WARN log + audit_log 'users.duplicate_emails_detected').
DO $$
DECLARE
    dup_groups bigint;
BEGIN
    SELECT count(*) INTO dup_groups
    FROM (
        SELECT 1 FROM users GROUP BY lower(email) HAVING count(*) > 1
    ) d;

    IF dup_groups = 0 THEN
        BEGIN
            CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email_lower_unique ON users (lower(email));
        EXCEPTION WHEN unique_violation THEN
            -- A concurrent writer (old instance in a rolling deploy) inserted a twin between the
            -- check above and the build. Skip rather than fail startup.
            RAISE NOTICE 'idx_users_email_lower_unique NOT created: a case-insensitive duplicate appeared concurrently';
        END;
    ELSE
        RAISE NOTICE 'idx_users_email_lower_unique NOT created: % group(s) of users share an email ignoring case; resolve them manually, then run: CREATE UNIQUE INDEX idx_users_email_lower_unique ON users (lower(email))', dup_groups;
    END IF;
END
$$;

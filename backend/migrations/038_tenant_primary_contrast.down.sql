-- Reverts only rows that still hold the 038 defaults. A colour a tenant chose by hand
-- as exactly #2E6DB4 or #C8A84B is indistinguishable from the default and is reverted too.
UPDATE tenant_config
   SET value = '"#0ea5e9"'::jsonb, updated_at = now()
 WHERE key = 'primary_color' AND value = '"#2E6DB4"'::jsonb;

UPDATE tenant_config
   SET value = '"#f59e0b"'::jsonb, updated_at = now()
 WHERE key = 'accent_color' AND value = '"#C8A84B"'::jsonb;

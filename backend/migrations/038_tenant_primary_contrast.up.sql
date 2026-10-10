-- #443 (FR-BB320 AC-1): the seeded default primary (#0ea5e9, migration 002) gives 2.8:1 text
-- contrast on white. Move the untouched defaults to the design-system blue #2E6DB4 (5.3:1)
-- and the design-system gold accent #C8A84B. Only rows that still hold the 002 seed value
-- change; a colour a tenant chose is kept.
UPDATE tenant_config
   SET value = '"#2E6DB4"'::jsonb, updated_at = now()
 WHERE key = 'primary_color' AND value = '"#0ea5e9"'::jsonb;

UPDATE tenant_config
   SET value = '"#C8A84B"'::jsonb, updated_at = now()
 WHERE key = 'accent_color' AND value = '"#f59e0b"'::jsonb;

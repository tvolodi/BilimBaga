-- Migration: 002_tenant_config.up.sql

CREATE TABLE tenant_config (
    key        TEXT        PRIMARY KEY,
    value      JSONB       NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Seed default configuration
INSERT INTO tenant_config (key, value) VALUES
    ('app_name',          '"BilimBaga"'),
    ('logo',              'null'),
    ('primary_color',     '"#0ea5e9"'),
    ('accent_color',      '"#f59e0b"'),
    ('default_locale',    '"kk"'),
    ('available_locales', '["kk","ru","en"]')
ON CONFLICT (key) DO NOTHING;

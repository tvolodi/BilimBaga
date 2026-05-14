-- Migration 007 supersedes the audit_log created in 004_auth.
-- Drop the old single-tenant schema and replace with the multi-tenant version.
DROP TABLE IF EXISTS audit_log;

CREATE TABLE audit_log (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   TEXT        NOT NULL,
    actor_id    UUID,
    action      TEXT        NOT NULL,
    entity_type TEXT,
    entity_id   UUID,
    ip          TEXT,
    metadata    JSONB,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_audit_log_tenant_created ON audit_log(tenant_id, created_at DESC);
CREATE INDEX idx_audit_log_actor_id       ON audit_log(actor_id);
CREATE INDEX idx_audit_log_action         ON audit_log(action);
CREATE INDEX idx_audit_log_entity_type    ON audit_log(entity_type);

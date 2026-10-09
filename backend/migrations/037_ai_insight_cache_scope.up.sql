-- #263: durable scope-keyed insight cache. Existing rows were generated for the
-- unrestricted (super_admin) scope, so they become scope_key = 'all'. Department-
-- scoped callers get one row per (exam, scope hash) instead of the in-process map.
ALTER TABLE ai_insight_cache ADD COLUMN scope_key TEXT NOT NULL DEFAULT 'all';
ALTER TABLE ai_insight_cache ALTER COLUMN scope_key DROP DEFAULT;
ALTER TABLE ai_insight_cache DROP CONSTRAINT ai_insight_cache_pkey;
ALTER TABLE ai_insight_cache ADD CONSTRAINT ai_insight_cache_pkey PRIMARY KEY (exam_id, scope_key);

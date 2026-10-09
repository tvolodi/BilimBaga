-- Scoped rows cannot fit the old (exam_id) primary key: drop them first, keep 'all'.
DELETE FROM ai_insight_cache WHERE scope_key <> 'all';
ALTER TABLE ai_insight_cache DROP CONSTRAINT ai_insight_cache_pkey;
ALTER TABLE ai_insight_cache ADD CONSTRAINT ai_insight_cache_pkey PRIMARY KEY (exam_id);
ALTER TABLE ai_insight_cache DROP COLUMN scope_key;

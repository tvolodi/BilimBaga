-- Migration 010: FR-BB23 — Question CRUD API
-- Adds a covering index on question_tags(tag_id) to support efficient tag-filter queries.
CREATE INDEX IF NOT EXISTS idx_question_tags_tag_id ON question_tags(tag_id);

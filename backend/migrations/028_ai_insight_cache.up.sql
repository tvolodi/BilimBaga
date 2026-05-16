CREATE TABLE ai_insight_cache (
  exam_id       UUID        PRIMARY KEY REFERENCES exams(id) ON DELETE CASCADE,
  insights      JSONB       NOT NULL,
  generated_at  TIMESTAMPTZ NOT NULL,
  generated_by  UUID        REFERENCES users(id) ON DELETE SET NULL
);

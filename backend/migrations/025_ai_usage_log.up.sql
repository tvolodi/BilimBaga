CREATE TABLE ai_usage_log (
  id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id     UUID        REFERENCES users(id) ON DELETE CASCADE,
  feature     TEXT        NOT NULL,
  tokens_used INT         NOT NULL,
  model       TEXT        NOT NULL,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_ai_usage_log_user_created
  ON ai_usage_log (user_id, created_at DESC);

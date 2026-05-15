-- FR-BB43: Certificate Generation

CREATE TABLE IF NOT EXISTS certificates (
  id                UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
  session_id        UUID         NOT NULL UNIQUE REFERENCES exam_sessions(id),
  verification_code UUID         NOT NULL UNIQUE DEFAULT gen_random_uuid(),
  issued_at         TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
  employee_name     TEXT         NOT NULL,
  exam_title        TEXT         NOT NULL,
  score_pct         DECIMAL(5,2) NOT NULL,
  template_snapshot JSONB        NOT NULL
);

-- idx_certificates_verification_code is intentionally omitted:
-- the UNIQUE constraint already creates an implicit B-tree index.
CREATE INDEX IF NOT EXISTS idx_certificates_session_id ON certificates(session_id);

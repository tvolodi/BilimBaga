-- FR-BB51: Dashboard Metrics API — performance indexes
-- These indexes support the four dashboard queries without modifying existing schema.

-- Overdue employees query: filter by deadline and look up sessions by exam+user.
CREATE INDEX IF NOT EXISTS idx_exam_assignments_deadline
    ON exam_assignments(deadline)
    WHERE deadline IS NOT NULL;

-- Recent activity query: sessions ordered by submitted_at for submitted/grading_pending.
CREATE INDEX IF NOT EXISTS idx_exam_sessions_submitted_at
    ON exam_sessions(submitted_at DESC)
    WHERE status IN ('submitted', 'grading_pending');

-- Track score query: session_question_scores → questions lookup.
-- idx_questions_category_id already exists from migration 009; guard with IF NOT EXISTS.
CREATE INDEX IF NOT EXISTS idx_questions_category_id ON questions(category_id);

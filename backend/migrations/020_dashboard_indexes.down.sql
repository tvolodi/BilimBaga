-- FR-BB51: Dashboard Metrics API — rollback performance indexes
DROP INDEX IF EXISTS idx_exam_assignments_deadline;
DROP INDEX IF EXISTS idx_exam_sessions_submitted_at;
-- idx_questions_category_id is owned by migration 009; do not drop it here.

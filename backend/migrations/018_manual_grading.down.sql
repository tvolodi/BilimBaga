-- FR-BB42: Manual Grading Queue — rollback

DELETE FROM role_permissions
WHERE permission_id IN (
  SELECT id FROM permissions WHERE resource = 'grading'
);

DELETE FROM permissions WHERE resource = 'grading';

ALTER TABLE session_question_scores
  DROP COLUMN IF EXISTS manual_feedback,
  DROP COLUMN IF EXISTS graded_by,
  DROP COLUMN IF EXISTS graded_at;

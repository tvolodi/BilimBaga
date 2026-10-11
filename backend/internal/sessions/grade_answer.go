package sessions

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// Errors of the examiner's grade (FR-BB73 AC-19 to AC-21). The handler maps each to its status and code.
var (
	// ErrSessionFinalised: the session is final and the answer is already graded by a person (AC-19, 409).
	ErrSessionFinalised = errors.New("session is final and this answer is already graded")
	// ErrCertificateIssued: a certificate exists, so a final session may not be overridden (AC-20, 409).
	ErrCertificateIssued = errors.New("a certificate was issued for this session")
	// ErrCannotGradeOwnSession: the acting user is the employee of the session, for every role (AC-21, 403).
	ErrCannotGradeOwnSession = errors.New("the employee cannot grade their own session")
)

// gradeLockedSession is the session as read under its row lock, before any grading rule is applied.
type gradeLockedSession struct {
	UserID string `db:"user_id"`
	Status string `db:"status"`
	Passed bool   `db:"passed"`
}

// gradeRow is the answer being graded, read under the session lock.
type gradeRow struct {
	GradingStatus string          `db:"grading_status"`
	MaxScore      float64         `db:"max_score"`
	Score         float64         `db:"score"`
	Suggested     sql.NullFloat64 `db:"ai_suggested_score_pct"`
	Flags         pq.StringArray  `db:"ai_flags"`
}

// GradeAnswer scores one short-text answer inside one transaction (FR-BB42 AC-7/AC-10; FR-BB73 AC-17 to AC-21). The
// order is fixed: the session lock first; then the owner and the status are read under it; then the self-grading rule;
// then the row rules. Nothing is written when a rule refuses.
func (r *postgresRepository) GradeAnswer(ctx context.Context, sessionID, questionID, graderID, tenantID, actorIP string, scorePct float64, feedback string) (*gradeAnswerResult, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("sessions: GradeAnswer: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	sess, err := lockSessionForGrade(ctx, tx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("sessions: GradeAnswer: %w", err)
	}
	// AC-21: checked here, in the transaction, so no handler can skip it.
	if sess.UserID == graderID {
		return nil, ErrCannotGradeOwnSession
	}

	row, err := readGradeRow(ctx, tx, sessionID, questionID)
	if err != nil {
		return nil, fmt.Errorf("sessions: GradeAnswer: %w", err)
	}

	final := sess.Status == "submitted" || sess.Status == "auto_submitted"
	override := row.GradingStatus == "ai_graded"
	if final && row.GradingStatus == "graded" {
		// AC-19: a final session's graded answer is not re-graded silently.
		return nil, ErrSessionFinalised
	}
	if final && override {
		// AC-20: an override of a final session with an issued certificate changes nothing. The certificate is read
		// under the session lock, which the issuance also takes first (FR-BB73 AC-20).
		issued, err := certificateIssued(ctx, tx, sessionID)
		if err != nil {
			return nil, fmt.Errorf("sessions: GradeAnswer: %w", err)
		}
		if issued {
			return nil, ErrCertificateIssued
		}
	}

	// The posted score is compared and stored at two decimals (FR-BB73 AC-17), the precision of the suggestion column.
	posted := roundTo2(scorePct)
	score := posted / 100.0 * row.MaxScore
	const updateQ = `
UPDATE session_question_scores
SET score           = $3,
    grading_status  = 'graded',
    manual_feedback = $4,
    graded_by       = $5,
    graded_at       = NOW()
WHERE session_id = $1 AND question_id = $2`
	if _, err := tx.ExecContext(ctx, updateQ, sessionID, questionID, score, feedback, graderID); err != nil {
		return nil, fmt.Errorf("sessions: GradeAnswer: update score: %w", err)
	}

	fin, err := finaliseSessionIfAllGraded(ctx, tx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("sessions: GradeAnswer: %w", err)
	}

	if err := writeGradeAudit(ctx, tx, tenantID, graderID, questionID, sessionID, actorIP, posted, override, row); err != nil {
		return nil, fmt.Errorf("sessions: GradeAnswer: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("sessions: GradeAnswer: commit: %w", err)
	}
	return &gradeAnswerResult{
		allGraded:      fin.allGraded,
		finalScorePct:  fin.finalScorePct,
		passed:         fin.passed,
		sessionStatus:  fin.sessionStatus,
		previousStatus: sess.Status,
		passedFlipped:  final && fin.passed != nil && *fin.passed != sess.Passed,
	}, nil
}

// lockSessionForGrade takes the session row lock and reads the owner, the status and the stored pass under it.
// SubmitSession locks the same row first, so the lock order is the same everywhere (FR-BB73 AC-19 step 1).
func lockSessionForGrade(ctx context.Context, tx *sqlx.Tx, sessionID string) (gradeLockedSession, error) {
	const q = `
SELECT user_id, status, COALESCE(passed, false) AS passed
FROM exam_sessions
WHERE id = $1
FOR UPDATE`
	var sess gradeLockedSession
	if err := tx.GetContext(ctx, &sess, q, sessionID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return gradeLockedSession{}, ErrSessionNotFound
		}
		return gradeLockedSession{}, fmt.Errorf("lock session: %w", err)
	}
	return sess, nil
}

// readGradeRow reads the answer being graded. The session lock is already held.
func readGradeRow(ctx context.Context, tx *sqlx.Tx, sessionID, questionID string) (gradeRow, error) {
	const q = `
SELECT grading_status, max_score, score, ai_suggested_score_pct, ai_flags
FROM session_question_scores
WHERE session_id = $1 AND question_id = $2`
	var row gradeRow
	if err := tx.GetContext(ctx, &row, q, sessionID, questionID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return gradeRow{}, ErrSessionNotFound
		}
		return gradeRow{}, fmt.Errorf("fetch answer: %w", err)
	}
	return row, nil
}

// certificateIssued reports whether a certificate exists for the session. The session lock is already held.
func certificateIssued(ctx context.Context, tx *sqlx.Tx, sessionID string) (bool, error) {
	const q = `SELECT EXISTS (SELECT 1 FROM certificates WHERE session_id = $1)`
	var issued bool
	if err := tx.GetContext(ctx, &issued, q, sessionID); err != nil {
		return false, fmt.Errorf("certificate check: %w", err)
	}
	return issued, nil
}

// writeGradeAudit writes the audit row of a grade (FR-BB73 AC-17) or of an override of an ai_graded answer (AC-19).
// The server writes the suggested score, whether the posted score accepted it (at two decimals), and the flag codes.
func writeGradeAudit(ctx context.Context, tx *sqlx.Tx, tenantID, graderID, questionID, sessionID, actorIP string, posted float64, override bool, row gradeRow) error {
	var suggested *float64
	if row.Suggested.Valid {
		v := roundTo2(row.Suggested.Float64)
		suggested = &v
	}
	if override {
		previous := 0.0
		if row.MaxScore > 0 {
			previous = roundTo2(row.Score / row.MaxScore * 100)
		}
		meta, err := json.Marshal(map[string]any{
			"session_id":         sessionID,
			"grader_id":          graderID,
			"previous_score_pct": previous,
			"new_score_pct":      posted,
		})
		if err != nil {
			return fmt.Errorf("override metadata: %w", err)
		}
		const q = `
INSERT INTO audit_log (tenant_id, actor_id, action, entity_type, entity_id, ip, metadata)
VALUES ($1, $2, 'answer.ai_override', 'session_answer', $3, $4, $5::jsonb)`
		_, err = tx.ExecContext(ctx, q, tenantID, graderID, questionID, actorIP, string(meta))
		return err
	}

	flags := []string(row.Flags)
	if flags == nil {
		flags = []string{}
	}
	accepted := suggested != nil && *suggested == posted
	meta, err := json.Marshal(map[string]any{
		"grader_id":           graderID,
		"session_id":          sessionID,
		"score_pct":           posted,
		"suggested_score_pct": suggested,
		"accepted":            accepted,
		"flags":               flags,
	})
	if err != nil {
		return fmt.Errorf("grade metadata: %w", err)
	}
	const q = `
INSERT INTO audit_log (tenant_id, actor_id, action, entity_type, entity_id, ip, metadata)
VALUES ($1, $2, 'answer.grade', 'session_answer', $3, $4, $5::jsonb)`
	_, err = tx.ExecContext(ctx, q, tenantID, graderID, questionID, actorIP, string(meta))
	return err
}

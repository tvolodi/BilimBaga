package sessions

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/jmoiron/sqlx"
)

// expiredSession is the minimal data fetched per expired session row.
type expiredSession struct {
	ID        string    `db:"id"`
	ExamID    string    `db:"exam_id"`
	UserID    string    `db:"user_id"`
	ExpiresAt time.Time `db:"expires_at"`
}

// jobDeps holds injectable dependencies for the auto-submit job, allowing unit tests
// to replace DB-touching functions without a live database.
type jobDeps struct {
	queryExpired func(ctx context.Context, db *sqlx.DB) ([]expiredSession, error)
	processOne   func(ctx context.Context, db *sqlx.DB, logger *slog.Logger, sess expiredSession) error
}

func defaultJobDeps(engine GradingEngine) jobDeps {
	return jobDeps{
		queryExpired: queryExpiredSessions,
		processOne: func(ctx context.Context, db *sqlx.DB, logger *slog.Logger, sess expiredSession) error {
			return processOneExpiredSession(ctx, db, logger, sess, engine)
		},
	}
}

// AutoSubmitJob polls every 60 seconds for in_progress sessions whose
// expires_at has passed, auto-submits them, grades them, and writes audit log
// entries. It shuts down when ctx is cancelled (AC-6).
//
// Call as: go AutoSubmitJob(ctx, db, logger, engine)
func AutoSubmitJob(ctx context.Context, db *sqlx.DB, logger *slog.Logger, engine GradingEngine) {
	autoSubmitJobWithDeps(ctx, db, logger, defaultJobDeps(engine))
}

func autoSubmitJobWithDeps(ctx context.Context, db *sqlx.DB, logger *slog.Logger, deps jobDeps) {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	logger.Info("auto-submit job started")

	for {
		select {
		case <-ctx.Done():
			logger.Info("auto-submit job stopping")
			return
		case <-ticker.C:
			runTick(ctx, db, logger, deps)
		}
	}
}

// runTick executes one polling cycle. Exposed for testing via deps injection.
func runTick(ctx context.Context, db *sqlx.DB, logger *slog.Logger, deps jobDeps) {
	defer func() {
		// AC-7: recover panics so the goroutine survives.
		if r := recover(); r != nil {
			logger.Error("auto-submit job panic recovered", "panic", r)
		}
	}()

	sessions, err := deps.queryExpired(ctx, db)
	if err != nil {
		logger.Error("auto-submit: query failed", "err", err)
		return
	}

	// AC-10: INFO level with count of sessions processed.
	logger.Info("auto-submit: tick", "expired_count", len(sessions))

	for _, sess := range sessions {
		// AC-10: DEBUG level with session IDs.
		logger.Debug("auto-submit: processing session", "session_id", sess.ID)

		if err := deps.processOne(ctx, db, logger, sess); err != nil {
			// AC-8: failure on one session does not stop processing others.
			logger.Error("auto-submit: session failed", "session_id", sess.ID, "err", err)
		} else {
			logger.Debug("auto-submit: session processed", "session_id", sess.ID)
		}
	}
}

// queryExpiredSessions fetches all in_progress sessions whose expires_at < NOW().
// FOR UPDATE SKIP LOCKED prevents double-processing in multi-instance deployments (AC-9).
func queryExpiredSessions(ctx context.Context, db *sqlx.DB) ([]expiredSession, error) {
	const q = `
SELECT id, exam_id, user_id, expires_at
FROM exam_sessions
WHERE status = 'in_progress'
  AND expires_at < NOW()
FOR UPDATE SKIP LOCKED`

	// FOR UPDATE requires a transaction.
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("auto-submit: queryExpiredSessions: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	rows, err := tx.QueryxContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("auto-submit: queryExpiredSessions: query: %w", err)
	}

	var result []expiredSession
	for rows.Next() {
		var s expiredSession
		if err := rows.StructScan(&s); err != nil {
			rows.Close()
			return nil, fmt.Errorf("auto-submit: queryExpiredSessions: scan: %w", err)
		}
		result = append(result, s)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("auto-submit: queryExpiredSessions: rows: %w", err)
	}

	// Commit releases the FOR UPDATE locks.
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("auto-submit: queryExpiredSessions: commit: %w", err)
	}
	return result, nil
}

// processOneExpiredSession runs the full auto-submit lifecycle for a single
// session inside its own transaction (AC-8: one tx per session).
func processOneExpiredSession(ctx context.Context, db *sqlx.DB, logger *slog.Logger, sess expiredSession, engine GradingEngine) error {
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	// AC-4: check for shorttext questions to determine target status.
	const hasShortTextQ = `
SELECT EXISTS (
    SELECT 1 FROM session_questions sq
    JOIN questions q ON q.id = sq.question_id
    WHERE sq.session_id = $1 AND q.type = 'shorttext'
)`
	var hasShortText bool
	if err := tx.QueryRowContext(ctx, hasShortTextQ, sess.ID).Scan(&hasShortText); err != nil {
		return fmt.Errorf("check short_text: %w", err)
	}

	newStatus := "auto_submitted"
	if hasShortText {
		newStatus = "grading_pending"
	}

	// AC-3: submitted_at = expires_at (not NOW()).
	const updateQ = `
UPDATE exam_sessions
SET status = $2, submitted_at = $3
WHERE id = $1 AND status = 'in_progress'`
	res, err := tx.ExecContext(ctx, updateQ, sess.ID, newStatus, sess.ExpiresAt)
	if err != nil {
		return fmt.Errorf("update status: %w", err)
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if rowsAffected == 0 {
		// Session was already processed by another instance (harmless race).
		return nil
	}

	// Grade the session using the engine (handles all question types).
	if _, _, err := engine.Grade(tx, sess.ID); err != nil {
		return fmt.Errorf("grading: %w", err)
	}

	// AC-5: write audit log entry.
	meta, _ := json.Marshal(map[string]any{
		"exam_id": sess.ExamID,
		"user_id": sess.UserID,
	})
	const auditQ = `
INSERT INTO audit_log (tenant_id, actor_id, action, entity_type, entity_id, ip, metadata)
SELECT e.tenant_id, NULL, 'session.auto_submit', 'exam_session', $1, '', $2::jsonb
FROM exams e WHERE e.id = $3`
	if _, err := tx.ExecContext(ctx, auditQ, sess.ID, string(meta), sess.ExamID); err != nil {
		// Log but don't abort — the session state transition already happened.
		logger.Error("auto-submit: audit log write failed", "session_id", sess.ID, "err", err)
	}

	return tx.Commit()
}

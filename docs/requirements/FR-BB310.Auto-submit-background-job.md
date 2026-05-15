# FR-BB310 — Auto-Submit Background Job

## Metadata
| Field | Value |
|-------|-------|
| ID | FR-BB310 |
| Phase | 3 — Exam Engine |
| Priority | 2 |
| Status | implemented |
| Depends On | FR-BB39 |

## Description
Implements a background goroutine that runs every 60 seconds during API server lifetime. It queries for `in_progress` sessions whose `expires_at` has passed, marks each as `auto_submitted`, triggers the grading engine, and writes audit log entries. The goroutine shuts down cleanly when the server receives a termination signal.

## Acceptance Criteria
- [ ] AC-1: The ticker goroutine starts on API server startup within the `main()` or application-wiring layer, not within an HTTP handler.
- [ ] AC-2: Each tick queries `exam_sessions WHERE status = 'in_progress' AND expires_at < NOW()` and processes all matching rows.
- [ ] AC-3: For each expired session, `status` is updated to `'auto_submitted'` and `submitted_at` is set to `expires_at` (not `NOW()`), preserving the true end-of-exam time.
- [ ] AC-4: The grading engine (FR-BB311) is invoked for each auto-submitted session; sessions with short-text questions transition to `grading_pending` rather than retaining `auto_submitted` for pending-grade signalling (set `status = 'grading_pending'`).
- [ ] AC-5: An audit log entry is written per processed session: `{ action: "session.auto_submit", entity_type: "exam_session", entity_id: session_id, metadata: { exam_id, user_id } }`.
- [ ] AC-6: The goroutine shuts down cleanly when the application context is cancelled (e.g. SIGINT/SIGTERM); it does not leak goroutines.
- [ ] AC-7: Panics inside the ticker loop are recovered and logged as errors; the goroutine restarts its wait cycle after recovery without crashing the server.
- [ ] AC-8: Processing each expired session is wrapped in its own database transaction; a failure on one session does not prevent processing of other sessions in the same tick.
- [ ] AC-9: The job does not re-process sessions already in `submitted`, `auto_submitted`, or `grading_pending` status (ensured by the `WHERE status = 'in_progress'` filter).
- [ ] AC-10: Log output at `INFO` level is emitted at each tick showing the count of sessions processed; `DEBUG` level logs include session IDs.

## Technical Specification

### Background Job

```go
// AutoSubmitJob polls for expired sessions and processes them.
// Call as: go AutoSubmitJob(ctx, db, gradingEngine, auditLogger)
func AutoSubmitJob(ctx context.Context, db *sqlx.DB, grader GradingEngine, audit AuditLogger) {
    ticker := time.NewTicker(60 * time.Second)
    defer ticker.Stop()

    log.Info("auto-submit job started")

    for {
        select {
        case <-ctx.Done():
            log.Info("auto-submit job stopping")
            return
        case <-ticker.C:
            processExpiredSessions(ctx, db, grader, audit)
        }
    }
}

func processExpiredSessions(ctx context.Context, db *sqlx.DB, grader GradingEngine, audit AuditLogger) {
    defer func() {
        if r := recover(); r != nil {
            log.Errorf("auto-submit job panic recovered: %v", r)
        }
    }()

    sessions, err := queryExpiredSessions(ctx, db)
    if err != nil {
        log.Errorf("auto-submit: query failed: %v", err)
        return
    }

    log.Infof("auto-submit: processing %d expired sessions", len(sessions))

    for _, sess := range sessions {
        if err := processOne(ctx, db, grader, audit, sess); err != nil {
            log.Errorf("auto-submit: failed session %s: %v", sess.ID, err)
            // continue to next session
        }
    }
}

func processOne(ctx context.Context, db *sqlx.DB, grader GradingEngine, audit AuditLogger, sess Session) error {
    tx, err := db.BeginTxx(ctx, nil)
    if err != nil { return fmt.Errorf("begin tx: %w", err) }
    defer tx.Rollback()

    hasShortText := sessionHasShortTextQuestions(tx, sess.ID)
    newStatus := "auto_submitted"
    if hasShortText { newStatus = "grading_pending" }

    if err := updateSessionStatusAndSubmittedAt(tx, sess.ID, newStatus, sess.ExpiresAt); err != nil {
        return fmt.Errorf("update status: %w", err)
    }

    if newStatus == "auto_submitted" {
        score, passed, err := grader.Grade(tx, sess.ID)
        if err != nil { return fmt.Errorf("grading: %w", err) }
        if err := updateSessionScore(tx, sess.ID, score, passed); err != nil {
            return fmt.Errorf("update score: %w", err)
        }
    }

    audit.Log(tx, AuditEntry{
        Action:     "session.auto_submit",
        EntityType: "exam_session",
        EntityID:   sess.ID,
        Metadata: map[string]any{
            "exam_id": sess.ExamID,
            "user_id": sess.UserID,
        },
    })

    return tx.Commit()
}
```

### Query for Expired Sessions

```sql
SELECT id, exam_id, user_id, expires_at
FROM exam_sessions
WHERE status = 'in_progress'
  AND expires_at < NOW()
FOR UPDATE SKIP LOCKED;
```

`FOR UPDATE SKIP LOCKED` ensures that if multiple API instances run concurrently (horizontal scaling), only one instance processes each session.

## Notes
- The `FOR UPDATE SKIP LOCKED` pattern is essential in multi-instance deployments to avoid double-processing of sessions.
- The ticker interval of 60 seconds means sessions may be auto-submitted up to 60 seconds after their actual expiry. This is acceptable; the `submitted_at` is set to `expires_at` to preserve accuracy.
- If the server restarts while sessions are expired, the job will catch and process them on the first tick after startup.
- The job should be wired into the application context derived from `signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)` so it respects graceful shutdown.
- Metrics (Prometheus or equivalent) can optionally count auto-submitted sessions per tick; instrumentation is out of scope for this requirement.

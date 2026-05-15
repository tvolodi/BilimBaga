# FR-BB39 — Session Submission

## Metadata
| Field | Value |
|-------|-------|
| ID | FR-BB39 |
| Phase | 3 — Exam Engine |
| Priority | 1 |
| Status | implemented |
| Depends On | FR-BB37 |

## Description
Implements the explicit session-submission endpoint (`POST /api/v1/portal/sessions/:id/submit`). When the user deliberately ends their exam, the endpoint validates ownership and session state, determines whether all questions can be auto-graded or require manual review, transitions the session status accordingly, triggers the grading engine (FR-BB311) synchronously for fully auto-gradable sessions, and returns the result or a pending notification.

## Acceptance Criteria
- [ ] AC-1: Returns HTTP 403 if the session does not belong to the calling user.
- [ ] AC-2: Returns HTTP 422 with code `SESSION_NOT_ACTIVE` if the session `status` is not `'in_progress'`.
- [ ] AC-3: Submission is accepted even if `expires_at` has already passed (the client may submit slightly after expiry due to network delay); the background job (FR-BB310) will not re-process an already-submitted session.
- [ ] AC-4: If the session contains any questions of type `short_text`, status transitions to `grading_pending` and an audit log entry is written with `action = 'session.pending_manual_grade'`.
- [ ] AC-5: If no `short_text` questions exist in the session, status transitions to `submitted` and the grading engine runs synchronously within the same request; `score_pct` and `passed` are populated in the response.
- [ ] AC-6: `submitted_at` is set to `NOW()` at time of submission; `expires_at` is not modified.
- [ ] AC-7: The response always includes `session_id` and `status`; it includes `score_pct` and `passed` only when `status = 'submitted'` (not when `grading_pending`).
- [ ] AC-8: The endpoint is idempotent: if called twice on an already-submitted session, the second call returns the current state with HTTP 200 rather than an error.
- [ ] AC-9: An audit log entry `{ action: 'session.submit', entity_type: 'exam_session', entity_id: session_id }` is written for every successful explicit submission.
- [ ] AC-10: The entire submission (status update + grading) executes within a single database transaction to ensure score is never written without the status transition.

## Technical Specification

### API Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/api/v1/portal/sessions/:id/submit` | any authenticated | Explicitly submit the exam session |

#### Request / Response Shapes

**POST /api/v1/portal/sessions/:id/submit — Request**
```json
{}
```
*(No request body required.)*

**Response — Auto-graded (200)**
```json
{
  "data": {
    "session_id": "sess-uuid-42",
    "status": "submitted",
    "submitted_at": "2026-05-14T11:15:00Z",
    "score_pct": 82.50,
    "passed": true
  },
  "error": null
}
```

**Response — Pending manual grading (200)**
```json
{
  "data": {
    "session_id": "sess-uuid-43",
    "status": "grading_pending",
    "submitted_at": "2026-05-14T11:15:00Z",
    "score_pct": null,
    "passed": null
  },
  "error": null
}
```

**Response — Already submitted (idempotent) (200)**
```json
{
  "data": {
    "session_id": "sess-uuid-42",
    "status": "submitted",
    "submitted_at": "2026-05-14T11:15:00Z",
    "score_pct": 82.50,
    "passed": true
  },
  "error": null
}
```

**Error — Not Session Owner (403)**
```json
{
  "data": null,
  "error": {
    "code": "FORBIDDEN",
    "message": "You do not have access to this session."
  }
}
```

### Handler Logic (Go pseudocode)

```go
func handleSubmitSession(w http.ResponseWriter, r *http.Request) {
    sessionID := chi.URLParam(r, "id")
    userID := r.Context().Value(ctxUserID).(uuid.UUID)

    tx := db.BeginTx(r.Context())
    defer tx.Rollback()

    session := fetchSessionForUserTx(tx, sessionID, userID) // 403 if not owner
    if session == nil { respond(w, 403, errForbidden); return }

    // Idempotency: already submitted
    if session.Status != "in_progress" {
        respond(w, 200, buildResult(session))
        return
    }

    hasShortText := sessionHasShortTextQuestions(tx, sessionID)

    newStatus := "submitted"
    if hasShortText { newStatus = "grading_pending" }

    updateSessionStatus(tx, sessionID, newStatus, time.Now())
    writeAuditLog(tx, "session.submit", "exam_session", sessionID)

    var score float64
    var passed *bool
    if newStatus == "submitted" {
        score, passed = runGradingEngine(tx, sessionID) // FR-BB311
        updateSessionScore(tx, sessionID, score, *passed)
    } else {
        writeAuditLog(tx, "session.pending_manual_grade", "exam_session", sessionID)
    }

    tx.Commit()
    respond(w, 200, SubmitResult{
        SessionID:   sessionID,
        Status:      newStatus,
        SubmittedAt: time.Now(),
        ScorePct:    score,
        Passed:      passed,
    })
}
```

## Notes
- The `hasShortText` check queries `session_questions JOIN questions WHERE type = 'short_text'`; this is a small indexed lookup and acceptable in the submission path.
- Manual grading workflow notification (e.g. sending a notification to HR admins) is out of scope for this requirement; the audit log entry serves as the trigger signal until a notification system is built.
- For partial-credit multiple-choice exams with no short-text questions, grading runs synchronously and completes within the request timeout; if grading is expected to be slow (e.g. large Likert surveys), a queue-based approach should be adopted in a future phase.
- Certificate generation (if `certificate_enabled = TRUE` and `passed = TRUE`) is triggered post-transaction as a fire-and-forget goroutine; it is not part of this requirement's scope.

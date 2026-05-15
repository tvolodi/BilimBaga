# FR-BB38 — Tab-Switch Events

## Metadata
| Field | Value |
|-------|-------|
| ID | FR-BB38 |
| Phase | 3 — Exam Engine |
| Priority | 2 |
| Status | Implemented |
| Depends On | FR-BB37 |

## Description
Implements the anti-cheat event-reporting endpoint that the exam-taking frontend calls when it detects a tab switch, window blur, or fullscreen exit. The server records the event and enforces the exam-level `on_tab_switch` policy: logging only, warning the user, or immediately auto-submitting the session.

## Acceptance Criteria
- [ ] AC-1: `POST /api/v1/portal/sessions/:id/events` inserts a row into `tab_switch_events` with `occurred_at = NOW()` and the resolved `action_taken` value for every valid request.
- [ ] AC-2: When `exam.on_tab_switch = 'submit'`, the endpoint triggers the same auto-submit flow as FR-BB39, setting session `status = 'auto_submitted'` and `submitted_at = NOW()`, then triggers grading; the response reflects the submitted state.
- [ ] AC-3: When `exam.on_tab_switch = 'warn'`, the response includes `{ warn: true, event_count: N }` where `N` is the total count of tab-switch events for this session including the current one.
- [ ] AC-4: When `exam.on_tab_switch = 'log'`, the response is HTTP 200 with `{ warn: false }`.
- [ ] AC-5: Returns HTTP 422 with code `SESSION_NOT_ACTIVE` if the session is not `in_progress`.
- [ ] AC-6: Returns HTTP 422 with code `SESSION_EXPIRED` if `expires_at <= NOW()`.
- [ ] AC-7: Returns HTTP 403 if the session does not belong to the calling user.
- [ ] AC-8: The `type` field in the request body accepts only `'tab_switch'`, `'blur'`, or `'fullscreen_exit'`; any other value returns HTTP 400.
- [ ] AC-9: The endpoint is idempotent from an audit perspective — each call always records a new event row; it does not deduplicate rapid successive events.
- [ ] AC-10: `action_taken` stored in `tab_switch_events` matches the policy in effect at event time (derived from the exam config, not from the request body).

## Technical Specification

### API Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/api/v1/portal/sessions/:id/events` | any authenticated | Report a tab-switch or focus-loss event |

#### Request / Response Shapes

**POST /api/v1/portal/sessions/:id/events — Request**
```json
{
  "type": "tab_switch"
}
```

**Response — on_tab_switch = 'log' (200)**
```json
{
  "data": {
    "warn": false,
    "event_count": 3
  },
  "error": null
}
```

**Response — on_tab_switch = 'warn' (200)**
```json
{
  "data": {
    "warn": true,
    "event_count": 3
  },
  "error": null
}
```

**Response — on_tab_switch = 'submit' (200)**
```json
{
  "data": {
    "warn": false,
    "session_id": "sess-uuid-42",
    "status": "auto_submitted",
    "score_pct": null,
    "passed": null
  },
  "error": null
}
```

**Error — Session Expired (422)**
```json
{
  "data": null,
  "error": {
    "code": "SESSION_EXPIRED",
    "message": "Your exam session has expired."
  }
}
```

**Error — Invalid Event Type (400)**
```json
{
  "data": null,
  "error": {
    "code": "INVALID_EVENT_TYPE",
    "message": "Event type must be one of: tab_switch, blur, fullscreen_exit."
  }
}
```

### Handler Logic (Go pseudocode)

```go
func handleSessionEvent(w http.ResponseWriter, r *http.Request) {
    sessionID := chi.URLParam(r, "id")
    userID := r.Context().Value(ctxUserID).(uuid.UUID)

    var body struct { Type string `json:"type"` }
    decode(r, &body)

    if !validEventTypes[body.Type] {
        respond(w, 400, errInvalidEventType)
        return
    }

    session, exam := fetchSessionAndExam(sessionID, userID) // 403/422 on failure
    validateSessionActive(session) // 422 SESSION_NOT_ACTIVE or SESSION_EXPIRED

    actionTaken := exam.OnTabSwitch.String()
    insertTabSwitchEvent(sessionID, body.Type, actionTaken)

    eventCount := countTabSwitchEvents(sessionID)

    switch exam.OnTabSwitch {
    case "submit":
        result := autoSubmitSession(session, exam)
        respond(w, 200, result)
    case "warn":
        respond(w, 200, map[string]any{"warn": true, "event_count": eventCount})
    default: // "log"
        respond(w, 200, map[string]any{"warn": false, "event_count": eventCount})
    }
}
```

## Notes
- The auto-submit triggered by `on_tab_switch = 'submit'` should call the same internal `submitSession()` function used by FR-BB39 to avoid logic duplication.
- Rapid successive tab-switch events (e.g. user switches in and out quickly) will each trigger a separate event row; the frontend should implement a debounce of at least 500 ms before calling this endpoint.
- The `event_count` in the warn response gives the frontend enough context to show escalating warning messages (e.g. "Warning 1 of 3 — further violations will auto-submit").
- If the exam `on_tab_switch = 'submit'` but grading encounters short-text questions, the session status transitions to `grading_pending` (same as manual submission with short-text answers) rather than directly to `submitted`.

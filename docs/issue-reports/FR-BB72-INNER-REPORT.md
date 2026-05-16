# FR-BB72 Inner Report — Adaptive Difficulty

## Summary

Completed implementation of FR-BB72 (Adaptive Difficulty exam mode), picking up from a partially started state.

## Work Done

### Backend

**Pre-existing (from previous agent pass):**
- `backend/internal/exams/adaptive.go`: `NextDifficulty`, `difficultyIndex` algorithm
- `backend/internal/sessions/model.go`: `AdaptiveState`, `NextQuestionResponse`, `ErrNotAdaptive`, `ErrNoQuestionsAvailable`, `Adaptive` field on `examConfig` and `ResumeSessionResponse`
- `backend/internal/sessions/service.go`: `SelectNextAdaptiveQuestion`, `RecordAdaptiveAnswer` methods
- `backend/internal/exams/service.go`: Publish validation (`INSUFFICIENT_ADAPTIVE_QUESTIONS`), `AdaptivePublishValidationError`
- `backend/internal/exams/handler.go`: `INSUFFICIENT_ADAPTIVE_QUESTIONS` 422 response
- `backend/internal/exams/repository.go`: `CountQuestionsPerDifficulty` interface + implementation
- `backend/migrations/026_adaptive_exam.up.sql`: `adaptive BOOL` on exams, `adaptive_state JSONB` on exam_sessions

**Completed in this pass:**
- `backend/internal/sessions/repository.go`: Added `Adaptive bool` field to `sessionStateRow`; updated `GetSessionForUser` query to fetch `e.adaptive`
- `backend/internal/sessions/service.go`: Populated `Adaptive` field in `ResumeSessionResponse`
- `backend/internal/sessions/handler.go`: Added `GetNextQuestion` handler (returns 404/NOT_ADAPTIVE, 403/SESSION_FORBIDDEN, 404/SESSION_NOT_FOUND, 200 with question or `done: true`)
- `backend/internal/router/router.go`: Registered `GET /portal/sessions/{id}/next-question` route inside JWT middleware group

**Test fixes:**
- `backend/internal/exams/service_test.go`: Added `CountQuestionsPerDifficulty` fn field and method to `mockRepo`
- `backend/internal/sessions/service_test.go`: Added all FR-BB72 adaptive fn fields + implementations to `mockRepo`
- `backend/internal/sessions/handler_test.go`: Added `SelectNextAdaptiveQuestion` and `RecordAdaptiveAnswer` to `mockSvc`

**New tests written:**
- `backend/internal/exams/adaptive_test.go`: 9 tests covering all `NextDifficulty` branches (empty results, increase, decrease, floor, ceiling, last-3-only window, mixed results, unknown difficulty)
- `backend/internal/sessions/service_test.go`: 5 new adaptive tests (`NotAdaptive`, `ReturnsCurrentQuestion` idempotency, `SelectsNewQuestion`, `DoneWhenNoQuestionsAvailable`, `RecordAdaptiveAnswer`)
- `backend/internal/sessions/handler_test.go`: 6 new handler tests (`200 question`, `200 done`, `404 NOT_ADAPTIVE`, `404 SESSION_NOT_FOUND`, `403 SESSION_FORBIDDEN`, `500 internal error`)

### Frontend

- `frontend/src/api/sessions.ts`: Added `adaptive: boolean` to `ResumeSessionResponse`, added `NextQuestionResponse` interface, added `useNextAdaptiveQuestion` hook
- `frontend/src/pages/ExamTaking/ExamLayout.tsx`: Adaptive mode support — detects `session.adaptive`, fetches `next-question` query, auto-submits on `done: true`, invalidates query after each answer save, renders single-question view for adaptive sessions
- `frontend/src/pages/ExamTaking/ExamTopBar.tsx`: Added `adaptive` prop; shows `session.questionsAnswered` instead of `N of M` when adaptive; hides mobile navigator button in adaptive mode

### i18n

Added `session.questionsAnswered` key to all three locales:
- `en.json`: `"Questions answered: {{count}}"`
- `kk.json`: `"Жауапталған сұрақтар: {{count}}"`
- `ru.json`: `"Отвечено вопросов: {{count}}"`

### Documentation

- `docs/requirements/README.md`: FR-BB72 status → `Implemented`
- `docs/requirements/FR-BB72.Adaptive-difficulty.md`: Status → `Implemented`

## Test Results

```
ok  github.com/bilimbaga/bilimbaga/internal/exams   (all 9 NextDifficulty tests + existing passing)
ok  github.com/bilimbaga/bilimbaga/internal/sessions (all 5 service adaptive + 6 handler adaptive tests + existing passing)
All 26 backend packages: PASS
Frontend TypeScript: no errors in production code
```

## Migration

`backend/migrations/026_adaptive_exam.up.sql` — applied successfully via `docker compose run --rm api ./bin/api migrate`.

## AC Coverage

| AC | Status |
|----|--------|
| AC-1: `adaptive` field, locked after publish | Implemented |
| AC-2: Publish validates ≥5 per difficulty (422) | Implemented |
| AC-3: `GET /next-question` endpoint | Implemented |
| AC-4: `NextDifficulty` algorithm | Implemented |
| AC-5: Idempotent `current_question_id` | Implemented |
| AC-6: Answer save clears current_question_id | Implemented |
| AC-7: Frontend hides count, shows answered | Implemented |
| AC-8: Non-adaptive sessions unchanged | Verified (all existing tests pass) |

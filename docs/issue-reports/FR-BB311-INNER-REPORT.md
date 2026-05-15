# FR-BB311 — Grading Engine — Inner Report

## Summary

Implemented the grading engine for BilimBaga exam sessions (FR-BB311).

## Files Created

| File | Purpose |
|------|---------|
| `backend/migrations/016_session_question_scores.up.sql` | New `session_question_scores` table + `grading_status` ENUM |
| `backend/migrations/016_session_question_scores.down.sql` | Rollback migration |
| `backend/internal/sessions/grading.go` | `GradingEngine` interface, `DefaultGradingEngine`, scoring helpers |
| `backend/internal/sessions/grading_test.go` | 22 unit tests covering all four question types + edge cases |

## Files Modified

| File | Change |
|------|--------|
| `backend/internal/sessions/model.go` | Added `SessionQuestionScore` struct |
| `backend/internal/sessions/repository.go` | Added `engine GradingEngine` to `postgresRepository`; `NewRepository` now accepts `GradingEngine`; replaced CTE grading in `SubmitSession` and `AutoSubmitSession` with `engine.Grade`; fixed `'short_text'` → `'shorttext'` type check |
| `backend/internal/sessions/autojob.go` | `AutoSubmitJob` now accepts `GradingEngine`; `processOneExpiredSession` uses `engine.Grade`; removed obsolete `gradeAndUpdateScore`; fixed `'short_text'` → `'shorttext'` |
| `backend/cmd/api/main.go` | Wires `sessions.NewGradingEngine()` into repository and autojob |
| `docs/requirements/FR-BB311.Grading-engine.md` | All AC checkboxes marked; status → Implemented |
| `docs/requirements/README.md` | Status column → Implemented |

## Acceptance Criteria Verification

| AC | Coverage |
|----|----------|
| AC-1 | `TestGradeSingleOrTrueFalse` (4 subtests) — correct/wrong/no/multi selection |
| AC-2 | `TestGradeMultipleChoice` (6 subtests) — exact match, partial credit, all wrong, no-correct error |
| AC-3 | `TestGradeLikert` (4 subtests) + 3 edge case tests — positive/negative/neutral/zero-weight/all-positive/clamp |
| AC-4 | `TestGradeQuestion_ShortText_PendingManual` |
| AC-5 | Implemented in `Grade` — INSERT per question in loop |
| AC-6 | `TestScorePct_Rounding` + `TestScorePct_ZeroMax` |
| AC-7 | `Grade` performs all writes within the caller-owned `*sqlx.Tx` |
| AC-8 | `passed := scorePct >= exam.PassingScorePct` in `Grade` |
| AC-9 | `GradingEngine.Grade(tx *sqlx.Tx, sessionID string) (float64, bool, error)` — no HTTP deps |
| AC-10 | `gradeSingleOrTrueFalse` and `gradeMultipleChoice` return errors on missing correct options |

## Test Results

```
ok  github.com/bilimbaga/bilimbaga/internal/sessions  0.806s
ok  github.com/bilimbaga/bilimbaga/internal/...       (all 15 packages)
```

All 22 new grading tests pass. Full backend suite (15 packages) passes with 0 failures.

## DB Migration Status

Migration 016 applied to live database:
- `grading_status` ENUM: `'graded'`, `'pending_manual'`, `'ai_graded'`
- `session_question_scores` table with composite PK, FK constraints, and two indexes

## Side-fixes Included

- Fixed pre-existing bug: `q.type = 'short_text'` → `q.type = 'shorttext'` in `SubmitSession`, `AutoSubmitSession`, and `processOneExpiredSession` (the DB CHECK constraint uses `'shorttext'` not `'short_text'`).

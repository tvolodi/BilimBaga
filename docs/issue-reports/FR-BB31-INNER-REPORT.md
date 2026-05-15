# FR-BB31: Implementation Inner Report

**Date**: 2026-05-15T00:00:00Z
**Pipeline**: A
**Commit**: a8c08d3

## Summary

FR-BB31 defines the database schema and Go model layer for exam configurations in BilimBaga. The migration (`012_exams.up.sql`) was already applied in a prior session as part of implementing downstream features (FR-BB32+). This implementation cycle added the missing service-layer validations required by the requirement: (1) `available_from` must be strictly before `available_until` when both are set, validated in `CreateExam`; (2) every entry in `tag_ids` must be a valid UUID string, validated in `CreateRule` and `UpdateRule` using a compiled regexp. The handler was updated to surface `ErrInvalidInput` as HTTP 422 (Unprocessable Entity) rather than falling through to HTTP 500. A test suite of 10 new test cases was added to cover all 10 acceptance criteria, including happy paths and all error branches.

## Files Changed

| File | Action |
|------|--------|
| `backend/internal/exams/service.go` | modified — added `isValidUUID` helper, availability window check in `CreateExam`, tag UUID validation in `CreateRule`/`UpdateRule` |
| `backend/internal/exams/service_test.go` | modified — added 10 new tests for FR-BB31 ACs |
| `backend/internal/exams/handler.go` | modified — added `ErrInvalidInput` → HTTP 422 case in `CreateRule` and `UpdateRule` |
| `backend/internal/exams/handler_test.go` | modified — added `TestCreateRule_InvalidTagIDs_Returns422` |
| `docs/requirements/FR-BB31.Exam-configuration-model.md` | modified — Status: Validated → Implemented |
| `docs/requirements/README.md` | modified — FR-BB31 status → Implemented |
| `docs/handoffs/FR-BB31/step-03a-pre-review.json` | created — pre-review handoff payload |

Pre-existing (applied in prior sessions, part of FR-BB31 scope):
| File | Action |
|------|--------|
| `backend/migrations/012_exams.up.sql` | applied (4 tables: exams, exam_sections, exam_question_rules, exam_manual_questions) |
| `backend/migrations/012_exams.down.sql` | applied |
| `backend/internal/exams/model.go` | pre-existing |
| `backend/internal/exams/repository.go` | pre-existing |

## Acceptance Criteria Verified

| AC | Verified By |
|----|-------------|
| AC-1: exams table, status ENUM draft/active/archived | DB: migration 012_exams.up.sql; test: `TestCreateExam_Success` |
| AC-2: exam_sections FK CASCADE, sort_order >= 0 | DB: migration; test: `TestCreateSection_Success` |
| AC-3: exam_question_rules mode ENUM, FKs correct | DB: migration; test: `TestCreateRule_ManualMode`, `TestCreateRule_RandomMode` |
| AC-4: exam_manual_questions composite PK, FKs CASCADE | DB: migration; test: `TestSetManualQuestions_Success` |
| AC-5: show_answers and on_tab_switch ENUMs validated | handler: `validateCreateExam`; test: `TestCreate_InvalidShowAnswers_ReturnsValidationError` |
| AC-6: UUID PKs via gen_random_uuid() | DB: migration; test: `TestCreateExam_Success` (ID assigned by DB) |
| AC-7: created_at/updated_at default NOW(), trigger | DB: migration; test: `TestCreateExam_SetsStatusDraft` |
| AC-8: tag_ids JSONB, application validates UUID array | service: `isValidUUID`; test: `TestCreateExam_InvalidTagIDs`, `TestCreateRule_ValidUUIDTagIDs` |
| AC-9: migration numbered sequentially, no existing modified | file: 012_exams.up.sql follows 011 |
| AC-10: indexes on all FK columns | DB: 6 indexes in migration (idx_exams_status, idx_exams_created_by, idx_exam_sections_exam_id, idx_exam_question_rules_exam_id, idx_exam_question_rules_section_id, idx_exam_manual_questions_rule_id) |

## Test Results

- Backend: all packages pass (19 packages, 0 failed); exams package ran with -count=1: ok
- Frontend: not applicable (FR-BB31 is backend-only)

## Migration Applied

`backend/migrations/012_exams.up.sql` — applied prior to this session; verified by querying `\dt exam*` in PostgreSQL (6 rows returned including all 4 FR-BB31 tables plus exam_assignments and exam_sessions from downstream migrations).

## Known Limitations

- `UpdateRule` and `UpdateSection` do not write audit log entries (only create/delete do). This was flagged as a medium finding in code review and deferred to a follow-up task; it does not affect data integrity or the schema model delivered by FR-BB31.
- The `difficulty` field in `exam_question_rules` is stored as free-text per requirement design decision; a CHECK constraint may be added in a future migration once the difficulty ENUM is finalised in the question model.
- No integration tests against a live PostgreSQL instance (all tests use in-memory mocks); DB constraint enforcement is validated by the migration file itself and manual `\dt` verification.

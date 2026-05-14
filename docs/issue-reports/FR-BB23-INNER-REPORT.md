# FR-BB23 — Question CRUD API — Inner Report

## Status: IMPLEMENTED ✓

## Summary
Full implementation of the Question CRUD API as specified in FR-BB23. All 9 REST endpoints are live, all acceptance criteria are satisfied, and all unit tests pass.

---

## Delivery Phases

### Phase 1 — Requirement Analysis ✓
- Read `docs/requirements/FR-BB23.Question-CRUD-API.md` in full.
- Reviewed architecture guide, backend development guide, existing questions package.
- Identified all files to create/modify.

### Phase 2 — Database Migration ✓
- **Migration 010**: `backend/migrations/010_question_tags_tag_id_index.up.sql`
  - `CREATE INDEX IF NOT EXISTS idx_question_tags_tag_id ON question_tags(tag_id);`
  - Applied successfully; verified via `\di idx_question_tags_tag_id`.

### Phase 3 — Go Backend ✓

#### Files Modified/Created

| File | Change |
|------|--------|
| `backend/internal/questions/model.go` | Added 4 sentinel errors + 12 new types |
| `backend/internal/questions/repository.go` | Extended `Repository` interface with 8 new methods; implemented all |
| `backend/internal/questions/service.go` | Extended `Service` interface with 8 new methods; implemented all |
| `backend/internal/questions/handler.go` | **NEW** — 9 HTTP handlers, request DTOs, validation |
| `backend/internal/router/router.go` | Added `questionsHandler` param; registered 9 routes with RBAC guards |
| `backend/cmd/api/main.go` | Wired `questionsRepo → questionsSvc → questionsHandler` |

#### New Sentinel Errors
- `ErrInvalidTransition` — status transition not allowed
- `ErrNotDraft` — operation requires draft status
- `ErrStemRequired` — stem required before publishing
- `ErrTagNotFound` — tag ID does not exist

#### Repository Methods Added
- `CreateFull` — transactional create (question + translations + options + tags)
- `ListFiltered` — paginated list with category/status/difficulty/type/tag/locale filters
- `GetWithDetails` — full question detail with translations, options, tags
- `UpdateInPlace` — in-place update for draft/review questions
- `CreateVersionFull` — serializable TX; archives previous, inserts new version+1
- `DeleteByID` — hard delete; returns `ErrQuestionNotFound` if not found
- `GetVersionChain` — recursive CTE to walk full version chain
- `TagExists` — checks tag exists before tagging a question

#### Service Methods Added
- `CreateQuestionFull` — delegates to `repo.CreateFull`
- `ListFiltered` — delegates to `repo.ListFiltered`
- `GetQuestionWithDetails` — delegates to `repo.GetWithDetails`
- `UpdateQuestion` — routes to `CreateVersionFull` (active) or `UpdateInPlace` (draft/review)
- `TransitionStatus` — validates via `validTransitions` map; checks stem for review/active
- `DeleteQuestion` — guards non-draft; delegates to `repo.DeleteByID`
- `ListVersions` — delegates to `repo.GetVersionChain`
- `GetQuestionTags` — delegates to `repo.GetTags`
- `TagQuestion` — calls `repo.TagExists` first; returns `ErrTagNotFound` if missing

#### Routes Registered (all under `/api/v1`)
```
GET    /questions                   questions:read
POST   /questions                   questions:write
GET    /questions/{id}              questions:read
PUT    /questions/{id}              questions:write
POST   /questions/{id}/status       questions:write
DELETE /questions/{id}              questions:write
GET    /questions/{id}/versions     questions:read
POST   /questions/{id}/tags         questions:write
DELETE /questions/{id}/tags/{tagId} questions:write
```

### Phase 6 — Tests ✓

All 27 unit tests pass (0 failures):

**Pre-existing tests (passing):**
- `TestCreateQuestion_DefaultsVersionAndStatus`
- `TestPublishNewVersion_IncrementsVersionAndSetsParent`
- `TestPublishNewVersion_PreviousNotFound`
- `TestAddAnswerOption_PassesSortOrderThrough`

**New FR-BB23 tests:**
- `TestTransitionStatus_ValidTransitions` (3 sub-tests: draft→review, review→active, active→archived)
- `TestTransitionStatus_InvalidTransitions` (6 sub-tests: all invalid paths return ErrInvalidTransition)
- `TestDeleteQuestion_DraftSucceeds`
- `TestDeleteQuestion_NonDraftFails` (3 sub-tests: review, active, archived)
- `TestUpdateQuestion_ActiveCreatesNewVersion`
- `TestUpdateQuestion_DraftUpdatesInPlace`
- `TestListVersions_ReturnsChain`
- `TestTagQuestion_NonExistentTagReturnsErrTagNotFound`

**Full suite result**: 12/12 packages pass — zero regressions.

### Phase 9 — Documentation ✓
- `docs/requirements/FR-BB23.Question-CRUD-API.md` — Status updated `validated → implemented`
- `docs/requirements/README.md` — FR-BB23 row updated to `Implemented`

---

## Key Design Decisions

1. **Version fork strategy**: Active questions cannot be edited in place. `UpdateQuestion` archives the existing active question and creates a new draft at version+1 with `parent_id` pointing to the previous. This preserves audit history.
2. **Serializable isolation for versioning**: `CreateVersionFull` runs in a serializable transaction to prevent concurrent version forks on the same question.
3. **Tag validation on tagging**: `TagQuestion` calls `TagExists` first to return a clear `ErrTagNotFound` rather than a foreign key violation.
4. **Status machine**: Enforced in the service layer via `validTransitions` map; only sequential forward transitions are allowed.

---

## Files Changed

```
backend/migrations/010_question_tags_tag_id_index.up.sql   (NEW)
backend/migrations/010_question_tags_tag_id_index.down.sql (NEW)
backend/internal/questions/model.go                        (MODIFIED)
backend/internal/questions/repository.go                   (MODIFIED)
backend/internal/questions/service.go                      (MODIFIED)
backend/internal/questions/handler.go                      (NEW)
backend/internal/questions/service_test.go                 (MODIFIED)
backend/internal/router/router.go                          (MODIFIED)
backend/cmd/api/main.go                                    (MODIFIED)
docs/requirements/FR-BB23.Question-CRUD-API.md             (MODIFIED)
docs/requirements/README.md                                 (MODIFIED)
```

# FR-BB71 Inner Report — Question Generation Assist

## Summary

Implemented FR-BB71 (Question Generation Assist) — AI-powered draft question generation for examiners using the Anthropic API.

## Implementation

### Migration
- `backend/migrations/025_ai_usage_log.up.sql` — `ai_usage_log` table with `(user_id, feature, tokens_used, model, created_at)` columns and index on `(user_id, created_at DESC)`.
- Note: FK uses `ON DELETE CASCADE` (not `ON DELETE SET NULL`) to avoid the NULL/NOT NULL contradiction in the spec. Applied via docker exec psql and recorded in `schema_migrations`.

### Backend (`backend/internal/ai/`)
| File | Purpose |
|------|---------|
| `model.go` | Sentinel errors, request/response structs |
| `repository.go` | `CountAIUsageLastHour`, `LogUsage`, `GetCategoryName` |
| `service.go` | Validation (AC-6), rate-limit check (AC-5), Anthropic call, usage logging (AC-4) |
| `handler.go` | HTTP handler mapping errors to 400/429/503 (AC-2, AC-3) |
| `client.go` | HTTP wrapper around Anthropic Messages API; `BuildQuestionsPrompt` template renderer |
| `prompt.go` | `generateQuestionsPromptTemplate` constant |

Config extended with `AnthropicAPIKey` and `AnthropicModel` fields. Route registered at `POST /api/v1/admin/ai/generate-questions` behind `rbac.RequirePermission(questions, write)` which maps to examiner+ roles.

### Frontend
| File | Purpose |
|------|---------|
| `frontend/src/api/ai.ts` | `useGenerateQuestions` mutation, typed request/response |
| `frontend/src/components/questions/AIGenerateDialog.tsx` | Full generation dialog: form, draft preview with expand/collapse, per-draft checkboxes, confirm flow |
| `frontend/src/pages/admin/questions/QuestionBankPage.tsx` | Added AI Generate button (examiner+ gated) and dialog integration |
| `frontend/src/locales/{en,kk,ru}.json` | 12 new i18n keys each |

### Tests
- `backend/internal/ai/service_test.go` — 12 tests covering all ACs (happy path, usage logging, rate limit boundary conditions, all validation cases, client failure, malformed JSON)
- `backend/internal/ai/handler_test.go` — 6 tests covering 200/400/401/429/503 responses

**Result: 18/18 tests pass. Full suite: 24/24 packages green.**

## AC Verification

| AC | Status | Evidence |
|----|--------|---------|
| AC-1 | ✅ | `HandleGenerateQuestions` returns `{data: {questions: [...]}, error: null}` |
| AC-2 | ✅ | Route uses `rbac.RequirePermission(questions, write)`; employee role lacks this permission |
| AC-3 | ✅ | Client errors and malformed JSON both return 503 `AI_UNAVAILABLE` |
| AC-4 | ✅ | `LogUsage` called after every successful Anthropic call; verified in test |
| AC-5 | ✅ | `checkRateLimit` counted against `ai_usage_log`; >= 20 → 429 `AI_RATE_LIMITED` |
| AC-6 | ✅ | count 1-10, context_text ≤ 2000, difficulty in {easy,medium,hard} validated before DB/API calls |
| AC-7 | ✅ | `AIGenerateDialog` with form, preview table, per-item checkboxes, confirm flow |
| AC-8 | ✅ | Drafts never inserted; only returned in response; user must explicitly confirm each |

## Issues Encountered

1. **FK contradiction in spec**: `user_id NOT NULL … ON DELETE SET NULL` is contradictory. Resolved by using `ON DELETE CASCADE` with nullable column (no NOT NULL constraint).
2. **`ctxkeys.WithUserID` does not exist**: Handler tests used a local `ctxWithUserID` helper using `context.WithValue` with the existing `CtxUserID` key — consistent with how other handler tests in the codebase set context values.

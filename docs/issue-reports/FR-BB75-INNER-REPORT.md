# FR-BB75: Implementation Inner Report

**Date**: 2026-05-16T00:00:00Z
**Pipeline**: A
**Commit**: 4a0115f3999b4c31ad4577c8127f20939ab94e2a

## Summary

Implemented FR-BB75 (Loyalty Profile Narrative), a Phase 7 AI layer feature that generates on-demand 2–3 sentence values profile narratives for loyalty-track exam sessions. An admin HTTP endpoint (`GET /api/v1/admin/ai/loyalty-summary/:sessionId`) fetches Likert responses, applies polarity inversion, builds an anonymized prompt, calls Anthropic Claude, logs usage to `ai_usage_log` under the requesting admin's user ID (no employee PII in prompt), and returns the narrative. The frontend surfaces the result in a new `ValuesProfileSection` on the Employee Record page, hidden automatically for non-loyalty sessions.

## Files Changed

| File | Action |
|------|--------|
| `backend/internal/ai/model.go` | modified — added types, sentinel errors, feature const |
| `backend/internal/ai/repository.go` | modified — 3 new interface methods + Postgres implementations |
| `backend/internal/ai/prompt.go` | modified — added `BuildLoyaltyPrompt` |
| `backend/internal/ai/service.go` | modified — extended interface; `GetLoyaltyNarrative` implementation |
| `backend/internal/ai/handler.go` | modified — added HTTP handler for new route |
| `backend/internal/ai/handler_test.go` | modified — mock updated |
| `backend/internal/ai/service_test.go` | modified — 8 unit tests |
| `backend/internal/router/router.go` | modified — registered new route |
| `backend/internal/reports/model.go` | modified — `ExamCategoryTrack *string` on `SessionRecord` |
| `backend/internal/reports/repository.go` | modified — LEFT JOIN categories in session history query |
| `docs/requirements/FR-BB75.Loyalty-profile-narrative.md` | modified — status updated to validated |
| `docs/requirements/README.md` | modified — index entry updated |
| `frontend/src/api/ai.ts` | modified — added `fetchLoyaltyNarrative` |
| `frontend/src/api/employees.ts` | modified — added `exam_category_track` to `SessionRecord` |
| `frontend/src/components/ui/card.tsx` | modified — added `CardTitle` export |
| `frontend/src/pages/admin/EmployeeRecordPage.tsx` | modified — added `ValuesProfileSection` |
| `frontend/src/locales/en.json` | modified — 4 new `employee_record.*` i18n keys |
| `frontend/src/locales/ru.json` | modified — 4 new i18n keys |
| `frontend/src/locales/kk.json` | modified — 4 new i18n keys |
| `frontend/src/pages/admin/__tests__/AIInsightsCard.test.tsx` | modified — updated for mock changes |
| `frontend/e2e/loyalty-narrative.spec.ts` | created — 4 Playwright e2e tests |

## Acceptance Criteria Verified

| AC | Verified By |
|----|-------------|
| GET endpoint returns narrative for loyalty session | test: TestGetLoyaltyNarrative_Success |
| Returns 404 for non-existent session | test: TestGetLoyaltyNarrative_SessionNotFound |
| Returns 400 for non-loyalty session | test: TestGetLoyaltyNarrative_NotLoyaltySession |
| Returns 403 for admin outside department | test: TestGetLoyaltyNarrative_Forbidden |
| Polarity inversion applied before prompt | test: TestGetLoyaltyNarrative_PolarityInversion |
| No PII in prompt | test: TestBuildLoyaltyPrompt_Anonymized |
| Usage logged with admin user_id | test: TestGetLoyaltyNarrative_UsageLogged |
| ValuesProfileSection hidden for non-loyalty sessions | e2e: loyalty-narrative.spec.ts |

## Test Results

- Backend: 8 passed, 0 failed (ai package unit tests)
- Frontend: 4 passed, 0 failed (Playwright e2e)

## Migration Applied

none

## Known Limitations

- Playwright e2e tests require a running dev stack with a seeded loyalty-track session; they are marked to run in the `e2e` suite only.
- Narrative is not cached; each request triggers a fresh Anthropic API call.

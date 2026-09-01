---
id: ISS-050
title: seed-test-env.ts fails against live test environment (rate limiting, wrong question type/field names, invalid likert_polarity)
status: resolved
severity: medium
layer: config
module: questions
tags: [seed-script, rate-limit, 429, ERR_VALIDATION, ERR_INTERNAL, likert_polarity, question_type, tsx]
created: 2026-09-01
resolved: 2026-09-01
recurrence_count: 1
related_issues: []
regression_test: null
---

## Symptom

Running `npx tsx scripts/seed-test-env.ts` against `https://bilimbaga-test.ai-dala.com` failed at three successive points as each was fixed:

1. `Fatal error: Error: Could not login as examiner2@bilimbaga-test.local with temp pass` (scripts/seed-test-env.ts:248)
2. `Fatal error: Error: Failed to create question "...": ERR_VALIDATION`
3. `Fatal error: Error: Failed to create question "...": ERR_INTERNAL`

## Root Cause

Three independent bugs in `scripts/seed-test-env.ts`, surfaced sequentially as each blocker was cleared:

1. **No rate-limit handling.** The script fires logins/creates back-to-back for ~15 users with no delay or retry. Once enough requests land inside the API's rate-limit window, `POST /api/v1/auth/login` returns 429, which `loginUser()` treated as a plain auth failure (returned `null`), and `ensureUser()` treated any `null` as fatal instead of retrying. Confirmed via API logs on `bilimbaga-test-api-1`: `status_code=429` right after a burst of 200s.

2. **Wrong wire values sent to `POST /api/v1/questions`.**
   - `type` field: the script sent its own internal labels (`single_choice`, `multiple_choice`, `true_false`, `short_text`) directly, but the API's `validateCreateRequest` (backend/internal/questions/handler.go:105) only accepts `single`, `multiple`, `truefalse`, `likert`, `shorttext`.
   - Answer option translation body: the script sent `translations: { en: { body: text } } }`, but the API's `answerTranslationReq` DTO (backend/internal/questions/handler.go:65) expects JSON key `text`, not `body`.
   Both together produced `ERR_VALIDATION` (422) on the very first question.

3. **Invalid `likert_polarity` value.** After fixing #2, the script got further and hit an unhandled 500 (`ERR_INTERNAL`) on the first likert question. The `answer_options.likert_polarity` column (backend/migrations/009_questions.up.sql:39) has `CHECK (likert_polarity IN ('positive','negative'))` — only two values are legal, NULL is allowed but `'neutral'` is not. The script's 5-point Likert scale included a middle option with `likert_polarity: 'neutral'`, which violated the CHECK constraint at insert time inside `applySubObjects` (backend/internal/questions/repository.go). The handler wraps all internal errors into a generic `ERR_INTERNAL` with no server-side logging of the underlying SQL error, making this hard to diagnose from logs alone — confirmed only by reading the migration's CHECK constraint directly.

This surfaced only when actually exercising the live test environment end-to-end (not covered by any existing test), following deploy of migration 030 (see `docs/handoffs/infra-seed-20260901/`) which unblocked the earlier admin-account lockout that had been masking these three bugs entirely.

## Fix Applied

All fixes in `scripts/seed-test-env.ts`:

1. Added `fetchWithRateLimitRetry()` wrapping every request in `apiPost`/`apiPut`/`apiGet` — retries up to 5 times on HTTP 429 with exponential backoff (honoring `Retry-After` header when present), instead of failing immediately.
2. Added `QUESTION_TYPE_WIRE` map translating the script's descriptive type labels to the API's wire format (`single_choice`→`single`, `multiple_choice`→`multiple`, `true_false`→`truefalse`, `short_text`→`shorttext`, `likert`→`likert`) and applied it in `createQuestion()`'s request body. Changed answer-option translation key from `body` to `text` to match the API DTO.
3. Changed the Likert "Neutral" option's `likert_polarity` from `'neutral'` to `null` to satisfy the DB CHECK constraint.

## Files Changed

| File | Change |
|------|--------|
| scripts/seed-test-env.ts | Added rate-limit retry helper for all API calls; fixed question `type` wire values; fixed answer-option translation JSON key (`body`→`text`); fixed likert "Neutral" option's `likert_polarity` (`'neutral'`→`null`) |

## Regression Test

None added — this is a one-off operational seed script for the live test environment, not covered by an existing test harness (no `scripts/` test directory exists in this repo). Verified instead by two full successful live runs against `https://bilimbaga-test.ai-dala.com` (first run created all data, second confirmed full idempotency with exit code 0 and no new writes needed beyond expected in-progress session state).

## Resolution Results
- Tests: N/A (no scripts/ test harness in this repo; verified via live end-to-end run instead)
- Migration applied: no (this issue is unrelated to schema; migration 030 was applied separately, see docs/handoffs/infra-seed-20260901/)
- Build clean: yes (`cd backend && go build ./...` — no backend files changed by this fix, sanity-checked anyway)
- Live seed run: PASSED (exit 0), confirmed idempotent on re-run

## Recurrence Log
| Date | Trigger | Action Taken |
|------|---------|-------------- |

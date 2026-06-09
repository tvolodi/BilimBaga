# FR-BB315 — Exam Step 4: Eligible Question Counts per Rule

**Status**: uat-verified  
**Phase**: 3 (enhancement to 3.12)  
**Depends on**: FR-BB312, FR-BB32, FR-BB22

## Summary

On the Exam Edit wizard's Step 4 "Review & Publish" tab, each question rule in the summary section currently shows only mode, count, and difficulty. The user has no way to know whether the question bank can actually satisfy a rule until they attempt to publish and receive a 422 error. This requirement adds a live "eligible questions" count badge next to each rule in the Step 4 summary, fetched from a new read-only API endpoint. The badge is colour-coded: green when sufficient questions are available, amber when available but insufficient, and red when zero questions match the rule's filters. This gives exam authors actionable feedback before they click Publish.

## Scope

| Layer | Items |
|-------|-------|
| Database | No schema changes — reuses existing `questions` table and `exam_question_rules` table |
| API endpoints | `GET /api/v1/exams/{id}/rules/eligible-counts` — returns per-rule eligible question counts |
| Frontend pages/components | `src/pages/ExamWizard/Step4Review.tsx` — adds eligible-count badges to the rules list |
| i18n keys | `exam.wizard.step4.eligibleCount`, `exam.wizard.step4.eligibleCountLoading`, `exam.wizard.step4.eligibleCountError`, `exam.wizard.step4.ruleStatusOk`, `exam.wizard.step4.ruleStatusInsufficient`, `exam.wizard.step4.ruleStatusNone` |

## Acceptance Criteria

- **AC1**: `GET /api/v1/exams/{id}/rules/eligible-counts` returns HTTP 200 with a JSON body `{ data: { counts: [{ rule_id, eligible }] }, error: null }` listing one entry per rule defined on the exam. Requires `exams:read` permission.
- **AC2**: For a `random` rule, the `eligible` count equals the number of active questions matching the rule's `category_id`, `difficulty`, and `tag_ids` filters (same logic as `CountAvailableForRule` used during Publish validation).
- **AC3**: For a `manual` rule, the `eligible` count equals the total number of active questions explicitly linked to that rule (i.e., questions in `exam_manual_questions` whose corresponding `questions.status = 'active'`).
- **AC4**: When Step 4 is rendered, the eligible counts are fetched automatically via a React Query query; a spinner/loading state is shown next to the rules list while the request is in flight.
- **AC5**: Each rule row in Step 4 displays the eligible count as a badge: green (`eligible >= rule.count`), amber (`0 < eligible < rule.count`), or red (`eligible === 0`).
- **AC6**: If the fetch fails (network error or non-2xx), the badge area shows a small error indicator (icon + i18n key `exam.wizard.step4.eligibleCountError`) without blocking the rest of the Step 4 UI or the Publish button.
- **AC7**: The endpoint returns `404` with `{ data: null, error: { code: "EXAM_NOT_FOUND", message: "..." } }` when the exam ID does not exist.
- **AC8**: Counts are computed at request time with no caching — they reflect the current state of the question bank at the moment the request is made.
- **AC9**: When the exam exists but has no question rules defined, the endpoint returns HTTP 200 with `{ data: { counts: [] }, error: null }`, and the frontend Step 4 rules list shows no rule rows (empty state unchanged).

## Technical Notes

### Database

No DDL changes required. The query reuses the existing `questions` table (filtered on `status = 'active'`) and `exam_question_rules` / `exam_manual_questions` tables already present from migrations 009, 012, and 015.

### API Contract

**Request**

```
GET /api/v1/exams/{id}/rules/eligible-counts
Authorization: Bearer <token>
```

**Success response** (`200 OK`)

```json
{
  "data": {
    "counts": [
      {
        "rule_id": "06f834c1-xxxx-4xxx-yxxx-xxxxxxxxxxxx",  // UUID v4
        "eligible": 12
      },
      {
        "rule_id": "9a3b1c2d-xxxx-4xxx-yxxx-xxxxxxxxxxxx",  // UUID v4
        "eligible": 0
      }
    ]
  },
  "error": null
}
```

**Error responses**

| Status | code | Condition |
|--------|------|-----------|
| 404 | `EXAM_NOT_FOUND` | Exam ID does not exist |
| 401 | `UNAUTHORIZED` | Missing or invalid JWT |
| 403 | `FORBIDDEN` | Caller lacks `exams:read` permission |

### Go Implementation Notes

**Package**: `internal/exams`

- **Repository** (`repository.go`): Add `CountAvailableForManualRule(ctx context.Context, ruleID string) (int, error)` to `Repository` interface and implement on `postgresRepository`. Query: `SELECT COUNT(*) FROM exam_manual_questions emq JOIN questions q ON q.id = emq.question_id WHERE emq.rule_id = $1 AND q.status = 'active'`.
- **Service** (`service.go`): Add `GetEligibleCounts(ctx context.Context, examID string) ([]RuleEligibleCount, error)` to `Service` interface. It first calls `r.repo.GetByID(ctx, examID)`; if that returns `ErrNotFound`, propagate `ErrNotFound` immediately. Only after confirming the exam exists does it call `ListRulesForExam`, then for each rule call `CountAvailableForRule` (random) or `CountAvailableForManualRule` (manual).
- **Model** (`model.go`): Add `RuleEligibleCount struct { RuleID string \`json:"rule_id"\`; Eligible int \`json:"eligible"\` }`. `RuleID` is a UUID v4 string.
- **Handler** (`handler.go`): Add `GetEligibleCounts(w http.ResponseWriter, r *http.Request)` handler. Reads `{id}` URL param, calls `svc.GetEligibleCounts`, writes standard `{ data: { counts: [...] }, error: null }` response.
- **Router** (`internal/router/router.go`): Register `GET /exams/{id}/rules/eligible-counts` under the `exams:read` permission middleware, alongside the existing exam read routes.

No new middleware is required. Existing JWT + RBAC middleware covers this endpoint.

### Frontend Implementation Notes

**Route/component**: `src/pages/ExamWizard/Step4Review.tsx`

- **New React Query hook** in `src/api/exams.ts`: `useEligibleCounts(examId: string)` — calls `GET /api/v1/exams/{examId}/rules/eligible-counts`. React Query key: `['exams', examId, 'eligibleCounts']`. Enabled only when `examId` is non-empty.
- **New TypeScript type** in `src/api/exams.ts`: `export interface RuleEligibleCount { rule_id: string; eligible: number }`.
- **Step4Review**: Import `useEligibleCounts`. In the rules list section, for each rule look up the matching count from the query result. Display a `<Badge>` (shadcn/ui) with text `{eligible} {t('exam.wizard.step4.eligibleCount')}` coloured by status:
  - `eligible >= rule.count` → green: `className="bg-green-100 text-green-800"`
  - `0 < eligible < rule.count` → amber (`bg-amber-100 text-amber-800`)
  - `eligible === 0` → destructive (`bg-red-100 text-red-800`)
- While loading: show a `<Loader2 className="h-3 w-3 animate-spin" />` inline next to the rule text.
- On error: show a small `<AlertTriangle className="h-3 w-3" />` icon with the `eligibleCountError` i18n string as a tooltip or inline label.
- **i18n keys** (add to `en.json`, `ru.json`, `kk.json` under `exam.wizard.step4`):
  - `eligibleCount`: `"eligible"` (used as unit label, e.g. "12 eligible")
  - `eligibleCountLoading`: `"Checking availability…"`
  - `eligibleCountError`: `"Could not load counts"`
  - `ruleStatusOk`: `"Sufficient"`
  - `ruleStatusInsufficient`: `"Insufficient"`
  - `ruleStatusNone`: `"No matching questions"`

## Out of Scope

- Caching or real-time invalidation of eligible counts when the question bank changes (counts are fetched fresh on each Step 4 render).
- Showing eligible counts in Step 2 (Question Rules) or anywhere other than Step 4.
- Blocking the Publish button based on eligible counts (the existing Publish validation at the service layer already prevents publishing when rules are unsatisfied; this feature only adds visibility).
- Filtering or sorting rules by eligibility status.
- Counts for adaptive exam difficulty bands beyond the per-rule level (those are handled by FR-BB72).

## Test Strategy

- **Backend unit tests** (`service_test.go`): add `TestGetEligibleCounts_*` table-driven tests covering: all random rules satisfied, some rules with zero eligible, mix of manual and random rules, exam not found.
- **Backend unit tests** (`handler_test.go`): add `TestGetEligibleCountsHandler_*` covering: 200 with correct JSON shape, 404 for unknown exam.
- **Backend repository test** (integration, if integration harness exists): `TestCountAvailableForManualRule` verifying count against seed data.
- **Frontend component tests** (`Step4Review.test.tsx` or similar): mock `useEligibleCounts` to return known counts and assert badge colours/text render correctly for each state (ok / insufficient / zero / loading / error).

# FR-BB75 — Loyalty Profile Narrative

## Metadata
| Field | Value |
|-------|-------|
| ID | FR-BB75 |
| Phase | 7 — AI Layer |
| Priority | 2 |
| Status | implemented |
| Depends On | FR-BB311, FR-BB53, FR-BB58, FR-BB71 |

## Description
Provides department admins with an AI-generated 2–3 sentence narrative describing an employee's values profile derived from their Likert-scale responses on a loyalty-track exam session. The narrative is generated on demand (not automatically), uses only anonymized response weights (no PII in the prompt), and is surfaced in the employee record page. Results are not cached, ensuring the narrative always reflects the most recent session data.

## Acceptance Criteria
- [ ] AC-1: `GET /api/v1/admin/ai/loyalty-summary/:sessionId` validates that the session belongs to an exam in a category with `track = 'loyalty'`; sessions from non-loyalty exams return `400` with error code `NOT_A_LOYALTY_SESSION`.
- [ ] AC-2: The endpoint requires JWT auth with role ≥ `department_admin`; requests from examiners with no department match or from employees return `403 Forbidden`.
- [ ] AC-3: The prompt sent to the Anthropic API contains only anonymized Likert option weights and polarity labels — no employee name, user ID, email, or any other personally identifying information is included in the prompt text.
- [ ] AC-4: The endpoint returns a narrative of 2–3 complete sentences describing the employee's apparent values and workplace orientation based on the Likert response pattern; the narrative does not contain any identifying language (no "this employee" — use "this profile" or "the responses indicate").
- [ ] AC-5: Every call to this endpoint logs to `ai_usage_log` with `feature = "loyalty_narrative"`, the calling admin's `user_id`, `tokens_used` from the API response, and `model` set to the Anthropic model name used; the session employee's user ID is NOT logged in `ai_usage_log` to avoid tying AI usage records to individual employees.
- [ ] AC-6: When the Anthropic API is unavailable, the endpoint returns `503` with error code `AI_UNAVAILABLE`; no partial or cached response is returned.
- [ ] AC-7: The frontend employee record page (FR-BB58) shows a "Values Profile" section visible only to users with role ≥ `department_admin`; the section contains a "Generate" button; clicking it calls the endpoint and displays the narrative text with a clearly visible "AI-generated summary" disclaimer.
- [ ] AC-8: The "Generate" button is only visible for sessions associated with loyalty-track exams; for non-loyalty sessions the "Values Profile" section is entirely hidden.
- [ ] AC-9: The employee record endpoint (FR-BB53) includes `exam_category_track: string | null` in each session object in its response; when the exam's category has `track = 'loyalty'`, this field equals `"loyalty"`; for all other exams it is `null`.

## Technical Specification

### Configuration / Infrastructure
No new tables or migrations required beyond `ai_usage_log` (created in FR-BB71).

`categories.track TEXT` already exists — it was added in migration 008 (`008_categories_tags.up.sql`) and is seeded with values such as `'loyalty'`, `'security'`, and `'safety'`. No migration is required for this column.

### API Endpoints

#### GET /api/v1/admin/ai/loyalty-summary/:sessionId
- **Auth**: JWT required; role ≥ `department_admin`
- **No caching** — always calls Anthropic API
- **Request**: no body; session ID in URL path
- **Validation steps**:
  1. Load `exam_sessions` where `id = :sessionId`.
  2. Load session's `exam_id` → `exams.category_id` → `categories.track`.
  3. If `track != 'loyalty'` → `400 NOT_A_LOYALTY_SESSION`.
  4. Check that the session's employee is in a department managed by the requesting admin (or requester is `super_admin`); else `403`.
- **Success response** `200`:
  ```json
  {
    "data": {
      "narrative": "The responses indicate a strong alignment with collaborative values and high emphasis on organizational loyalty. The profile suggests a preference for structured environments with clear hierarchies and long-term commitment over short-term flexibility. Responses to autonomy-related questions were moderate, suggesting comfort with guided independence.",
      "generated_at": "2026-05-14T11:00:00Z"
    },
    "error": null
  }
  ```
- **Error responses**:
  - `400` — `NOT_A_LOYALTY_SESSION`
  - `403` — insufficient role or department access
  - `404` — session not found
  - `503` — `AI_UNAVAILABLE`

### Implementation Details

**Data collection** (`internal/ai/loyalty.go`):

Before constructing the prompt, collect the session's Likert answers with their option metadata (normalized weight, question category dimension — but NOT the employee's identity). Polarity inversion is applied here so the prompt receives only normalized weights:
```go
type LikertResponseData struct {
    DimensionLabel  string // e.g. "Loyalty & Values" — sourced from c.name (categories)
    NormalizedWeight int   // 1-5 weight after polarity inversion applied
}

func (r *postgresRepository) CollectLikertResponses(ctx context.Context, sessionID uuid.UUID) ([]LikertResponseData, error) {
    // JOIN session_answers sa
    //   → answer_options ao ON ao.id = (sa.selected_option_ids->>0)::uuid
    //   → questions q        ON q.id  = ao.question_id
    //   → categories c       ON c.id  = q.category_id
    //   JOIN session_questions sq ON sq.question_id = q.id AND sq.session_id = $1
    // SELECT c.name AS dimension_label,
    //        ROUND(ao.likert_weight)::int AS raw_weight,
    //        ao.likert_polarity
    // WHERE sa.session_id = $1 AND q.type = 'likert'
    // ORDER BY sq.sort_order
    //
    // After scanning each row into rawWeight (int) and polarity (string),
    // apply polarity inversion in Go before populating NormalizedWeight
    // (see score normalization below).
}
```

`DimensionLabel` uses `c.name` (the category name from the `categories` table, e.g. "Loyalty & Values") joined via `questions.category_id`. This avoids sending question text to Anthropic. Note: `answer_options.likert_weight` is `DECIMAL(5,2)` — cast to `int` using `ROUND()` in SQL before scanning.

**Score normalization for negative-polarity items** (applied in the repository/service layer before building the prompt):
```go
// After scanning raw_weight (DECIMAL cast to int) and likert_polarity:
normalizedWeight := rawWeight
if polarity == "negative" {
    normalizedWeight = 6 - rawWeight // invert: 1→5, 2→4, 3→3, 4→2, 5→1
}
resp.NormalizedWeight = normalizedWeight
responses = append(responses, resp)
```

Polarity inversion is applied before `LikertResponseData` is populated. The prompt receives only `NormalizedWeight`; raw polarity values are never included in the prompt text.

**Prompt wrapper and construction** (`internal/ai/prompt.go`):
```go
// LoyaltyPromptData is declared in internal/ai/model.go — do not redeclare here.

// BuildLoyaltyPrompt renders the loyalty narrative prompt using text/template.
func BuildLoyaltyPrompt(data LoyaltyPromptData) string { /* ... */ }

const loyaltyNarrativePromptTemplate = `
You are an organizational psychologist interpreting Likert-scale survey responses from a corporate values assessment.

Below are the anonymized response patterns from a single assessment. Each line shows:
  Dimension | Normalized Weight (1=Low alignment, 5=High alignment — polarity already normalized)

{{range .Responses}}
- {{.DimensionLabel}} | NormalizedWeight: {{.NormalizedWeight}}
{{end}}

Write a 2-3 sentence narrative (third person, professional tone) describing the values profile these
responses suggest. Do NOT mention any individual by name or pronoun. Use language like "the responses indicate"
or "this profile suggests". Do not diagnose, make medical claims, or render employment judgments.
Focus only on workplace values and organizational alignment patterns.
`
```

**Shared type declarations** (`internal/ai/model.go`):

The following types must be declared in `internal/ai/model.go` (not scattered across `service.go`, `repository.go`, or `loyalty.go`) so all layers share a single source of truth without circular imports:

```go
// LikertResponseData holds one polarity-inverted Likert answer for prompt construction.
// DimensionLabel comes from categories.name; no question text or PII is included.
type LikertResponseData struct {
    DimensionLabel  string // e.g. "Loyalty & Values" — from categories.name
    NormalizedWeight int   // 1–5 after polarity inversion
}

// LoyaltyPromptData is passed to BuildLoyaltyPrompt in internal/ai/prompt.go.
type LoyaltyPromptData struct {
    Responses []LikertResponseData
}

// LoyaltyNarrativeResult is returned by GetLoyaltyNarrative in internal/ai/service.go.
type LoyaltyNarrativeResult struct {
    Narrative   string
    GeneratedAt time.Time
}

// ErrLoyaltySessionNotFound is returned by GetSessionCategoryTrack when no
// session row exists for the given ID. The handler maps this to HTTP 404.
var ErrLoyaltySessionNotFound = errors.New("session not found")
```

**Service interface extension** (`internal/ai/service.go`):
```go
// Add to Service interface:
GetLoyaltyNarrative(ctx context.Context, sessionID uuid.UUID, adminUserID uuid.UUID) (*LoyaltyNarrativeResult, error)
```

**Repository interface extension** (`internal/ai/repository.go`):
```go
// Add to Repository interface:

// GetSessionCategoryTrack loads the exam's category track and the session
// employee's user ID for the given session.
GetSessionCategoryTrack(ctx context.Context, sessionID uuid.UUID) (track string, employeeUserID uuid.UUID, err error)

// IsEmployeeInAdminDepartment returns true if employeeUserID belongs to any
// department managed by adminUserID.
IsEmployeeInAdminDepartment(ctx context.Context, adminUserID, employeeUserID uuid.UUID) (bool, error)

// CollectLikertResponses returns anonymised, polarity-inverted Likert
// response data for the given session.
CollectLikertResponses(ctx context.Context, sessionID uuid.UUID) ([]LikertResponseData, error)
```

**Rate limiting**: The `GET /api/v1/admin/ai/loyalty-summary/:sessionId` endpoint applies the same per-user rate limit as other AI endpoints (20 calls/hour per admin user), enforced via the existing `ratelimit` middleware registered in the router for the `/api/v1/admin/ai/` prefix.

**AI client call**: uses the same `internal/ai.Client` as FR-BB71 and FR-BB73. Advisory `max_tokens = 512` (sufficient for 3 sentences); note that the existing `AnthropicClient.GenerateText` method does not expose a `maxTokens` parameter — the model's default token limit applies. If token budgeting is needed in future, extend the client interface in a follow-up.

**Response parsing**: the model returns plain text (not JSON) for this feature; no parsing required. Trim whitespace. If response is empty, return `503 AI_UNAVAILABLE`.

**Usage logging**: log with the requesting admin's `user_id`, NOT the employee's user ID. The session ID is not logged in `ai_usage_log`.

### FR-BB53 Employee Record API Extension

The `GET /api/v1/admin/users/{id}/record` endpoint (FR-BB53) must be extended so each session object includes `exam_category_track`.

**Model change** (`internal/reports/model.go`): Add the following field to the `SessionRecord` struct (after `CertificateID`):
```go
ExamCategoryTrack *string `json:"exam_category_track" db:"exam_category_track"`
```

**Repository change** (`internal/reports/repository.go`): `GetUserSessionHistory` must be updated to join `categories` and project the track column. Add `LEFT JOIN categories cat ON cat.id = e.category_id` to the `FROM` clause and `cat.track AS exam_category_track` to the `SELECT` list. The function signature and return type (`[]SessionRecord`) do not change — only the query body changes.

Updated query:
```sql
SELECT
  es.id AS session_id,
  es.exam_id,
  e.title AS exam_title,
  es.started_at,
  es.submitted_at,
  es.score_pct,
  es.passed,
  EXTRACT(EPOCH FROM (es.submitted_at - es.started_at))::INT AS time_taken_seconds,
  es.status,
  c.id AS certificate_id,
  cat.track AS exam_category_track          -- NEW
FROM exam_sessions es
JOIN exams e ON e.id = es.exam_id
LEFT JOIN certificates c ON c.session_id = es.id
LEFT JOIN categories cat ON cat.id = e.category_id  -- NEW
WHERE es.user_id = $1
  AND es.status != 'in_progress'
ORDER BY es.started_at DESC
LIMIT $2 OFFSET $3
```

### Frontend API

```typescript
// src/api/ai.ts
// apiGet<T> is defined locally in this file (not in a shared client module).
export async function fetchLoyaltyNarrative(sessionId: string): Promise<{ narrative: string; generated_at: string }> {
  return apiGet<{ narrative: string; generated_at: string }>(`/api/v1/admin/ai/loyalty-summary/${sessionId}`);
}
```

### Frontend Components

**Employee Record Page** (`src/pages/admin/EmployeeRecord.tsx`) — Values Profile section:

```tsx
function ValuesProfileSection({ sessionId, isLoyaltySession }: {
  sessionId: string;
  isLoyaltySession: boolean;
}) {
  const { t } = useTranslation();
  const [narrative, setNarrative] = useState<string | null>(null);
  const [generatedAt, setGeneratedAt] = useState<string | null>(null);

  const { mutate, isPending, isError } = useMutation({
    mutationFn: () => fetchLoyaltyNarrative(sessionId),
    onSuccess: (data) => {
      setNarrative(data.narrative);
      setGeneratedAt(data.generated_at);
    },
  });

  if (!isLoyaltySession) return null;

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t('employee_record.valuesProfileTitle')}</CardTitle>
      </CardHeader>
      <CardContent className="space-y-3">
        {narrative ? (
          <>
            <p className="text-sm">{narrative}</p>
            <p className="text-xs text-muted-foreground">
              {t('common.aiGenerated')} · {formatDate(generatedAt!, i18n.language)}
            </p>
            <Button variant="outline" size="sm" onClick={() => mutate()} disabled={isPending}>
              {t('employee_record.regenerateNarrative')}
            </Button>
          </>
        ) : (
          <>
            {isError && (
              <p className="text-destructive text-sm">{t('errors.aiUnavailable')}</p>
            )}
            <Button onClick={() => mutate()} disabled={isPending}>
              {isPending ? t('employee_record.generating') : t('employee_record.generateNarrative')}
            </Button>
          </>
        )}
      </CardContent>
    </Card>
  );
}
```

#### i18n Keys

All new keys must be added to all three locale files (`src/locales/kk.json`, `src/locales/ru.json`, `src/locales/en.json`) under the `employee_record` namespace:

| Key | en | ru | kk |
|-----|----|----|----|
| `employee_record.valuesProfileTitle` | "Values Profile" | "Ценностный профиль" | "Құндылықтар профилі" |
| `employee_record.generateNarrative` | "Generate" | "Создать" | "Жасау" |
| `employee_record.regenerateNarrative` | "Regenerate" | "Обновить" | "Жаңарту" |
| `employee_record.generating` | "Generating…" | "Создаётся…" | "Жасалуда…" |

`common.aiGenerated` already exists with value `"AI-generated"` — no addition to the `common` namespace is required.

> **Auth note (L2)**: `fetchLoyaltyNarrative` follows the same no-token pattern as `fetchAIInsights` — it calls `apiGet<T>(url)` with no explicit token argument. Authentication is handled by `credentials: 'include'` (HTTP-only session cookie) baked into the `apiGet` helper. No token is extracted from the React Query cache for this call.

**Placement**: the `ValuesProfileSection` is rendered below the exam result summary card on the employee record page. The section is conditionally rendered based on:
1. The authenticated user has role ≥ `department_admin`.
2. The selected session is from a loyalty-track exam (determined from the session's exam category `track` field, returned by the employee record API).

**Detection of loyalty sessions**: the employee record API response (FR-BB53) should include `session.exam_category_track` so the frontend does not need a separate request to determine this.

## Out of Scope
- Caching loyalty narrative results between requests.
- Storing narratives in the database for later retrieval (requires a separate retention-policy-compliant table).
- Generating narratives for non-Likert question types.
- Bulk generation for multiple employees at once.

## Test Strategy
- **Unit**: `TestGetLoyaltyNarrative_NotLoyaltySession` (400), `TestGetLoyaltyNarrative_ForbiddenDepartment` (403), `TestGetLoyaltyNarrative_SessionNotFound` (404), `TestGetLoyaltyNarrative_AIUnavailable` (503), `TestGetLoyaltyNarrative_Success` (200 with mocked Anthropic client).
- **Unit**: `TestCollectLikertResponses_NormalizedWeights` — verifies negative-polarity items are inverted before population of `NormalizedWeight`.
- **Unit**: `TestBuildLoyaltyPrompt` — verifies template renders correctly; no PII fields appear in output.
- **Integration**: Employee record page shows Values Profile section for loyalty-track sessions, hidden for non-loyalty sessions.
- **Integration**: FR-BB53 employee record API returns `exam_category_track: "loyalty"` for loyalty exam sessions and `null` for others.

## Notes
- This feature intentionally does not cache results. Loyalty profile narratives are sensitive HR data; caching creates risk of stale narratives being presented for an employee who retook the exam or whose responses were corrected. Each on-demand generation reflects current session data.
- The "AI-generated summary" disclaimer label is required on every display of the narrative. It must never be removed or hidden, even if the narrative is stored for display on a later page load.
- `DimensionLabel` is sourced from `c.name` in the `categories` table (e.g. "Loyalty & Values"), not from any question text. This ensures no question wording is ever included in the Anthropic prompt. Implementors must not substitute question stems or translations for the dimension label.
- Department admin access control: the service layer must verify that `session.user_id` is an employee in one of the departments managed by the requesting admin, not just that the admin has `department_admin` role globally.
- FR-BB311 (Grading Engine) established the `session_question_scores` and `session_answers` tables that `CollectLikertResponses` queries; this feature depends on those tables being present.
- Future consideration: if loyalty narrative results are to be stored for HR record-keeping, a separate `loyalty_narratives` table with GDPR-compliant retention policies would be required. This is out of scope for Phase 7.

# FR-BB75 — Loyalty Profile Narrative

## Metadata
| Field | Value |
|-------|-------|
| ID | FR-BB75 |
| Phase | 7 — AI Layer |
| Priority | 2 |
| Status | Draft |
| Depends On | FR-BB311, FR-BB53, FR-BB58, FR-BB71 |

## Description
Provides department admins with an AI-generated 2–3 sentence narrative describing an employee's values profile derived from their Likert-scale responses on a loyalty-track exam session. The narrative is generated on demand (not automatically), uses only anonymized response weights (no PII in the prompt), and is surfaced in the employee record page. Results are not cached, ensuring the narrative always reflects the most recent session data.

## Acceptance Criteria
- [ ] AC-1: `GET /api/v1/admin/ai/loyalty-summary/:sessionId` validates that the session belongs to an exam in a category with `track = 'loyalty'`; sessions from non-loyalty exams return `400` with error code `NOT_A_LOYALTY_SESSION`.
- [ ] AC-2: The endpoint requires JWT auth with role ≥ `department_admin`; requests from examiners with no department match or from employees return `403 Forbidden`.
- [ ] AC-3: The prompt sent to the Anthropic API contains only anonymized Likert option weights and polarity labels — no employee name, user ID, email, or any other personally identifying information is included in the prompt text.
- [ ] AC-4: The endpoint returns a narrative of 2–3 complete sentences describing the employee's apparent values and workplace orientation based on the Likert response pattern; the narrative does not contain any identifying language (no "this employee" — use "this profile" or "the responses indicate").
- [ ] AC-5: Every call to this endpoint logs to `ai_usage_log` with `feature = "loyalty_narrative"`, the calling admin's `user_id`, and `tokens_used` from the API response; the session employee's user ID is NOT logged in `ai_usage_log` to avoid tying AI usage records to individual employees.
- [ ] AC-6: When the Anthropic API is unavailable, the endpoint returns `503` with error code `AI_UNAVAILABLE`; no partial or cached response is returned.
- [ ] AC-7: The frontend employee record page (FR-BB58) shows a "Values Profile" section visible only to users with role ≥ `department_admin`; the section contains a "Generate" button; clicking it calls the endpoint and displays the narrative text with a clearly visible "AI-generated summary" disclaimer.
- [ ] AC-8: The "Generate" button is only visible for sessions associated with loyalty-track exams; for non-loyalty sessions the "Values Profile" section is entirely hidden.

## Technical Specification

### Configuration / Infrastructure
No new tables or migrations required beyond `ai_usage_log` (created in FR-BB71).

The `categories` table must have a `track TEXT` column (e.g. `'loyalty'`, `'knowledge'`, `null`). If not already present, add via migration:
```sql
ALTER TABLE categories
  ADD COLUMN IF NOT EXISTS track TEXT;
```

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

Before constructing the prompt, collect the session's Likert answers with their option metadata (weight, polarity, question dimension label — but NOT the employee's identity):
```go
type LikertResponseData struct {
    DimensionLabel string  // e.g. "Loyalty to organization"
    OptionWeight   int     // 1-5 numeric weight
    Polarity       string  // "positive" | "negative" (reversal for negative items)
}

func (s *AIService) CollectLikertResponses(ctx context.Context, sessionID uuid.UUID) ([]LikertResponseData, error) {
    // JOIN session_answers → questions → question_options
    // SELECT q.stem_label (or category dimension), o.weight, o.polarity
    // WHERE session_id = $1 AND q.type = 'likert'
    // ORDER BY q.order_num
}
```

`q.stem_label` is a short dimension tag (max 50 chars), not the full question text, to minimize PII risk from question wording.

**Prompt construction** (`internal/ai/prompt.go`):
```go
const loyaltyNarrativePromptTemplate = `
You are an organizational psychologist interpreting Likert-scale survey responses from a corporate values assessment.

Below are the anonymized response patterns from a single assessment. Each line shows:
  Dimension | Weight (1=Strongly Disagree, 5=Strongly Agree) | Polarity (positive/negative item)

{{range .Responses}}
- {{.DimensionLabel}} | Weight: {{.OptionWeight}} | Polarity: {{.Polarity}}
{{end}}

Write a 2-3 sentence narrative (third person, professional tone) describing the values profile these
responses suggest. Do NOT mention any individual by name or pronoun. Use language like "the responses indicate"
or "this profile suggests". Do not diagnose, make medical claims, or render employment judgments.
Focus only on workplace values and organizational alignment patterns.
`
```

**Score normalization for negative-polarity items**: before adding to the prompt, reverse negative-polarity weights:
```go
effectiveWeight := resp.OptionWeight
if resp.Polarity == "negative" {
    effectiveWeight = 6 - resp.OptionWeight // invert: 1→5, 2→4, 3→3, 4→2, 5→1
}
```

**AI client call**: uses the same `internal/ai.Client` as FR-BB71 and FR-BB73; `max_tokens = 512` (sufficient for 3 sentences).

**Response parsing**: the model returns plain text (not JSON) for this feature; no parsing required. Trim whitespace. If response is empty, return `503 AI_UNAVAILABLE`.

**Usage logging**: log with the requesting admin's `user_id`, NOT the employee's user ID. The session ID is not logged in `ai_usage_log`.

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
        <CardTitle>{t('employee.valuesProfileTitle')}</CardTitle>
      </CardHeader>
      <CardContent className="space-y-3">
        {narrative ? (
          <>
            <p className="text-sm">{narrative}</p>
            <p className="text-xs text-muted-foreground">
              {t('common.aiGeneratedSummary')} · {formatDate(generatedAt!, i18n.language)}
            </p>
            <Button variant="outline" size="sm" onClick={() => mutate()} disabled={isPending}>
              {t('employee.regenerateNarrative')}
            </Button>
          </>
        ) : (
          <>
            {isError && (
              <p className="text-destructive text-sm">{t('errors.aiUnavailable')}</p>
            )}
            <Button onClick={() => mutate()} disabled={isPending}>
              {isPending ? t('common.generating') : t('employee.generateNarrative')}
            </Button>
          </>
        )}
      </CardContent>
    </Card>
  );
}
```

**Placement**: the `ValuesProfileSection` is rendered below the exam result summary card on the employee record page. The section is conditionally rendered based on:
1. The authenticated user has role ≥ `department_admin`.
2. The selected session is from a loyalty-track exam (determined from the session's exam category `track` field, returned by the employee record API).

**Detection of loyalty sessions**: the employee record API response (FR-BB53) should include `session.exam_category_track` so the frontend does not need a separate request to determine this.

## Notes
- This feature intentionally does not cache results. Loyalty profile narratives are sensitive HR data; caching creates risk of stale narratives being presented for an employee who retook the exam or whose responses were corrected. Each on-demand generation reflects current session data.
- The "AI-generated summary" disclaimer label is required on every display of the narrative. It must never be removed or hidden, even if the narrative is stored for display on a later page load.
- If the question bank uses full question stems as `DimensionLabel`, there is risk that sensitive question text is sent to Anthropic. Implementors must ensure `stem_label` is a short, de-identified dimension tag (e.g. "Organizational loyalty", "Peer collaboration") and not the full question text.
- Department admin access control: the service layer must verify that `session.user_id` is an employee in one of the departments managed by the requesting admin, not just that the admin has `department_admin` role globally.
- Future consideration: if loyalty narrative results are to be stored for HR record-keeping, a separate `loyalty_narratives` table with GDPR-compliant retention policies would be required. This is out of scope for Phase 7.

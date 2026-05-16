# FR-BB74 — Performance Insight Summaries

## Metadata
| Field | Value |
|-------|-------|
| ID | FR-BB74 |
| Phase | 7 — AI Layer |
| Priority | 2 |
| Status | implemented |
| Depends On | FR-BB52, FR-BB71 |

## Description
Allows examiners and admins to request AI-generated natural-language summaries of exam performance data for a specific exam. Anthropic Claude analyses anonymized aggregate statistics from the analytics API and returns 3–5 observation bullets. Results are cached for 24 hours to minimize API costs. A force-refresh option bypasses the cache. The frontend surfaces insights in a collapsible card on the per-exam analytics page.

## Scope

| Layer | Items |
|-------|-------|
| Database | New `ai_insight_cache` table (migration `028_ai_insight_cache`) |
| Backend | `internal/ai/` — handler, service (`insights.go`), repository (`repo.go`), prompt builder (`prompt.go`); router wiring in `internal/router/` |
| Frontend | `src/pages/admin/ExamAnalytics.tsx` — `AIInsightsCard` component; `src/api/ai.ts` — `fetchAIInsights` wrapper |
| i18n | `analytics.aiInsightsTitle`, `common.aiGenerated`, `analytics.generatedAt`, `analytics.cached`, `analytics.regenerate`, `common.toggleExpand`, `errors.aiUnavailable` |

## Acceptance Criteria
- [x] AC-1: `GET /api/v1/admin/ai/insights/:examId` returns between 3 and 5 natural-language insight strings derived from the exam's aggregate analytics data; each insight is a complete, actionable observation (e.g. identifying low-performing questions, time-to-complete patterns, pass/fail distribution anomalies).
- [x] AC-2: The endpoint requires JWT auth with role ≥ `examiner`; requests from employees return `403 Forbidden`; requests for exams the requester cannot access (wrong tenant) return `404`.
- [x] AC-3: Insight results are cached in the `ai_insight_cache` table for 24 hours; a second request within the cache window returns the cached result with `"cached": true` without calling the Anthropic API.
- [x] AC-4: Passing `?refresh=true` to the endpoint bypasses the cache, calls the Anthropic API, updates the cache entry, and returns `"cached": false`.
- [x] AC-5: The data sent to the Anthropic API contains only aggregate statistics (pass rate, avg score, per-question correct rates, avg completion time) — no employee names, user IDs, or other personally identifying information are included in the prompt.
- [x] AC-6: When the Anthropic API is unavailable, the endpoint returns `503` with error code `AI_UNAVAILABLE`; it does not fall back to returning stale cache beyond the 24-hour window.
- [x] AC-7: The frontend per-exam analytics page renders an "AI Insights" collapsible card showing the insight bullet list, a `"AI-generated"` label in the card header, a timestamp of when insights were generated, and a "Regenerate" button that calls the endpoint with `?refresh=true`.
- [x] AC-8: Every Anthropic API call for this feature is recorded in `ai_usage_log` with `feature = "exam_insights"` and the calling user's ID.

## Technical Specification

### Configuration / Infrastructure

**New migration** — `migrations/028_ai_insight_cache.up.sql`:
```sql
CREATE TABLE ai_insight_cache (
  exam_id       UUID        PRIMARY KEY REFERENCES exams(id) ON DELETE CASCADE,
  insights      JSONB       NOT NULL,   -- array of insight strings
  generated_at  TIMESTAMPTZ NOT NULL,
  generated_by  UUID        REFERENCES users(id) ON DELETE SET NULL
);
```

`insights` column stores a JSON array of strings, e.g. `["Insight 1", "Insight 2", ...]`.

**Down migration** — `migrations/028_ai_insight_cache.down.sql`:
```sql
DROP TABLE IF EXISTS ai_insight_cache;
```

### API Endpoints

#### GET /api/v1/admin/ai/insights/:examId
- **Auth**: JWT required; role ≥ `examiner`; exam must belong to the requester's tenant
- **Query params**: `refresh=true` (optional; bypasses cache)
- **Cache logic**:
  1. Look up `ai_insight_cache` where `exam_id = :examId`.
  2. If found AND `generated_at > NOW() - INTERVAL '24 hours'` AND `?refresh` is absent → return cached result.
  3. Otherwise → call Anthropic API → upsert cache → return fresh result.
- **Success response** `200`:
  ```json
  {
    "data": {
      "insights": [
        "Question 7 has a 23% correct rate, suggesting the stem may be ambiguous or the topic needs additional training.",
        "The average completion time of 42 minutes is well within the 60-minute limit, indicating the exam duration is appropriate.",
        "The pass rate of 68% is below the 75% target; consider reviewing content coverage for Category B topics."
      ],
      "generated_at": "2026-05-14T10:30:00Z",
      "cached": true
    },
    "error": null
  }
  ```
- **Error responses**:
  - `403` — `{ "data": null, "error": { "code": "FORBIDDEN", "message": "insufficient role" } }`
  - `404` — `{ "data": null, "error": { "code": "EXAM_NOT_FOUND", "message": "exam not found or access denied" } }`
  - `503` — `{ "data": null, "error": { "code": "AI_UNAVAILABLE", "message": "AI service unavailable" } }`

### Implementation Details

**Analytics data gathered before calling AI** (`internal/ai/insights.go`):

Calls internal service methods (not the HTTP endpoint) to collect:
```go
type ExamInsightData struct {
    ExamTitle         string
    TotalAttempts     int
    PassRate          float64  // 0.0-1.0
    AvgScorePct       float64
    AvgCompletionSecs int
    PassingScorePct   int
    QuestionStats     []QuestionStat
}

type QuestionStat struct {
    OrderNum       int
    Stem           string    // truncated to 100 chars for prompt brevity
    CorrectRate    float64   // 0.0-1.0
    AvgTimeSecs    int
}
```

**Anthropic prompt** (`internal/ai/prompt.go`):
```go
const insightPromptTemplate = `
You are an expert HR and learning analytics consultant reviewing exam performance data.
Analyse the following aggregate statistics for the exam "{{.ExamTitle}}" and generate
3 to 5 concise, specific, actionable insights for the exam administrator.

IMPORTANT: Do not mention any individual employees. Only discuss aggregate patterns.

Exam Statistics:
- Total attempts: {{.TotalAttempts}}
- Pass rate: {{printf "%.1f" (mul .PassRate 100)}}%
- Passing score threshold: {{.PassingScorePct}}%
- Average score: {{printf "%.1f" (mul .AvgScorePct 100)}}%
- Average completion time: {{.AvgCompletionSecs}} seconds

Per-Question Performance (Question # — Correct Rate — Stem excerpt):
{{range .QuestionStats}}
  Q{{.OrderNum}} ({{printf "%.0f" (mul .CorrectRate 100)}}% correct, avg {{.AvgTimeSecs}}s): {{.Stem}}
{{end}}

Return ONLY a JSON array of insight strings, no markdown:
["insight 1", "insight 2", ...]
`
```

**Cache upsert** (`internal/ai/repo.go`):
```go
const upsertInsightCache = `
INSERT INTO ai_insight_cache (exam_id, insights, generated_at, generated_by)
VALUES (:exam_id, :insights, NOW(), :user_id)
ON CONFLICT (exam_id)
DO UPDATE SET insights = EXCLUDED.insights,
              generated_at = EXCLUDED.generated_at,
              generated_by = EXCLUDED.generated_by
`
```

**Response parsing**: unmarshal the model response as a JSON array of strings. If the array has fewer than 3 or more than 5 elements, log a warning and return what was generated (do not reject; partial insights are better than none).

**AI usage logging**: called after successful Anthropic response, before cache write.

### Frontend Components

**AI Insights Card** (`src/pages/admin/ExamAnalytics.tsx`):
```tsx
function AIInsightsCard({ examId }: { examId: string }) {
  const { t } = useTranslation();
  const [refresh, setRefresh] = useState(false);

  const { data, isLoading, isError, refetch } = useQuery({
    queryKey: ['aiInsights', examId, refresh],
    queryFn: () => fetchAIInsights(examId, refresh),
    staleTime: 0, // always use server response; cache is managed server-side
  });

  const handleRegenerate = () => {
    setRefresh(true);
    refetch();
  };

  return (
    <Collapsible defaultOpen={false}>
      <Card>
        <CardHeader className="flex flex-row items-center justify-between">
          <div className="flex items-center gap-2">
            <CardTitle>{t('analytics.aiInsightsTitle')}</CardTitle>
            <Badge variant="outline" className="text-xs">{t('common.aiGenerated')}</Badge>
          </div>
          <div className="flex items-center gap-2">
            {data && (
              <span className="text-xs text-muted-foreground">
                {t('analytics.generatedAt', { date: formatDate(data.generated_at, i18n.language) })}
                {data.cached && ` · ${t('analytics.cached')}`}
              </span>
            )}
            <Button variant="outline" size="sm" onClick={handleRegenerate} disabled={isLoading}>
              {t('analytics.regenerate')}
            </Button>
            <CollapsibleTrigger asChild>
              <Button variant="ghost" size="icon" aria-label={t('common.toggleExpand')}>
                <ChevronDown className="h-4 w-4" aria-hidden="true" />
              </Button>
            </CollapsibleTrigger>
          </div>
        </CardHeader>
        <CollapsibleContent>
          <CardContent>
            {isLoading && <Skeleton className="h-20 w-full" />}
            {isError  && <p className="text-destructive text-sm">{t('errors.aiUnavailable')}</p>}
            {data && (
              <ul className="list-disc list-inside space-y-1">
                {data.insights.map((insight, i) => (
                  <li key={i} className="text-sm">{insight}</li>
                ))}
              </ul>
            )}
          </CardContent>
        </CollapsibleContent>
      </Card>
    </Collapsible>
  );
}
```

The `AIInsightsCard` is rendered below the existing charts on the per-exam analytics page, visible only when the authenticated user has `examiner` or higher role.

## Out of Scope
- Streaming insight responses
- Per-department aggregate insights
- Scheduled or webhook-triggered auto-generation
- Insight generation for question banks outside an exam context

## Test Strategy
- **Unit** — Prompt builder tested with a fixed `ExamInsightData` fixture; verify the rendered prompt contains expected statistics and no PII.
- **Unit** — Cache hit/miss logic tested with a mock repository; verify that requests within 24 hours return cached data and that `?refresh=true` bypasses the cache.
- **Unit** — Anthropic response parser tested with arrays of 2, 3, 5, and 6 strings; verify boundary logging.
- **Integration** — `GET /api/v1/admin/ai/insights/:examId` tested with a real DB and mocked Anthropic client; verify 200 (cache miss), 200 (cache hit, `cached: true`), 403 (employee role), 404 (wrong tenant), 503 (Anthropic error).
- **Frontend** — `AIInsightsCard` component tested for loading skeleton, error state, populated insight list, and "Regenerate" button click triggering a refetch with `?refresh=true`.

## Notes
- The 24-hour cache window balances freshness with API cost; if the exam has no new attempts since the last generation, regenerating produces identical insights. The "Regenerate" UI is provided but discouraged for frequent use — consider adding a tooltip noting insights are cached for 24 hours.
- Question stem truncation to 100 characters in the prompt reduces token usage significantly for exams with long question text; the truncation should be done with `utf8.RuneCountInString` to avoid cutting multibyte characters.
- If the `ai_insight_cache` table is seeded with stale data (e.g. after a migration rollback), an admin can force refresh via `?refresh=true`.
- Future enhancement: webhook or scheduled job to auto-generate insights for exams with >50 new attempts since last generation.

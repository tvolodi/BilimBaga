# FR-BB71 — Question Generation Assist

## Metadata
| Field | Value |
|-------|-------|
| ID | FR-BB71 |
| Phase | 7 — AI Layer |
| Priority | 1 |
| Status | Draft |
| Depends On | FR-BB23 |

## Description
Enables HR admins and examiners to generate draft exam questions using Anthropic Claude. The requester provides a category, difficulty, count, and optional context text; the API returns a structured JSON preview of candidate questions. The user reviews and confirms the draft questions before they are committed via the existing question creation endpoint. All AI usage is recorded in an `ai_usage_log` table for cost tracking and auditing.

## Acceptance Criteria
- [ ] AC-1: `POST /api/v1/admin/ai/generate-questions` accepts a valid request body and returns up to 10 draft questions in the specified format; each draft includes `type`, `difficulty`, `stem`, `explanation`, `options` (with `text` and `is_correct`), and `tags`.
- [ ] AC-2: The endpoint is protected by JWT auth and requires at minimum the `examiner` role; requests from employees return `403 Forbidden`.
- [ ] AC-3: When the Anthropic API is unreachable or returns a non-200 response, the endpoint returns HTTP 503 with error code `AI_UNAVAILABLE`; the error never bubbles through as a 500.
- [ ] AC-4: Every successful AI call is recorded in the `ai_usage_log` table with `user_id`, `feature = "question_generation"`, `tokens_used` from the API response usage object, and `model` name.
- [ ] AC-5: The endpoint enforces a rate limit of 20 AI-generate requests per hour per authenticated user; requests beyond this limit return `429` with error code `AI_RATE_LIMITED`.
- [ ] AC-6: `count` is validated to be between 1 and 10 inclusive; `context_text` is validated to be at most 2000 characters; `difficulty` must be one of `"easy"`, `"medium"`, `"hard"`; invalid inputs return `400 Bad Request`.
- [ ] AC-7: The frontend question editor displays an "AI Generate" button (examiner+ only) that opens a generation dialog; after the API returns, draft questions are shown in a preview list; the user can individually select and confirm each question, which triggers `POST /api/v1/questions` for each confirmed draft.
- [ ] AC-8: Draft questions returned by the generate endpoint are never automatically inserted into the database; they exist only in the API response and the frontend's local state until the user explicitly confirms them.

## Technical Specification

### Configuration / Infrastructure

**New environment variable** (add to `.env.example`):
```
ANTHROPIC_API_KEY=sk-ant-...
ANTHROPIC_MODEL=claude-3-5-haiku-20241022
```

**New dependency** (`backend/go.mod`):
```
github.com/anthropics/anthropic-sdk-go
```

**New migration** — `migrations/NNN_ai_usage_log.sql`:
```sql
CREATE TABLE ai_usage_log (
  id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id     UUID        NOT NULL REFERENCES users(id) ON DELETE SET NULL,
  feature     TEXT        NOT NULL,
  tokens_used INT         NOT NULL,
  model       TEXT        NOT NULL,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_ai_usage_log_user_created
  ON ai_usage_log (user_id, created_at DESC);
```

### API Endpoints

#### POST /api/v1/admin/ai/generate-questions
- **Auth**: JWT required; role ≥ `examiner`
- **Rate limit**: 20 requests/hour per `user_id` (tracked against `ai_usage_log` count in last 60 min)
- **Request body**:
  ```json
  {
    "category_id":   "uuid",
    "difficulty":    "easy | medium | hard",
    "count":         5,
    "context_text":  "Optional background text (max 2000 chars)"
  }
  ```
- **Success response** `200`:
  ```json
  {
    "data": {
      "questions": [
        {
          "type":        "single_choice",
          "difficulty":  "medium",
          "stem":        "Which of the following best describes...",
          "explanation": "The correct answer is... because...",
          "options": [
            { "text": "Option A", "is_correct": false },
            { "text": "Option B", "is_correct": true },
            { "text": "Option C", "is_correct": false },
            { "text": "Option D", "is_correct": false }
          ],
          "tags": ["topic1", "subtopic2"]
        }
      ]
    },
    "error": null
  }
  ```
- **Error responses**:
  - `400` — validation failure
  - `403` — insufficient role
  - `429` — AI rate limit exceeded (`AI_RATE_LIMITED`)
  - `503` — Anthropic API unavailable (`AI_UNAVAILABLE`)

### Implementation Details

**Package**: `internal/ai/`
- `client.go` — wraps `anthropic-sdk-go`, implements `GenerateQuestions` and `GradeShortText`
- `prompt.go` — prompt templates
- `log.go` — `logUsage(ctx, userID, feature, tokensUsed, model)` repository function

**Anthropic prompt** (`prompt.go`):
```go
const generateQuestionsPromptTemplate = `
You are an expert exam question writer for corporate training platforms.
Generate {{.Count}} exam questions for the category "{{.CategoryName}}" at {{.Difficulty}} difficulty level.
{{if .ContextText}}Use the following context to inform the questions:\n{{.ContextText}}{{end}}

Return ONLY valid JSON with no markdown, matching exactly this schema:
{
  "questions": [
    {
      "type": "single_choice | multiple_choice | true_false",
      "difficulty": "{{.Difficulty}}",
      "stem": "string",
      "explanation": "string (why the correct answer is correct)",
      "options": [{"text": "string", "is_correct": bool}],
      "tags": ["string"]
    }
  ]
}

Requirements:
- Each single_choice question has exactly 4 options with exactly 1 correct
- Each multiple_choice question has 4-6 options with 2-3 correct
- Each true_false question has exactly 2 options (True, False)
- Explanations are 1-3 sentences
- Questions are clear, unambiguous, and professionally worded
`
```

**Response parsing**:
- Use `encoding/json` to unmarshal the model's text content into a typed struct `GeneratedQuestionsResponse`.
- If JSON unmarshal fails (model hallucination / malformed output): return `503 AI_UNAVAILABLE` and log the raw response at warn level.

**Per-user AI rate limit check** (service layer, before calling Anthropic):
```go
func (s *AIService) checkRateLimit(ctx context.Context, userID uuid.UUID) error {
    count, err := s.repo.CountAIUsageLastHour(ctx, userID, "question_generation")
    if err != nil { return err }
    if count >= 20 { return ErrAIRateLimited }
    return nil
}
```

### Frontend Components

**AI Generate Dialog** (`src/components/questions/AIGenerateDialog.tsx`):
- Trigger: "AI Generate" `<Button>` in the question bank toolbar (visible only for `examiner+`).
- Form fields: Category select, Difficulty select, Count (1–10), Context textarea (max 2000).
- On submit: calls `useMutation` → `POST /api/v1/admin/ai/generate-questions`.
- Loading state: spinner with i18n label `question.aiGenerating`.
- Success: shows a `<Table>` of draft questions; each row has a checkbox + a preview expand.
- "Confirm Selected" button: iterates selected drafts, calls `POST /api/v1/questions` mutation for each, shows progress toast per item.
- Error states: `AI_UNAVAILABLE` → toast `errors.aiUnavailable`; `AI_RATE_LIMITED` → toast `errors.aiRateLimited`.

## Notes
- The Anthropic `claude-3-5-haiku` model is chosen for low latency and cost; the model name is configurable via `ANTHROPIC_MODEL` so it can be upgraded without code changes.
- JSON-mode prompting (instructing the model to return only JSON) is preferred over tool-use for this feature to keep the integration surface minimal.
- Draft questions must not bypass the existing question validation in `POST /api/v1/questions` (required fields, option constraints); the AI-generated drafts are treated as user input and validated normally.
- Token usage from `anthropic.Response.Usage.InputTokens + OutputTokens` should be summed and stored as `tokens_used`.

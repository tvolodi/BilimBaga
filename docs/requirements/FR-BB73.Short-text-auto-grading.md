# FR-BB73 — Short-Text Auto-Grading

## Metadata
| Field | Value |
|-------|-------|
| ID | FR-BB73 |
| Phase | 7 — AI Layer |
| Priority | 1 |
| Status | Revised |
| Depends On | FR-BB311, FR-BB71 |

## Affected Layers

| Layer | Items |
|-------|-------|
| Database | `questions` table (new `auto_grade`, `model_answer` columns), `session_question_scores` table (new `ai_reasoning` column), `ai_usage_log` table (read/write) |
| Backend | `internal/sessions/grading.go`, `internal/questions/` (repo + service + handler), migration `027_short_text_autograding` |
| Frontend | `QuestionEditor` (type=shorttext fields), `GradingQueue` (AI badge + reasoning) |

## Description
Extends the grading engine so that short-text questions with a configured model answer are automatically graded by Anthropic Claude. The AI returns a percentage score and reasoning, which are stored in the session results. Examiners can override the AI grade at any time via the existing manual grading endpoints. If the AI is unavailable, the question falls back gracefully to the manual grading queue.

## Acceptance Criteria
- [ ] AC-1: The `questions` table gains two new columns: `auto_grade BOOL NOT NULL DEFAULT false` and `model_answer TEXT`; these columns are only meaningful for `shorttext` question type.
- [ ] AC-2: When the grading engine processes a `shorttext` question where `auto_grade = true` AND `model_answer IS NOT NULL`, it calls the Anthropic Claude API to score the employee's text answer; the result is stored with `grading_status = 'ai_graded'`.
- [ ] AC-3: The Anthropic prompt instructs the model to return a JSON object `{ "score_pct": 0-100, "reasoning": "string" }` and nothing else; if the response cannot be parsed as valid JSON matching this schema, the question falls back to `grading_status = 'pending_manual'` and an error is logged.
- [ ] AC-4: If the Anthropic API returns an error or is unreachable, the question is placed in the manual grading queue (`grading_status = 'pending_manual'`) rather than failing the grading run; a warning is logged with the session ID and question ID.
- [ ] AC-5: Examiners can override an AI-graded score via the existing manual grading endpoints (FR-BB42); the override sets `grading_status = 'graded'` and replaces the stored score; the `reasoning` from the AI is preserved as a reference note.
- [ ] AC-6: Every AI grading call is logged to `ai_usage_log` with `feature = "short_text_grading"`, `user_id = NULL` (system-initiated, no employee actor), and `tokens_used` from the API response.
- [ ] AC-7: The question editor frontend shows a "Model Answer" textarea (visible only when question `type = "shorttext"`) and an "Enable Auto-Grading" toggle; the toggle is disabled (and unchecked) when model_answer is empty; saving the question with `auto_grade = true` and no model answer returns a validation error.
- [ ] AC-8: The manual grading queue UI (FR-BB47) displays AI-graded questions with a distinct `"AI Graded"` badge and shows the AI's `reasoning` text as a collapsible note to assist the examiner's review.

## Technical Specification

### Configuration / Infrastructure

**New migrations** — `027_short_text_autograding.up.sql`:
```sql
ALTER TABLE questions
  ADD COLUMN auto_grade    BOOL NOT NULL DEFAULT false,
  ADD COLUMN model_answer  TEXT;

-- Store AI reasoning alongside score
ALTER TABLE session_question_scores
  ADD COLUMN ai_reasoning  TEXT;
```

**`027_short_text_autograding.down.sql`**:
```sql
ALTER TABLE session_question_scores DROP COLUMN IF EXISTS ai_reasoning;
ALTER TABLE questions DROP COLUMN IF EXISTS model_answer;
ALTER TABLE questions DROP COLUMN IF EXISTS auto_grade;
```

**`grading_status` enum values** (defined in migration `016`; no new values added by this feature):
- `'pending_manual'` — awaiting human grader
- `'ai_graded'` — scored by AI (overridable by examiner)
- `'graded'` — final score (set by deterministic engine for MC/TF, or by examiner override)

### API Endpoints

No new endpoints. Changes are in the grading engine internals and the question CRUD API.

#### PUT /api/v1/admin/questions/:id (modified)
- Accept `auto_grade: bool` and `model_answer: string | null` in request body.
- Validation: if `auto_grade = true`, `model_answer` must be non-empty; return `400` with `MISSING_MODEL_ANSWER` otherwise.
- `auto_grade` and `model_answer` are only valid for `type = "shorttext"`; return `400` with `INVALID_FIELD_FOR_TYPE` for other types.

#### POST /api/v1/admin/questions (modified)
- Same validation as PUT above for `auto_grade` / `model_answer`.

### Implementation Details

#### Database query extensions

The questions fetch query in `internal/sessions/grading.go` must be extended to also select `auto_grade` and `model_answer`, and `session_answers` must join on the `text_answer` column for `shorttext` questions. Update `gradingQuestionRow` accordingly:

```go
type gradingQuestionRow struct {
    QuestionID        string  `db:"question_id"`
    QuestionType      string  `db:"question_type"`
    SelectedOptionIDs []byte  `db:"selected_option_ids"`
    AutoGrade         bool    `db:"auto_grade"`
    ModelAnswer       *string `db:"model_answer"`
    TextAnswer        *string `db:"text_answer"`
}
```

Update the `questionsQ` query to include the new columns:
```sql
SELECT
    sq.question_id,
    q.type                                              AS question_type,
    COALESCE(sa.selected_option_ids, '[]'::jsonb)       AS selected_option_ids,
    q.auto_grade,
    q.model_answer,
    sa.text_answer
FROM session_questions sq
JOIN questions q ON q.id = sq.question_id
LEFT JOIN session_answers sa
    ON sa.session_id = sq.session_id AND sa.question_id = sq.question_id
WHERE sq.session_id = $1
```

#### AI-enabled grading engine

Introduce `AIGradingEngine` in `internal/sessions/grading.go` (package `sessions`). It wraps the `ai.AnthropicClient` interface and stores the model name. The existing `DefaultGradingEngine` is unchanged.

```go
// AIGradingEngine extends grading with Anthropic-backed short-text scoring.
type AIGradingEngine struct {
    aiClient ai.AnthropicClient
    model    string
    db       *sqlx.DB
    logger   *slog.Logger
}

// NewAIGradingEngine constructs an AIGradingEngine.
// model is the Anthropic model string (e.g. "claude-3-5-haiku-20241022").
func NewAIGradingEngine(aiClient ai.AnthropicClient, model string, db *sqlx.DB, logger *slog.Logger) GradingEngine {
    return &AIGradingEngine{aiClient: aiClient, model: model, db: db, logger: logger}
}
```

The `Grade` method on `AIGradingEngine` follows the same flow as `DefaultGradingEngine.Grade` but, for `shorttext` questions, delegates to `gradeShortText` instead of returning `pending_manual` immediately:

```go
func (e *AIGradingEngine) gradeShortText(
    sessionID, questionID string,
    autoGrade bool,
    modelAnswer, textAnswer *string,
) (score float64, maxScore float64, status GradingStatus, aiReasoning *string) {
    if !autoGrade || modelAnswer == nil || *modelAnswer == "" || textAnswer == nil {
        return 0, 1, GradingStatusPendingManual, nil
    }

    prompt := fmt.Sprintf(
        "You are grading a corporate exam short-answer question.\n\n"+
            "Model Answer (expert reference): %s\n\n"+
            "Employee's Answer: %s\n\n"+
            "Grade the employee's answer on a scale of 0 to 100 based on how well it "+
            "demonstrates understanding of the key concepts in the model answer. "+
            "Partial credit is appropriate.\n\n"+
            "Return ONLY valid JSON with no markdown:\n"+
            `{"score_pct": <integer 0-100>, "reasoning": "<1-2 sentence explanation>"}`,
        *modelAnswer, *textAnswer,
    )

    raw, tokensUsed, err := e.aiClient.GenerateText(context.Background(), prompt, e.model)
    if err != nil {
        e.logger.Warn("AI grading failed, falling back to manual queue", "session_id", sessionID, "question_id", questionID, "error", err)
        return 0, 1, GradingStatusPendingManual, nil
    }

    var parsed struct {
        ScorePct  int    `json:"score_pct"`
        Reasoning string `json:"reasoning"`
    }
    if err := json.Unmarshal([]byte(raw), &parsed); err != nil || parsed.ScorePct < 0 || parsed.ScorePct > 100 {
        e.logger.Error("AI response parse failed, falling back to manual queue", "session_id", sessionID, "question_id", questionID, "raw", raw, "error", err)
        return 0, 1, GradingStatusPendingManual, nil
    }

    // Log AI usage — user_id is NULL (system-initiated call, no employee actor).
    e.logAIUsage(context.Background(), "short_text_grading", tokensUsed)

    reasoning := parsed.Reasoning
    return float64(parsed.ScorePct) / 100.0, 1, GradingStatusAIGraded, &reasoning
}
```

`logAIUsage` inserts into `ai_usage_log` with `user_id = NULL`. The `user_id` in `ai_usage_log` is set to NULL for system-initiated grading calls because the AI grading is triggered by the grading engine, not by a specific authenticated user.
```go
func (e *AIGradingEngine) logAIUsage(ctx context.Context, feature string, tokensUsed int) {
    const q = `
INSERT INTO ai_usage_log (user_id, feature, tokens_used, model)
VALUES (NULL, $1, $2, $3)`
    // Best-effort; do not propagate failure.
    if _, err := e.db.ExecContext(ctx, q, feature, tokensUsed, e.model); err != nil {
        e.logger.Error("ai_usage_log insert failed", "error", err)
    }
}
```

The `session_question_scores` INSERT must be updated to also write `ai_reasoning`:
```sql
INSERT INTO session_question_scores
    (session_id, question_id, score, max_score, grading_status, ai_reasoning)
VALUES ($1, $2, $3, $4, $5, $6)
```

#### Wire-up

In `cmd/api/main.go` (or wherever `DefaultGradingEngine` is constructed), conditionally construct `AIGradingEngine` when `cfg.AnthropicAPIKey` is non-empty:

```go
var gradingEngine sessions.GradingEngine
if cfg.AnthropicAPIKey != "" {
    aiClient := ai.NewAnthropicClient(cfg.AnthropicAPIKey, logger)
    gradingEngine = sessions.NewAIGradingEngine(aiClient, cfg.AnthropicModel, db, logger)
} else {
    gradingEngine = sessions.NewGradingEngine()
}
```

`cfg.AnthropicModel` defaults to `"claude-3-5-haiku-20241022"` if not set.

### Frontend Components

**Question Editor** (`src/pages/admin/QuestionEditor.tsx`) — shorttext type additions:
```tsx
{watch('type') === 'shorttext' && (
  <div className="space-y-3">
    <div>
      <Label htmlFor="modelAnswer">{t('question.modelAnswerLabel')}</Label>
      <Textarea
        id="modelAnswer"
        {...register('model_answer')}
        placeholder={t('question.modelAnswerPlaceholder')}
        rows={4}
      />
    </div>
    <div className="flex items-center gap-3">
      <Switch
        id="autoGrade"
        checked={watch('auto_grade')}
        onCheckedChange={v => setValue('auto_grade', v)}
        disabled={!watch('model_answer')?.trim()}
      />
      <Label htmlFor="autoGrade">{t('question.enableAutoGrading')}</Label>
    </div>
  </div>
)}
```

**Manual Grading Queue** (`src/pages/admin/GradingQueue.tsx`) — AI badge and reasoning:
```tsx
{answer.grading_status === 'ai_graded' && (
  <div className="space-y-2">
    <Badge variant="secondary">{t('grading.aiGraded')}</Badge>
    {answer.ai_reasoning && (
      <Collapsible>
        <CollapsibleTrigger className="text-sm text-muted-foreground underline">
          {t('grading.viewAIReasoning')}
        </CollapsibleTrigger>
        <CollapsibleContent>
          <p className="text-sm text-muted-foreground mt-1">{answer.ai_reasoning}</p>
        </CollapsibleContent>
      </Collapsible>
    )}
  </div>
)}
```

On successful `PUT /api/v1/admin/questions/:id`, invalidate React Query keys `['questions', id]` and `['questions', 'list']` so that the question list and detail views reflect the updated `auto_grade` / `model_answer` values immediately.

**i18n keys to add** (in `src/locales/{kk,ru,en}.json`):
- `question.modelAnswerLabel`
- `question.modelAnswerPlaceholder`
- `question.enableAutoGrading`
- `grading.aiGraded`
- `grading.viewAIReasoning`

## Out of Scope

- Bulk re-grading of historical sessions with the new AI engine (no retroactive scoring).
- Per-question model selection (all AI grading uses the single tenant-level `AnthropicModel` config value).
- Streaming AI responses.
- Storing the full AI prompt or raw response beyond the parsed `reasoning` string.

## Test Strategy

### Backend unit tests (`internal/sessions/grading_test.go`)

1. **Happy path**: mock `AnthropicClient.GenerateText` returning valid JSON `{"score_pct":80,"reasoning":"good"}`. Assert `grading_status = 'ai_graded'`, `score = 0.80`, `ai_reasoning` populated.
2. **JSON parse failure**: mock returns non-JSON text. Assert fallback to `grading_status = 'pending_manual'`, `score = 0`, no error propagated.
3. **Out-of-range score**: mock returns `{"score_pct":150,"reasoning":"x"}`. Assert fallback to `pending_manual`.
4. **AI client error**: mock returns `error`. Assert fallback to `pending_manual` and that the overall `Grade` call does not return an error.
5. **auto_grade = false**: assert shorttext question routes directly to `pending_manual` without calling `GenerateText`.
6. **model_answer = NULL**: same as above — no AI call.

### Backend integration test

Confirm `ai_usage_log` row is inserted with `user_id IS NULL` and correct `tokens_used` after a successful AI grading call.

### API validation tests

- `PUT /questions/:id` with `auto_grade=true` and empty `model_answer` → `400 MISSING_MODEL_ANSWER`.
- `PUT /questions/:id` with `auto_grade=true` on a `singlechoice` question → `400 INVALID_FIELD_FOR_TYPE`.

### Frontend component tests

- `QuestionEditor`: when `type = "shorttext"`, model answer textarea and auto-grade toggle render.
- `QuestionEditor`: toggle is disabled when `model_answer` is empty; enabled after typing into textarea.
- `QuestionEditor`: submitting with `auto_grade=true` and blank `model_answer` shows validation error without API call.
- `GradingQueue`: answer with `grading_status = 'ai_graded'` shows `"AI Graded"` badge; collapsible reasoning opens on click.
```

## Notes
- The `user_id` in `ai_usage_log` is set to `NULL` for AI grading calls, because the grading engine is a system process not associated with an authenticated user. This avoids foreign-key issues and matches AC-6.
- AI grading is run synchronously within the grading engine; if it becomes a bottleneck (many short-text questions in one session), it can be moved to a background worker queue in a future phase.
- The `ai_reasoning` column on `session_question_scores` stores the raw text from Claude; it is surfaced to examiners but never to employees.
- Employee-facing result screens must not display `ai_reasoning` or reveal that AI grading was used, to avoid gaming.

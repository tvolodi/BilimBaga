# FR-BB73 — Short-Text Auto-Grading

## Metadata
| Field | Value |
|-------|-------|
| ID | FR-BB73 |
| Phase | 7 — AI Layer |
| Priority | 1 |
| Status | Draft |
| Depends On | FR-BB311, FR-BB71 |

## Description
Extends the grading engine so that short-text questions with a configured model answer are automatically graded by Anthropic Claude. The AI returns a percentage score and reasoning, which are stored in the session results. Examiners can override the AI grade at any time via the existing manual grading endpoints. If the AI is unavailable, the question falls back gracefully to the manual grading queue.

## Acceptance Criteria
- [ ] AC-1: The `questions` table gains two new columns: `auto_grade BOOL NOT NULL DEFAULT false` and `model_answer TEXT`; these columns are only meaningful for `shorttext` question type.
- [ ] AC-2: When the grading engine processes a `shorttext` question where `auto_grade = true` AND `model_answer IS NOT NULL`, it calls the Anthropic Claude API to score the employee's text answer; the result is stored with `grading_status = 'ai_graded'`.
- [ ] AC-3: The Anthropic prompt instructs the model to return a JSON object `{ "score_pct": 0-100, "reasoning": "string" }` and nothing else; if the response cannot be parsed as valid JSON matching this schema, the question falls back to `grading_status = 'pending_manual'` and an error is logged.
- [ ] AC-4: If the Anthropic API returns an error or is unreachable, the question is placed in the manual grading queue (`grading_status = 'pending_manual'`) rather than failing the grading run; a warning is logged with the session ID and question ID.
- [ ] AC-5: Examiners can override an AI-graded score via the existing manual grading endpoints (FR-BB42); the override sets `grading_status = 'manually_graded'` and replaces the stored score; the `reasoning` from the AI is preserved as a reference note.
- [ ] AC-6: Every AI grading call is logged to `ai_usage_log` with `feature = "short_text_grading"`, `user_id` set to the system actor UUID (not the employee's ID), and `tokens_used` from the API response.
- [ ] AC-7: The question editor frontend shows a "Model Answer" textarea (visible only when question `type = "shorttext"`) and an "Enable Auto-Grading" toggle; the toggle is disabled (and unchecked) when model_answer is empty; saving the question with `auto_grade = true` and no model answer returns a validation error.
- [ ] AC-8: The manual grading queue UI (FR-BB47) displays AI-graded questions with a distinct `"AI Graded"` badge and shows the AI's `reasoning` text as a collapsible note to assist the examiner's review.

## Technical Specification

### Configuration / Infrastructure

**New migration** — `migrations/NNN_short_text_autograding.sql`:
```sql
ALTER TABLE questions
  ADD COLUMN auto_grade    BOOL NOT NULL DEFAULT false,
  ADD COLUMN model_answer  TEXT;

-- Store AI reasoning alongside score
ALTER TABLE session_question_scores
  ADD COLUMN ai_reasoning  TEXT;
```

**`grading_status` enum values** (ensure these exist or add via migration):
- `'pending_manual'` — awaiting human grader
- `'ai_graded'` — scored by AI (overridable)
- `'manually_graded'` — scored by human (final)
- `'auto_graded'` — scored by deterministic grading engine (existing for MC/TF)

### API Endpoints

No new endpoints. Changes are in the grading engine internals and the question CRUD API.

#### PUT /api/v1/admin/questions/:id (modified)
- Accept `auto_grade: bool` and `model_answer: string | null` in request body.
- Validation: if `auto_grade = true`, `model_answer` must be non-empty; return `400` with `MISSING_MODEL_ANSWER` otherwise.
- `auto_grade` and `model_answer` are only valid for `type = "shorttext"`; return `400` with `INVALID_FIELD_FOR_TYPE` for other types.

#### POST /api/v1/admin/questions (modified)
- Same validation as PUT above for `auto_grade` / `model_answer`.

### Implementation Details

**Grading engine extension** (`internal/grading/engine.go`):
```go
func (e *Engine) gradeShortText(
    ctx context.Context,
    question *Question,
    answer *SessionAnswer,
) (*QuestionScore, error) {
    // Auto-grade path
    if question.AutoGrade && question.ModelAnswer != nil {
        score, reasoning, err := e.aiClient.GradeShortText(
            ctx, question.Stem, *question.ModelAnswer, answer.TextAnswer,
        )
        if err != nil {
            // Fallback to manual queue
            e.logger.Warn().
                Err(err).
                Str("session_id", answer.SessionID.String()).
                Str("question_id", question.ID.String()).
                Msg("AI grading failed, falling back to manual queue")
            return &QuestionScore{
                Status: GradingStatusPendingManual,
                Score:  0,
            }, nil
        }
        e.logAIUsage(ctx, "short_text_grading", score.TokensUsed, e.aiClient.Model())
        return &QuestionScore{
            Status:      GradingStatusAIGraded,
            Score:       float64(score.ScorePct) / 100.0, // normalize to 0.0-1.0
            AIReasoning: &reasoning,
        }, nil
    }

    // Manual queue path (no auto_grade or no model_answer)
    return &QuestionScore{Status: GradingStatusPendingManual, Score: 0}, nil
}
```

**AI client method** (`internal/ai/client.go`):
```go
type ShortTextGradeResult struct {
    ScorePct   int
    Reasoning  string
    TokensUsed int
}

func (c *Client) GradeShortText(
    ctx context.Context,
    questionStem, modelAnswer, employeeAnswer string,
) (*ShortTextGradeResult, error) {
    prompt := fmt.Sprintf(`
You are grading a corporate exam short-answer question.

Question: %s

Model Answer (expert reference): %s

Employee's Answer: %s

Grade the employee's answer on a scale of 0 to 100 based on how well it demonstrates
understanding of the key concepts in the model answer. Partial credit is appropriate.

Return ONLY valid JSON with no markdown:
{"score_pct": <integer 0-100>, "reasoning": "<1-2 sentence explanation>"}`,
        questionStem, modelAnswer, employeeAnswer,
    )

    resp, err := c.anthropic.Messages.New(ctx, anthropic.MessageNewParams{
        Model:     anthropic.F(c.model),
        MaxTokens: anthropic.F(int64(256)),
        Messages: anthropic.F([]anthropic.MessageParam{
            anthropic.UserMessageParam(anthropic.ContentBlockParamOfRequestTextBlock(prompt)),
        }),
    })
    if err != nil { return nil, fmt.Errorf("anthropic API: %w", err) }

    var result struct {
        ScorePct  int    `json:"score_pct"`
        Reasoning string `json:"reasoning"`
    }
    content := resp.Content[0].Text
    if err := json.Unmarshal([]byte(content), &result); err != nil {
        return nil, fmt.Errorf("parse AI response: %w", err)
    }
    if result.ScorePct < 0 || result.ScorePct > 100 {
        return nil, fmt.Errorf("AI returned out-of-range score: %d", result.ScorePct)
    }

    tokensUsed := int(resp.Usage.InputTokens + resp.Usage.OutputTokens)
    return &ShortTextGradeResult{
        ScorePct:   result.ScorePct,
        Reasoning:  result.Reasoning,
        TokensUsed: tokensUsed,
    }, nil
}
```

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

## Notes
- The system actor UUID (used in `ai_usage_log.user_id` for grading calls) should be a fixed, well-known UUID constant defined in `internal/constants/actors.go` to distinguish system-initiated AI calls from user-initiated ones.
- AI grading is run synchronously within the grading engine; if it becomes a bottleneck (many short-text questions in one session), it can be moved to a background worker queue in a future phase.
- The `ai_reasoning` column on `session_question_scores` stores the raw text from Claude; it is surfaced to examiners but never to employees.
- Employee-facing result screens must not display `ai_reasoning` or reveal that AI grading was used, to avoid gaming.

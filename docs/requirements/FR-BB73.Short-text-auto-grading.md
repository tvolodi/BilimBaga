# FR-BB73 — Short-Text Auto-Grading

## Metadata
| Field | Value |
|-------|-------|
| ID | FR-BB73 |
| Phase | 7 — AI Layer |
| Priority | 1 |
| Status | Implemented (revision 1); revision 2 Validated 2026-10-11 (PASS on the fourth draft), issue #534, architect constraints on the issue; merges after FR-BB322 (PR #541), which it cites |
| Depends On | FR-BB311, FR-BB71, FR-BB42, FR-BB47, FR-BB43, FR-BB312; FR-BB322 (pending-result experience, #535, defines `grading_expected_by`, whose predicate gains the mode clause of AC-11) |

## Affected Layers

| Layer | Items |
|-------|-------|
| Database | `questions` table (new `auto_grade`, `model_answer` columns), `session_question_scores` table (new `ai_reasoning` column), `ai_usage_log` table (read/write) |
| Backend | `internal/sessions/grading.go`, `internal/questions/` (repo + service + handler), migration `027_short_text_autograding` |
| Frontend | `QuestionEditor` (type=shorttext fields), `GradingQueue` (AI badge + reasoning); revision 2: exam wizard Step 1 select, suggestion panel in the grading screen |
| Revision 2 | `exams` (column, API), `session_question_scores` (three columns), `audit_log` (actions), `email` (`exam_result_corrected`), `certificates` (session read lock), FR-BB47 AC-5 amendment |

## Description
Extends the grading engine so that short-text questions with a configured model answer are automatically graded by the configured AI provider (Anthropic, or an OpenAI-compatible endpoint such as x.ai; section D of revision 2). The AI returns a percentage score and reasoning, which are stored in the session results. Examiners can override the AI grade at any time via the existing manual grading endpoints. If the AI is unavailable, the question falls back gracefully to the manual grading queue.

## Acceptance Criteria
- [ ] AC-1: The `questions` table gains two new columns: `auto_grade BOOL NOT NULL DEFAULT false` and `model_answer TEXT`; these columns are only meaningful for `shorttext` question type.
- [ ] AC-2: When the grading engine processes a `shorttext` question where `auto_grade = true` AND `model_answer IS NOT NULL`, the answer becomes a candidate for the background AI job (revision 2, AC-9 to AC-12); what the job does with the AI score depends on the exam's `ai_grading_mode` (`auto`: stored as `ai_graded` unless the answer is flagged (AC-14); `suggest`: stored only as a suggestion).
- [ ] AC-3: The Anthropic prompt instructs the model to return a JSON object `{ "score_pct": 0-100, "reasoning": "string" }` and nothing else; if the response cannot be parsed as valid JSON matching this schema, the question falls back to `grading_status = 'pending_manual'` and an error is logged.
- [ ] AC-4: If the Anthropic API returns an error or is unreachable, the question is placed in the manual grading queue (`grading_status = 'pending_manual'`) rather than failing the grading run; a warning is logged with the session ID and question ID.
- [ ] AC-5: Examiners can override an AI-graded score via the existing manual grading endpoints (FR-BB42); the override sets `grading_status = 'graded'` and replaces the stored score; the `reasoning` from the AI is preserved as a reference note. Re-grading rules and their limits: revision 2, AC-19 to AC-21.
- [ ] AC-6: Every AI grading call is logged to `ai_usage_log` with `feature = "short_text_grading"`, `user_id = NULL` (system-initiated, no employee actor), and `tokens_used` from the API response.
- [ ] AC-7: The question editor frontend shows a "Model Answer" textarea (visible only when question `type = "shorttext"`) and an "Enable Auto-Grading" toggle; the toggle is disabled (and unchecked) when model_answer is empty; saving the question with `auto_grade = true` and no model answer returns a validation error.
- [ ] AC-8: The manual grading queue UI (FR-BB47) displays AI-graded questions with a distinct `"AI Graded"` badge and shows the AI's `reasoning` text as a collapsible note to assist the examiner's review.

## Revision 2: grading policy, modes and provider (issue #534, 2026-10-11)

Why. Since PR 529 (#528) an AI job grades short-text answers outside the submit transaction and finalises the session from the AI score: pass or fail, the result mail and the certificate follow with no person involved. The employee's text is part of the grader prompt, so a manipulated answer can steer the score (the answer-injection risk named by the architect). This revision makes the trust in an AI score an explicit, per-exam decision, adds the rules that were missing around re-grading, and records the provider configuration that has no requirement of its own. Architect constraints: comment `architect-decision: approve` on #534 (T1). The default mode is the owner's product choice; this revision states `suggest` (the BA and architect recommendation). If the owner chooses otherwise, AC-9 is amended before the dev issue is dispatched.

Terms. `off`: the AI is never asked. `suggest`: the AI result is stored as a suggestion for a person; it changes nothing else. `auto`: the AI result is the grade (`ai_graded`) unless the answer is flagged.

### A. Modes and storage
- [ ] AC-9: `exams.ai_grading_mode TEXT NOT NULL DEFAULT 'suggest'` with `CHECK (ai_grading_mode IN ('off','suggest','auto'))`, in one new numbered migration (number taken under `swarm/locks/migration.lock`; 040 at the time of writing). Existing exams get `suggest`, so a host that runs `auto` today for every exam moves to `suggest` on deploy; the dev PR states this in its release note. Exams created later take the default unless the request names a mode (AC-22). The exam may be updated while published: the job reads the mode at grading time, so a change applies to answers not yet claimed.
- [ ] AC-10: `session_question_scores` gains `ai_suggested_score_pct NUMERIC NULL`, `ai_suggested_at TIMESTAMPTZ NULL` and `ai_flags TEXT[] NULL`. `ai_flags` is a firm addition to the architect's two columns: it is write-only for scoring and holds the codes of AC-15, because `decisive_pass` depends on session state at write time and cannot be recomputed on read. `ai_suggested_score_pct IS NULL` means no suggestion; `ai_flags = '{}'` means suggested with no flags. `ai_reasoning` already exists. The down migration drops the three columns and the exam column with its check. None of the columns is read by any score, `grading_status`, session status, pass/fail, mail or certificate computation (AC-13); the only reads are routing and display (AC-11, AC-17, AC-18, and `grading_expected_by` of FR-BB322).
- [ ] AC-11: The job's candidate query selects a row only when its exam has `ai_grading_mode IN ('suggest','auto')`, read at grading time, plus the conditions of #528 (`pending_manual`, question `auto_grade` with a non-empty model answer, answer text not null, session submitted in the last 24 h, `ai_grade_attempts < 2`; plus: an AI provider is configured and the count of `ai_usage_log` rows of feature `short_text_grading` in the last 24 h is below `AI_GRADING_DAILY_LIMIT`) and `ai_suggested_score_pct IS NULL`, so an answer receives at most one suggestion. Rows of `off` exams are never selected. The conditions of #528 are one shared per-session function used by the job and by `grading_expected_by` of FR-BB322 AC-4. This AC adds two clauses: the job accepts exams in `suggest` and `auto` and requires `ai_suggested_score_pct IS NULL`; `grading_expected_by` is `"ai"` only for exams in `auto` with no stored suggestion, and `"examiner"` for `off` and `suggest` exams and for answers that hold a suggestion.
- [ ] AC-12: Kept from #528 without change: the claim on `ai_grade_attempts`, the 30 s deadline outside any transaction, the write only while the row is `pending_manual`, the random per-call markers, the 4000-rune cap, and the daily cap `AI_GRADING_DAILY_LIMIT` (default 500; 0 turns the job off). A suggestion costs one provider call and counts against the same cap and the same `ai_usage_log` feature `short_text_grading`.

### B. The `suggest` invariant and the `auto` path
- [ ] AC-13: In `suggest`, an AI output never changes a `score`, a `grading_status`, the session `status`, `passed`, `score_pct`, a result mail, or any input of certificate issuance. The row stays `pending_manual`; the job stores `ai_suggested_score_pct`, `ai_suggested_at`, `ai_reasoning` and `ai_flags` and writes an audit row with action `answer.ai_suggest` (`entity_type = 'session_answer'`, metadata: session id, suggested score, flag codes); it does not call the finalise helper and sends no mail. The suggest branch is one named function, `applyAISuggestion` in `aigradejob.go` (the auto branch is `applyAIGrade`; today both are inside `writeAIGrade`, which is split); a source guard test asserts that `applyAISuggestion` has no call to `finaliseSessionIfAllGraded` or the email service.
- [ ] AC-14: In `auto`, an unflagged answer is graded as in revision 1: `ai_graded`, finalise through `finaliseSessionIfAllGraded`, audit action `answer.ai_grade`, result mail through the existing path once. The row also stores `ai_suggested_score_pct`, `ai_suggested_at` and `ai_flags = '{}'`, so `NULL` always means "the AI was not consulted or gave no usable answer". A flagged answer in `auto` is handled exactly as in `suggest` (AC-13), whatever the mode of the exam.
- [ ] AC-15: Flags. One pure function `flagAIGrade(answerText string, suggestedScorePct float64, modelAnswer string, ctx flagContext) []string` in `internal/sessions/aiflags.go`; it has a table test and is never delegated to the model. `flagContext` carries: `OtherPendingCount` (pending answers of the session besides this one), `OtherPointsEarned` (sum of points already graded), `ThisQuestionPoints`, `TotalPoints` and `PassingScorePct` of the exam. Rules, thresholds are named constants in that file:
  - `instruction_like`: the answer, lower-cased with whitespace collapsed, contains any phrase of the list in that file. The list starts with: `ignore previous`, `ignore all`, `ignore the above`, `disregard`, `system prompt`, `you are now`, `as the grader`, `give full`, `full marks`, `maximum score`, `score is`, `score of 100`, `100%`, and the Russian `игнорируй`, `забудь`, `поставь 100`, `максимальный балл`, `полный балл`. The Kazakh phrases are added by the content owner as one-line additions; the function takes the list from one table so no code change is needed.
  - `short_high`: trimmed answer shorter than 20 runes, suggested score at least 80, and model answer longer than 100 runes.
  - `decisive_pass`: evaluated only when `OtherPendingCount = 0`, inside the write transaction under the `exam_sessions` row lock: the session would pass with this score and would fail with this answer at 0.
  A flag only routes an answer to a person, so a false positive on a phrase such as `score is` or `100%` costs a click, not a wrong grade; the list is kept short and is reviewed with the false-positive count of the audit rows. The table test has at least two positive and two negative cases per rule, and one case per language of the list.
- [ ] AC-16: A flagged answer in `auto` stays `pending_manual` with the suggestion stored; it appears in the grading queue with its flag codes. The audit row of AC-13 carries the codes.

### C. Confirming a suggestion (examiner)
- [ ] AC-17: No new endpoint. The examiner confirms by `POST /api/v1/admin/grading/{sessionId}/answers/{questionId}` (FR-BB42) with a score; accepting the suggestion is posting the suggested value. Permission (`grading:write`) and department scope are the existing ones. The server, not the client, writes in the audit row of the grade: `suggested_score_pct` (or null), `accepted` (true when the posted score equals the suggestion), and the flag codes. The row becomes `graded` with `graded_by` set to the person; `ai_graded` remains the status of `auto` only. A person may post a different score; that is recorded as `accepted = false`.
- [ ] AC-18: The grading detail response (FR-BB42) returns, for pending rows, `ai_suggested_score_pct`, `ai_reasoning` and `ai_flags`. The grading screen (FR-BB47) shows the suggestion with an "Accept" button that posts the suggested value, the flags as chips, and the reasoning rendered as plain text only: never HTML, never markdown (a test renders `<img src=x onerror=alert(1)>` and `**bold**` and asserts literal text). The employee never receives these fields.

### D. Rulings on existing gaps (found by the architect while reading the code)
- [ ] AC-19: Re-grading. Department scope is enforced before the handler by the existing `gradingScope` middleware. `GradeAnswer` then runs in one transaction, in this order: (1) `lockSessionForGrading` (404 for an unknown session); (2) the `user_id` of the session is read under that lock and compared with the acting user (AC-21, 403); (3) the row rules below.
  - Session `grading_pending`: any row may be graded or re-graded by a person with `grading:write` (the grading screen of FR-BB47 pre-fills graded answers and posts all of them, AC-5 and AC-8 there, and keeps working unchanged). A row that was `ai_graded` is an override: audit action `answer.ai_override` with the previous and the new score. No mail is sent, the session is not final.
  - Session final (`submitted`): a row `graded` answers 409 `SESSION_FINALISED` and nothing is written. This closes the silent re-grade of a finished session (it changed a final score and re-sent the result mail with no rule). FR-BB47 AC-5 is amended to say a final session's graded answers are shown read-only. A row `ai_graded` of a final session (only `auto` produces them) is an override as above, allowed unless AC-20 forbids it; it recomputes the session through the one finalise path; no second result mail unless pass/fail flips, and when it flips exactly one mail `exam_result_corrected` goes to the employee (new template, kk, ru, en, stating the corrected outcome), because an employee who was told "passed" must not find "failed" on the page without notice. "Flips" is the `passed` value read under the lock before the override against the recomputed value.
- [ ] AC-20: Certificates. An override (AC-19) of a final session for which a certificate has been issued answers 409 `CERTIFICATE_ISSUED` and changes nothing; the existence of the certificate is read inside the override transaction under the session lock. Certificate issuance (`certificates.GetOrCreate`, today separate statements on `r.db`) runs in one transaction: `SELECT ... FROM exam_sessions ... FOR SHARE`, then the status and `passed` checks, the existence check and the insert, then commit, so an override that holds the row lock and an issuance cannot interleave (a Postgres test runs both concurrently and asserts that either the certificate exists and the override got 409, or the override won and issuance sees the new `passed`). Revocation of a certificate is out of scope; a session without an issued certificate may be overridden. (Certificates are issued on request, so one exists only after the employee or an admin asked for it.)
- [ ] AC-21: Self-grading. `GradeAnswer` answers 403 `CANNOT_GRADE_OWN_SESSION` when the acting user id equals the `user_id` of the session, for every role including `super_admin`; nothing is written. The comparison is a repository read of `exam_sessions.user_id` under the session lock (the department scope lets a caller see itself, so scope cannot be relied on). Consequence accepted: an installation with one `super_admin` who also sat the exam needs a second person with `grading:write` to grade that session. Handler test for status and code.

### E. Exam setting and provider configuration
- [ ] AC-22: Exam setting. The exam wizard Step 1 (`frontend/src/pages/ExamWizard/Step1BasicSettings.tsx`) gets an "AI grading" select with `off`, `suggest`, `auto`, always shown (rules select questions at run time, so the form cannot know whether a short-text question will be drawn); help text per value in kk, ru, en. The value is part of the exam create and update requests (`exams` `model.go`: `Exam`, the create and update request structs) as `ai_grading_mode`; any other value is 422 `VALIDATION_ERROR`; omitted on create takes the default of AC-9, omitted on update leaves it unchanged. The exam detail and the grading queue show the mode.
- [ ] AC-23: Provider configuration, documented as it is in `backend/internal/config/ai.go` (this AC adds no code): `AI_PROVIDER` is `anthropic` or `openai_compatible` (anything else fails startup, naming the variable); with it unset, `ANTHROPIC_API_KEY` selects anthropic, otherwise an x.ai key (`XAI_API_KEY`) selects openai_compatible at `https://api.x.ai/v1` with model `grok-4.7`, otherwise AI is off; `AI_API_KEY` wins over the vendor-named key; an x.ai key is used only when the base URL host is `api.x.ai`, and an Anthropic key is never used for `openai_compatible`; `AI_BASE_URL` must be https (http only for localhost) and carry no credentials; `AI_MODEL` is required for openai_compatible at any host other than api.x.ai; `AI_TIMEOUT_SECONDS` is 10 to 150 (default 90); `AI_GRADING_DAILY_LIMIT` (default 500, 0 off) and `AI_INSIGHTS_DAILY_LIMIT` (default 50) cap provider calls per 24 h. A key is never logged or printed: startup writes one INFO line with provider and model, or one WARN line with the reason for "AI is off". With AI off, the AI routes under `/api/v1/admin/ai/` answer 503 `AI_UNAVAILABLE`, the job does not start, and every short-text answer stays in the manual queue. `deploy/` documentation and `.env.example` list the variables; key rules are owner decisions per host (data goes to the provider).
- [ ] AC-24: Tests: migration up and down; candidate query per mode (`off` never, `suggest` once, `auto` unflagged graded, `auto` flagged suggested); the suggest source guard; the flag table test; `GradeAnswer` 409 and 403 cases and the override with and without a certificate; a Postgres test of the decisive-pass flag under the session lock (no `t.Skip` in CI, the verbose `TestAIGrade_` step of `backend-tests` shows PASS); frontend: the select, the suggestion panel, the plain-text rendering.

### Rollout and release note
Merging deploys nothing. On a host with a key, answers `pending_manual` from the last 24 h are suggested or graded within minutes of a deploy according to the mode of their exam; the owner is told before QA or the demo is redeployed. Until this revision is implemented, `AI_GRADING_DAILY_LIMIT=0` is the only switch that turns grading off (owner's call per host).

### Out of scope
Certificate revocation; correcting a human grade; any AI grading for question types other than short text; per-question mode; employee-visible AI output.

## Technical Specification (revision 2)

Sections of the original Technical Specification below that describe `AIGradingEngine` and synchronous grading are historical (see the superseded notes); the rules of this revision and of PR 529 apply.

### Database Schema
Migration `NNN_ai_grading_mode.up.sql` / `.down.sql` (number under the migration lock, AC-9):
```sql
ALTER TABLE exams ADD COLUMN ai_grading_mode TEXT NOT NULL DEFAULT 'suggest'
  CHECK (ai_grading_mode IN ('off','suggest','auto'));
ALTER TABLE session_question_scores
  ADD COLUMN ai_suggested_score_pct NUMERIC NULL,
  ADD COLUMN ai_suggested_at        TIMESTAMPTZ NULL,
  ADD COLUMN ai_flags               TEXT[] NULL;
```
The down file drops the three columns and then the exam column.

### API Contract
- No new endpoint. `POST /api/v1/admin/grading/{sessionId}/answers/{questionId}` (FR-BB42) gains the error codes 409 `SESSION_FINALISED`, 409 `CERTIFICATE_ISSUED`, 403 `CANNOT_GRADE_OWN_SESSION` (AC-19 to AC-21).
- Exam create and update (`/api/v1/admin/exams`): optional `ai_grading_mode` (AC-22).
- Grading detail (`GET /api/v1/admin/grading/{sessionId}`): each question item gains `ai_suggested_score_pct`, `ai_suggested_at`, `ai_reasoning`, `ai_flags` (nullable); never on an employee endpoint.
- Audit actions: `answer.ai_grade` (existing), `answer.ai_suggest`, `answer.ai_override`; the grade audit row gains `suggested_score_pct`, `accepted`, `flags` in metadata (AC-17).
- Mail template `exam_result_corrected` (kk, ru, en).

### Go Implementation Notes
- `internal/sessions/aigradejob.go`: candidate query with the mode join; `writeAIGrade` split into `applyAISuggestion` and `applyAIGrade`; flags computed in `aiflags.go`; `decisive_pass` computed inside the write transaction under the session lock; one shared eligibility function (AC-11).
- `internal/sessions/repository.go` `GradeAnswer`: checks after `lockSessionForGrading` in the order of AC-19; `GradingQuestionItem` gains the four fields.
- `internal/exams`: model, create and update structs, validation.
- `internal/certificates/service.go` and its repository: issuance in one transaction that reads the session `FOR SHARE` first (AC-20).
- `internal/email`: `exam_result_corrected`.

### Frontend
- `frontend/src/pages/ExamWizard/Step1BasicSettings.tsx`: the select (AC-22); `frontend/src/components/grading/QuestionGrader.tsx` and the grading detail page: suggestion panel, Accept button, flag chips, plain-text reasoning (AC-18); invalidate the grading detail and queue query keys after Accept.
- i18n keys (kk, ru, en) under `grading.suggestion.*`, `grading.flag.*` (one per code of AC-15), `examWizard.aiGradingMode.*` (label and help for each value).

## Technical Specification (revision 1, historical)

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

> Superseded by revision 2 and by PR 529 (#528): `AIGradingEngine` no longer exists; the provider is called only by `AIGradeJob`, outside any transaction. Kept below as history of revision 1.

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

## Out of Scope (revision 1, historical)

- Bulk re-grading of historical sessions with the new AI engine (no retroactive scoring).
- Per-question model selection (all AI grading uses the single tenant-level `AnthropicModel` config value).
- Streaming AI responses.
- Storing the full AI prompt or raw response beyond the parsed `reasoning` string.

## Test Strategy (revision 1, historical; revision 2 tests are AC-24)

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

## Notes
- The `user_id` in `ai_usage_log` is set to `NULL` for AI grading calls, because the grading engine is a system process not associated with an authenticated user. This avoids foreign-key issues and matches AC-6.
- (Historical, revision 1; superseded by PR 529 and revision 2) AI grading is run synchronously within the grading engine; if it becomes a bottleneck (many short-text questions in one session), it can be moved to a background worker queue in a future phase.
- The `ai_reasoning` column on `session_question_scores` stores the raw text from Claude; it is surfaced to examiners but never to employees.
- Employee-facing result screens must not display `ai_reasoning` or reveal that AI grading was used, to avoid gaming.

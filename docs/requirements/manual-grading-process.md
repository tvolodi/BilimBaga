---
slug: manual-grading
title: "Manual Grading"
type: process-description
status: uat-verified
created: 2026-06-09
related_requirements: [FR-BB42, FR-BB47, FR-BB73]
---

## Business Goal

When an exam includes short-text questions, automated grading alone cannot produce a final score. An examiner must evaluate each free-text answer against the expected answer and assign a score. This process ensures that every session requiring human review is identified, graded in a timely manner, and has its final score calculated and communicated to the employee. Success means no session is left indefinitely in `grading_pending` status, and every employee receives a definitive pass/fail result.

## Actors

| Actor | Role |
|-------|------|
| Examiner | Reviews text answers, assigns scores and optional feedback, submits grades |
| Super Admin | Has full access to the grading queue; can grade any session |
| AI Assistant | Provides automated score suggestions for short-text answers (Phase 7, when enabled) |
| Employee | Receives final result once all pending grades are submitted |

## Process Steps

### Step 1 — Session Enters the Grading Queue (System)

1. An employee submits a session that contains at least one short-text answer.
2. The grading engine processes all objective questions (single, multiple, true/false, Likert).
3. For short-text questions: the engine marks each question `pending_manual_grade`.
4. The session status is set to `grading_pending`.
5. A notification email is sent to examiners informing them that a new session awaits grading.
6. **Expected outcome**: The session appears in the Manual Grading queue.

### Step 2 — Open the Grading Queue (Examiner)

1. Examiner navigates to the Manual Grading section in the admin sidebar.
2. Examiner sees a table of sessions with status `grading_pending`, showing: employee name, exam name, submission date and time, and the number of questions still awaiting a grade.
3. Examiner can filter by exam name or date range.
4. **Expected outcome**: Examiner has a clear view of all sessions awaiting their attention.

### Step 3 — Grade a Session (Examiner)

1. Examiner clicks on a session row to open the grading detail page.
2. The page shows one short-text question at a time (with navigation to next/previous).
3. For each question, the examiner sees:
   - The question stem.
   - The employee's text answer.
   - The model answer (if one was provided when the question was authored).
   - A score input: a slider or numeric field from 0 to 100 (representing the percentage of points for this question).
   - An optional feedback text field (visible to the employee on the result screen if show_answers is enabled).
   - An AI-suggested score (if AI auto-grading is enabled and available for this question, Phase 7).
4. Examiner enters a score and optionally writes feedback.
5. Examiner moves to the next question.
6. Once all questions have scores entered, examiner clicks "Submit all grades".
7. **Expected outcome**: All short-text questions for the session have a score.

### Step 4 — Final Score Calculation (System)

1. The system detects that all `pending_manual_grade` questions in the session now have a score.
2. The system recalculates the total session score:
   - Objective question scores (already calculated) + short-text scores.
   - Total = `sum(all question_scores) / sum(max_possible_scores) × 100`.
3. The system sets the session `passed` flag based on the exam's passing threshold.
4. The session status changes from `grading_pending` to `submitted`.
5. A notification email is sent to the employee: pass (with certificate link if enabled) or fail (with retake information).
6. The audit log records each grade submitted by the examiner.
7. **Expected outcome**: The employee can now see their final result and download their certificate if they passed.

### Step 5 — AI-Suggested Grading (Examiner, Phase 7)

If a short-text question has `auto_grade = true` and a model answer set:

1. The grading engine sends the employee's answer and the model answer to the AI service.
2. The AI returns a suggested score (0–100) with a brief rationale.
3. The score is pre-filled in the score field, labelled "AI-suggested".
4. The examiner can accept the suggestion, adjust it, or override it entirely.
5. If the AI service is unavailable, the question falls back to the manual queue with no pre-fill.
6. **Expected outcome**: Examiners work faster with AI assistance while retaining full control over the final grade.

## Business Rules

- A session remains in `grading_pending` until ALL short-text questions have been graded; a partial grade submission is not allowed.
- An examiner cannot grade their own sessions (conflict of interest prevention).
- Once grades are submitted, the examiner cannot re-open and change them without super admin intervention.
- An AI-suggested score is a suggestion only; the examiner's submitted score is always the authoritative value.
- The score per short-text question is 0–100 representing the fraction of the full point value assigned to that question.
- Short-text questions that have `auto_grade = true` with a model answer are processed by the AI first; if the AI result arrives, it does not require examiner action but remains overridable.
- Every grading action is recorded in the audit log with the examiner's identity, the score assigned, and the timestamp.

## Acceptance Criteria (business language)

1. After an employee submits an exam with short-text questions, the session appears in the grading queue within one minute.
2. The grading queue shows the number of questions pending per session; a session with all questions graded no longer appears in the queue.
3. An examiner can score a short-text answer and add feedback text; both are saved when they click "Submit all grades".
4. After an examiner submits all grades for a session, the employee's result screen shows the final score and pass/fail banner.
5. The grading queue can be filtered by exam name to allow examiners to focus on their assigned content.
6. Each grade submission appears in the audit log with the examiner's name, the score given, and the date/time.

## Out of Scope

- Peer grading (grading by employees).
- Blind grading (examiner cannot see employee identity — not currently supported).
- Grading rubrics or structured scoring criteria beyond a single 0–100 score per question.

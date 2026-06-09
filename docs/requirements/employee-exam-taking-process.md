---
slug: employee-exam-taking
title: "Employee Exam Taking"
type: process-description
status: uat-verified
created: 2026-06-09
related_requirements: [FR-BB34, FR-BB35, FR-BB36, FR-BB37, FR-BB38, FR-BB39, FR-BB310, FR-BB311, FR-BB313, FR-BB314, FR-BB72]
---

## Business Goal

The organisation needs employees to complete assigned assessments under fair, consistent, and tamper-resistant conditions. This process covers everything from the employee deciding to start an exam through to the system grading it and notifying the employee of their result. Success means every session produces a trustworthy, reproducible score — regardless of whether the employee submitted manually, ran out of time, or switched tabs — and the organisation can rely on the outcome for compliance or competency decisions.

## Actors

| Actor | Role |
|-------|------|
| Employee | Takes the exam: starts, answers questions, submits |
| System | Manages session lifecycle: enforces time limits, detects integrity events, auto-submits expired sessions, grades objective questions |
| Examiner / Admin | Reviews results; handles manual grading of short-text questions after submission |

## Process Steps

### Step 1 — Choose an Exam to Start (Employee)

1. Employee logs in and navigates to the Employee Portal.
2. Employee sees their assigned exam cards. Each card shows:
   - Exam title and description.
   - Status: Not started / In progress / Passed / Failed / Expired.
   - Time limit and attempt counter (attempts used / max allowed).
   - Deadline countdown (if set).
3. Employee clicks "Start exam" on a Not Started exam.
4. A confirmation modal appears showing: time limit, passing score, tab-switch policy, and "You will not be able to leave this page".
5. Employee confirms.
6. **Expected outcome**: A session is created. The employee enters the exam-taking screen.

### Step 2 — Session Initialisation (System)

At the moment the employee confirms:

1. The system verifies:
   - The exam is Active and within its availability window.
   - The employee has attempts remaining.
   - No other in-progress session for this exam exists for this employee.
2. The system resolves questions from the exam's rules:
   - Manual rules: uses the exact pre-selected questions.
   - Random rules: draws the required number of matching active questions using a seeded random selection (seed stored for reproducibility).
3. The system records the question snapshot in `session_questions`.
4. The system sets the session timer (`expires_at = now + time_limit`).
5. The system returns the question list (stems and options only — no correct flags) and the remaining seconds.
6. **Expected outcome**: The employee sees the first question. The server-side countdown has started.

### Step 3 — Answer Questions (Employee)

1. Employee reads each question and selects their answer(s):
   - Single Choice: selects one radio option.
   - Multiple Choice: selects one or more checkboxes.
   - True/False: selects True or False.
   - Likert Scale: selects a scale value.
   - Short Text: types a free-text answer.
2. On each answer selection, the system immediately saves the answer (auto-save).
3. The system returns the updated remaining seconds on every save, keeping the client clock server-authoritative.
4. Employee can navigate freely between questions using the question navigator panel.
5. Employee can flag any question for review and return to it later.
6. The auto-save indicator shows "Saved" / "Saving…" / "Connection lost".
7. **Expected outcome**: Every answered question is safely saved. The employee can resume from any question without losing answers.

### Step 4 — Resume an Interrupted Session (Employee)

1. Employee closes the browser or loses connection mid-session.
2. Employee returns to the portal before the session timer expires.
3. The in-progress session appears with a "Continue" button.
4. Employee clicks "Continue".
5. The system returns all previously saved answers and the remaining seconds.
6. **Expected outcome**: The employee resumes exactly where they left off without data loss.

### Step 5 — Handle Integrity Events (System)

If the employee switches tabs, moves the window to the background, or exits full-screen:

- **Log only**: The event is recorded silently. The employee is unaware.
- **Warn**: The employee sees a warning modal. The event is recorded. The session continues.
- **Auto-submit**: The session is immediately submitted with all answers saved up to that point. The employee cannot continue.

The behaviour is determined by the exam's `on_tab_switch` setting. The employee was informed of this policy in Step 1.

### Step 6 — Submit the Exam (Employee)

1. Employee clicks "Finish exam".
2. A review screen lists: unanswered questions, flagged questions.
3. Employee can return to any question from this screen to answer or unflag it.
4. Employee clicks "Confirm and submit".
5. The system marks the session `submitted` and triggers grading.
6. **Expected outcome**: The exam is submitted. The employee is redirected to the result screen.

### Step 7 — Auto-Submit Expired Sessions (System)

1. A background job runs every 60 seconds.
2. It finds all sessions where `status = in_progress` AND `expires_at < now`.
3. For each expired session: it marks the session `auto_submitted` and triggers grading with the answers saved up to that point.
4. The audit log records each auto-submission.
5. **Expected outcome**: No session stays open beyond its time limit; partial answers are graded fairly.

### Step 8 — Grading (System)

Triggered immediately after submission (manual or auto):

1. **Single / True-False**: 1 point for correct, 0 for incorrect.
2. **Multiple Choice**: Full points if selected options exactly match the correct set. Partial credit (if enabled): `max(0, correct_selected − incorrect_selected) / total_correct`.
3. **Likert**: Score normalised to 0–100 across all Likert questions in the session.
4. **Short Text**: Marked `pending_manual_grade`; contributes 0 to the score until manually graded. Session status becomes `grading_pending` if any short-text answers are present.
5. Total score = `sum(question_scores) / sum(max_possible_scores) × 100`.
6. `passed` flag is set based on the exam's passing score threshold.
7. **Expected outcome**: The session has a final score (or `grading_pending`) and a pass/fail result.

### Step 9 — Adaptive Difficulty (Optional, Phase 7)

If the exam has `adaptive = true`:

- After each answer is saved, the system adjusts the difficulty of the next random-rule question based on the running correct rate.
- The adjustment is invisible to the employee.
- **Expected outcome**: High-performing employees face progressively harder questions; lower performers face progressively easier ones, providing a more precise ability estimate.

## Business Rules

- Only one in-progress session per (employee, exam) pair is permitted at any time.
- A session that has exceeded `expires_at` cannot accept new answers; any save attempt after expiry is rejected.
- An employee who has exhausted `max_attempts` cannot start a new session.
- Question order and option order are fixed at session creation time by the seed; they do not change if the employee navigates back.
- An employee's answered question is saved immediately; no answers are lost if the browser closes unexpectedly.
- The server-side timer is authoritative; client-side timers are display only.
- A grading_pending session is not counted as "passed" or "failed" until manual grading is complete and the final score is calculated.
- Auto-submission uses exactly the answers saved up to the moment the timer expired; no additional answers are accepted.
- Tab-switch events are always logged, regardless of the `on_tab_switch` setting.

## Acceptance Criteria (business language)

1. An employee can start an assigned exam and see all questions in the session.
2. Selecting an answer saves it automatically; the employee sees a "Saved" indicator within 2 seconds.
3. Closing and reopening the browser during an active session restores all previously saved answers and the correct remaining time.
4. When the timer reaches zero, the system submits the session automatically and the employee is redirected to the result screen.
5. Switching tabs on an exam with `on_tab_switch = auto-submit` immediately ends the session; the employee cannot continue.
6. After submitting an exam with only objective questions, the employee immediately sees their score and pass/fail result.
7. An exam that includes short-text questions shows "Grading in progress" on the result screen until manual grading is complete.
8. An employee who has used all allowed attempts sees "Attempts exhausted" on the exam card; the Start button is absent.

## Out of Scope

- Live screen recording or webcam monitoring during the exam.
- Real-time proctor intervention.
- Granting individual time extensions per session.

---
slug: exam-configuration
title: "Exam Configuration and Publishing"
type: process-description
status: uat-verified
created: 2026-06-09
related_requirements: [FR-BB31, FR-BB32, FR-BB312, FR-BB315, FR-BB318]
---

## Business Goal

The organisation needs to assemble approved questions into a named, configured assessment that reflects its specific evaluation goals — including time limits, passing thresholds, question selection rules, anti-cheat settings, and whether to issue a certificate on success. This process covers the full lifecycle of an exam from initial draft through publication. Success means a published exam reliably presents the right questions to the right employees under consistent conditions, and the examiner can be confident that enough active questions exist to satisfy the exam's rules before it goes live.

## Actors

| Actor | Role |
|-------|------|
| Examiner | Creates and configures exams, manages question rules, publishes and archives exams |
| Super Admin | Full access; can also archive or unpublish any exam |
| Department Admin | Can view exam configurations assigned to their department |

## Process Steps

### Step 1 — Create an Exam (Examiner)

1. Examiner navigates to Exams → New Exam.
2. Examiner fills in basic settings:
   - Title and description.
   - Time limit (minutes). Zero means no time limit.
   - Passing score threshold (percentage).
   - Maximum number of attempts. Zero means unlimited.
   - Availability window: optional start and end dates.
   - Option to shuffle question order for each session.
   - Option to shuffle answer option order for each session.
   - Show-answers policy: Never / After completion / After all attempts exhausted.
   - Tab-switch behaviour: Log only / Warn employee / Auto-submit session.
   - Certificate enabled: Yes / No.
3. Examiner saves the exam as Draft.
4. **Expected outcome**: A draft exam record exists with all basic settings saved.

### Step 2 — Define Question Rules (Examiner)

1. Examiner opens the exam in step 2 of the configuration wizard (Question Rules).
2. Examiner adds one or more question rules. Each rule specifies how questions are selected for a session:
   - **Manual rule**: Examiner picks specific questions from the question bank. The exam always uses exactly those questions.
   - **Random rule**: Examiner specifies a category, optional difficulty filter, optional tag filter, and a count. At session start, the system randomly picks that many matching active questions.
3. Examiner can add multiple rules to build a mixed exam (e.g. "5 easy Security questions randomly + 3 specific compliance questions manually").
4. The interface shows a live count of currently eligible questions per rule.
5. If a rule cannot be satisfied (e.g. "10 hard questions" but only 4 exist), a warning is shown immediately.
6. **Expected outcome**: Each rule is saved; the question count indicator is green for all rules.

### Step 3 — Review and Publish (Examiner)

1. Examiner proceeds to the Review step.
2. The system shows a summary: total question count, time limit, passing score, attempt limit, assignment target.
3. Examiner clicks "Publish".
4. The system runs a final validation:
   - All random rules have enough eligible active questions.
   - No rules are empty or contradictory.
5. If validation passes: exam status changes to Active. It can now be assigned to employees.
6. If validation fails: examiner sees the specific failing rules and must resolve them before publishing.
7. **Expected outcome**: The exam is Active and assignable. Employees can start sessions as soon as an assignment is created.

### Step 4 — Edit a Draft Exam (Examiner)

1. Examiner opens a Draft exam.
2. Examiner changes any settings or rules.
3. Examiner saves.
4. **Expected outcome**: Changes are saved; exam remains in Draft.
5. **Constraint**: Active exams cannot be edited (editing requires unpublishing first).

### Step 5 — Unpublish an Active Exam (Examiner / Super Admin)

1. Examiner navigates to the Active exam.
2. Examiner clicks "Unpublish".
3. The system returns the exam to Draft status.
4. Employees with in-progress sessions for this exam can continue to completion. No new sessions can be started.
5. Examiner can now edit the exam (Steps 1–2) and re-publish.
6. **Expected outcome**: The exam is back in Draft; no new sessions can start; active sessions are unaffected.

### Step 6 — Archive an Exam (Examiner / Super Admin)

1. Examiner navigates to an Active exam.
2. Examiner clicks "Archive".
3. The exam status becomes Archived. No new sessions can start and no assignments can be created.
4. Historical sessions, results, and certificates are permanently retained.
5. **Expected outcome**: The exam is retired; historical data is intact.

## Business Rules

- An exam cannot be published if any random rule has fewer eligible active questions than the count requested.
- An exam can only be edited while in Draft status. Editing an Active exam requires unpublishing it first.
- Archiving an exam does not delete any historical sessions, results, or certificates.
- Unpublishing stops new sessions but never interrupts in-progress ones.
- If `max_attempts` is set, employees cannot start more sessions than that limit for this exam.
- If `available_from` / `available_until` is set, sessions cannot start outside that window even if an assignment exists.
- `on_tab_switch = auto-submit` overrides any other submission setting; the session is submitted the instant a tab switch is detected.
- Certificate generation is only possible on sessions where `certificate_enabled = true` on the exam AND the employee passed.
- Shuffle settings are applied independently per session at session creation time; they cannot be changed after a session starts.

## Acceptance Criteria (business language)

1. An examiner can create a draft exam with a title, 30-minute time limit, 70% passing score, and 2 maximum attempts.
2. A random rule specifying "5 hard Security questions" shows a warning if fewer than 5 hard active Security questions exist in the bank.
3. Clicking Publish with all valid rules changes the exam status to Active and the exam appears in the assignment interface.
4. Clicking Publish with an unsatisfied rule shows a specific error message naming the failing rule; the exam remains in Draft.
5. An Active exam cannot be edited; the edit controls are disabled and a message explains that unpublishing is required first.
6. Unpublishing an Active exam returns it to Draft without affecting any in-progress sessions.
7. Archiving an exam removes it from the list of assignable exams but its historical session data remains accessible.

## Out of Scope

- Per-question point values or custom scoring formulas (scoring uses the grading engine's fixed rules).
- Section-level time limits (time limit applies to the whole exam only).
- Proctoring or identity verification during exam taking.

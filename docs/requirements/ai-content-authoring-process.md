---
slug: ai-content-authoring
title: "AI-Assisted Content Authoring"
type: process-description
status: draft
created: 2026-06-09
related_requirements: [FR-BB71, FR-BB72, FR-BB73, FR-BB74, FR-BB75]
---

## Business Goal

Building a large, high-quality question bank manually is time-consuming. The organisation needs to accelerate content creation and improve assessment quality by leveraging AI assistance at four points: generating draft questions, adapting exam difficulty dynamically, automatically grading short-text answers, and surfacing insight summaries from analytics data. This process covers how examiners and admins interact with the AI layer while retaining full human control over all content and grades. Success means AI assistance measurably reduces authoring effort and improves assessment insight, without compromising the integrity of any question, grade, or result.

## Actors

| Actor | Role |
|-------|------|
| Examiner | Requests AI question generation; reviews and accepts/rejects AI draft questions |
| Super Admin | Full access to all AI features; manages AI usage visibility |
| Department Admin | Can view AI-generated loyalty narratives for their employees |
| AI Service (Anthropic Claude) | Generates draft questions, grading suggestions, insight summaries, and loyalty narratives |

## Process Steps

### Step 1 — Generate Draft Questions (Examiner)

1. Examiner navigates to Questions → Generate with AI.
2. Examiner selects:
   - Category (from the existing category tree).
   - Difficulty level (Easy / Medium / Hard).
   - Number of questions to generate (1–10).
   - Optional: a context passage (text excerpt from policy, regulation, or training material) to anchor the questions.
3. Examiner clicks "Generate".
4. The system calls the Anthropic API with a structured prompt; the response is a list of draft questions in question import JSON format.
5. The generated questions are immediately inserted into the question bank as Drafts.
6. The system logs the usage to `ai_usage_log` (user, feature, tokens used, timestamp).
7. Examiner navigates to the question bank filtered by "Draft" and "AI-generated" to review the results.
8. Examiner reviews each question: edits the stem, adjusts options, verifies the correct answer, and changes the category or difficulty if needed.
9. Examiner saves the reviewed question and advances it to the standard review workflow (submit for review → approve → active).
10. **Expected outcome**: Examiners can rapidly populate the question bank with draft content; no AI-generated question reaches Active status without human review.

### Step 2 — Adaptive Difficulty Session (Employee, Phase 7)

Applies only to exams with `adaptive = true`:

1. The exam session starts normally (see Employee Exam Taking process, Step 2).
2. After the employee answers each question, the system recalculates the employee's running correct rate.
3. The next question's difficulty is adjusted based on this rate:
   - High correct rate → next question is harder.
   - Low correct rate → next question is easier.
   - The algorithm uses a 3PL IRT approximation.
4. The adjustment is invisible to the employee; they simply see the next question.
5. At session creation time the system validates that the question bank has sufficient questions at each difficulty level to support adaptive selection.
6. **Expected outcome**: Adaptive exams produce a more precise ability estimate by tailoring difficulty to each individual employee.

### Step 3 — AI Short-Text Auto-Grading (System / Examiner, Phase 7)

Applies only to short-text questions with `auto_grade = true` and a model answer set:

1. When the grading engine processes a submitted session, it identifies auto-gradable short-text questions.
2. The system calls the Anthropic API with the employee's answer and the model answer.
3. The AI returns a score (0–100) with a brief rationale.
4. The score is pre-filled in the manual grading interface, labelled "AI-suggested".
5. The session is NOT removed from the manual grading queue; an examiner must still review and confirm (or override) the AI score.
6. If the AI service is unavailable: the question falls back to a fully manual grade with no pre-fill.
7. **Expected outcome**: Examiners complete manual grading faster; the AI suggestion is a starting point, not a final decision.

### Step 4 — Performance Insight Summary (Admin, Phase 7)

1. Admin opens the per-exam analytics page.
2. Admin clicks "Generate AI insights".
3. The system collects anonymised aggregate statistics for this exam: pass rate, average score, per-question correct rates, answer distributions.
4. The system calls the Anthropic API with these statistics.
5. The AI returns 3–5 natural-language bullet observations about the exam's performance (e.g. "Question 12 has a 19% correct rate — the stem may contain ambiguous wording").
6. The insights are displayed on the analytics page and cached for 24 hours.
7. Admin can click "Regenerate" to force a fresh analysis after cache expiry.
8. **Expected outcome**: Admins receive actionable, plain-language recommendations without needing to interpret raw statistics manually.

### Step 5 — Loyalty Profile Narrative (Admin, Phase 7)

Applies only to exams in the loyalty/values assessment category:

1. Admin opens an employee's session result on the Employee Record page.
2. Admin clicks "Generate loyalty narrative".
3. The system collects the employee's Likert responses for this session.
4. The system calls the Anthropic API with the Likert data, category labels, and option polarity settings.
5. The AI returns a paragraph-length narrative describing the employee's values profile (e.g. emphasis areas, relative strengths).
6. The narrative is displayed on the page, clearly labelled "AI-generated summary — not a clinical assessment".
7. The narrative is visible only to Department Admin and Super Admin; employees cannot see it.
8. **Expected outcome**: Department admins gain a qualitative, human-readable interpretation of loyalty assessment results alongside the numeric score.

## Business Rules

- AI-generated questions are always inserted as Draft; no AI content can bypass the review workflow.
- The AI service is a dependency; if it is unavailable, all AI features degrade gracefully to their non-AI fallback (manual authoring, manual grading, no insight summary).
- AI usage is logged per user and per feature for cost attribution.
- Auto-grading suggestions are pre-fills only; a human examiner must submit a confirmed grade before the session leaves `grading_pending` status.
- AI-generated loyalty narratives are labelled as AI-generated in the UI at all times; they are never presented as objective assessments.
- Adaptive difficulty requires a minimum question bank depth per difficulty tier; this is validated at exam publish time and the exam cannot be published if the requirement is not met.
- All data sent to the Anthropic API is anonymised (no employee names, IDs, or personally identifiable information in prompts).
- Loyalty narratives and insight summaries are not stored permanently; they are regenerated on demand (insight summaries are cached for 24 hours only).

## Acceptance Criteria (business language)

1. An examiner can select a category and difficulty, click "Generate", and see draft questions appear in the question bank within 30 seconds.
2. AI-generated questions have Draft status and cannot be published or assigned until a human approves them.
3. On an adaptive exam, the examiner can see "Adaptive" labelled on the exam card; employees experience the same exam interface as a non-adaptive exam (the adaptation is invisible to them).
4. An exam with adaptive difficulty set to true cannot be published if the question bank does not have at least the required minimum number of questions per difficulty level.
5. In the manual grading interface, a short-text question with `auto_grade = true` shows an "AI-suggested" score pre-filled; the examiner can change the score before submitting.
6. The per-exam analytics page shows an AI insight summary after clicking "Generate AI insights"; the summary contains at least one actionable observation about question quality (Phase 7).
7. A loyalty narrative is visible to a Department Admin on the employee session result page; it is not visible to the employee on their own result screen (Phase 7).

## Out of Scope

- AI moderation of employee text answers for content policy violations.
- AI-generated exam configurations or assignment recommendations.
- Using AI to automatically activate or archive questions without human review.
- Training or fine-tuning AI models on the organisation's question data.

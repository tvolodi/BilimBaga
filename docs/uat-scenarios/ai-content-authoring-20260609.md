---
slug: ai-content-authoring
title: "AI-Assisted Content Authoring — UAT Scenario"
feature: ai-content-authoring (FR-BB71, FR-BB72, FR-BB73, FR-BB74, FR-BB75)
version: 1
created: 2026-06-09
author: Business Analyst
---

Target: local | qa (default: local; never the production-class demo instance, see DEC-001)

## Preconditions

- The platform is running at `http://localhost:5173`.
- Super Admin / Examiner: `admin@test.com` / `Admin1234!`.
- The AI service (Anthropic API) is configured and reachable from the backend.
- Category "UAT Security" exists in the question bank.
- At least one completed session exists for "UAT Security Assessment" with enough data for analytics.

> **Note**: Steps marked **(Phase 7)** test AI features that depend on the Anthropic API. If the API is not configured, these steps will show a graceful fallback (error message or "AI unavailable" indicator) — record the observed behaviour and mark accordingly.

---

## Scenario 1: AI Question Generation — Draft Questions Created

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Examiner | Log in as `admin@test.com` / `Admin1234!` | Admin shell visible | |
| 2 | Examiner | Navigate to Questions → "Generate with AI" (or AI generation button) | AI question generation form is visible | |
| 3 | Examiner | Select category "UAT Security", difficulty "Easy", count `3` | Fields populated | |
| 4 | Examiner | Click "Generate" | Loading indicator appears; after completion, a success message or the question bank is updated | |
| 5 | Examiner | Navigate to Questions, filter by status "Draft" | At least 3 new draft questions appear, labelled as AI-generated (or recently added) | |
| 6 | Examiner | Open one of the generated questions | Question has a stem, answer options, and at least one correct option marked | |
| 7 | Examiner | Assert: question status is "Draft" — not "Active" | Status shows "Draft" | |
| 8 | Examiner | Assert: there is no direct "Publish to Active" shortcut; the question must go through Submit for Review → Approve | Workflow buttons show "Submit for Review" only | |

---

## Scenario 2: AI Question Generation — Graceful Fallback if API Unavailable

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Examiner | Attempt to generate questions as in Scenario 1, Step 3–4, but when AI service is not configured or returns an error | Error message or "AI service unavailable" notification is shown to the examiner | |
| 2 | Examiner | Assert: no invalid draft questions are created (no half-formed records in the bank) | Question bank is unchanged from before the failed generation | |

---

## Scenario 3: Per-Exam AI Insight Summary **(Phase 7)**

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Super Admin | Navigate to Exams → "UAT Security Assessment" → Analytics tab | Per-exam analytics page is visible | |
| 2 | Super Admin | Click "Generate AI insights" | Loading indicator appears | |
| 3 | Super Admin | Assert: after loading, 3–5 bullet-point observations are displayed on the page | AI insight bullets are visible | |
| 4 | Super Admin | Assert: each observation is a plain-language sentence referencing the exam data (e.g. question quality, pass rate) | Observations are readable and relevant | |
| 5 | Super Admin | Navigate away and return to the analytics page | AI insights are still shown (cached for 24 hours) | |
| 6 | Super Admin | Click "Regenerate" | Loading indicator appears; new insight text is shown (may or may not differ from previous) | |

---

## Scenario 4: Adaptive Exam Cannot Publish Without Sufficient Question Bank **(Phase 7)**

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Super Admin | Create a new exam "UAT Adaptive Exam", enable "Adaptive difficulty" toggle | Adaptive flag is set | |
| 2 | Super Admin | Add a random rule with a category that has fewer questions than required per difficulty tier | Rule is saved | |
| 3 | Super Admin | Attempt to publish | Error message is shown stating the question bank does not have sufficient questions per difficulty level for adaptive mode; exam remains Draft | |

---

## Scenario 5: Loyalty Profile Narrative Visible to Admin, Hidden from Employee **(Phase 7)**

*Requires: a completed session on a loyalty/values assessment exam. If no such exam exists, skip this scenario and note it as N/A.*

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Super Admin | Navigate to Users → UAT Employee → View record | Employee Record page visible | |
| 2 | Super Admin | Find the loyalty/values exam session and click to open its detail | Session detail visible | |
| 3 | Super Admin | Click "Generate loyalty narrative" | Loading indicator; after completion, a paragraph-length narrative appears labelled "AI-generated summary — not a clinical assessment" | |
| 4 | Employee | Log in and navigate to "My Results" for the same loyalty session | Result screen is visible | |
| 5 | Employee | Assert: the AI loyalty narrative is NOT visible on the employee's result screen | No loyalty narrative section is shown to the employee | |

---

## Acceptance Criteria Coverage

| AC# | Criterion | Covered by Scenario |
|-----|-----------|---------------------|
| 1 | Examiner selects category/difficulty, clicks Generate, sees draft questions within 30s | Scenario 1, Steps 3–5 |
| 2 | AI-generated questions have Draft status; cannot be published without human approval | Scenario 1, Steps 7–8 |
| 3 | Adaptive exam labelled "Adaptive" on exam card; employees see same interface | Scenario 4, Step 1 (label check); employee experience not separately tested |
| 4 | Adaptive exam cannot publish without sufficient questions per difficulty level | Scenario 4, Steps 2–3 |
| 5 | Short-text question with auto_grade shows "AI-suggested" score pre-filled in grading UI | Not covered (requires short-text question with auto_grade=true; tested in Manual Grading extended run) |
| 6 | Per-exam analytics shows AI insight summary after clicking "Generate AI insights" | Scenario 3, Steps 1–4 |
| 7 | Loyalty narrative visible to Department Admin; not visible to employee | Scenario 5, Steps 3–5 |

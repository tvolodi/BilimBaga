---
slug: question-authoring
title: "Question Authoring"
type: process-description
status: draft
created: 2026-06-09
related_requirements: [FR-BB21, FR-BB22, FR-BB23, FR-BB24, FR-BB25, FR-BB26, FR-BB27, FR-BB28, FR-BB29, FR-BB71]
---

## Business Goal

The organisation needs a curated, multilingual bank of high-quality questions across its assessment tracks (security, safety, loyalty/values) so that exams can be built from validated content. This process covers everything from initial question drafting through review, translation, and activation. Success means the question bank contains enough active, correctly translated questions in the required difficulty distribution to satisfy all configured exam rules.

## Actors

| Actor | Role |
|-------|------|
| Examiner | Authors new questions, manages translations, advances questions through the review workflow, bulk imports content |
| Department Admin | Reviews and approves questions; may author questions in some configurations |
| Super Admin | Manages categories and tags; has full access to all question operations |
| AI Assistant | Generates draft question suggestions on examiner request (Phase 7) |

## Process Steps

### Step 1 — Organise the Content Structure (Super Admin / Examiner)

1. Admin navigates to Categories in the sidebar.
2. Admin creates the category tree: top-level tracks (Security, Safety, Loyalty/Values) with sub-categories as needed.
3. Admin navigates to Tags and creates any tags needed for cross-cutting classification (e.g. "GDPR", "fire-safety", "leadership").
4. **Expected outcome**: A category tree and tag vocabulary exist for question classification.

### Step 2 — Author a Question (Examiner)

1. Examiner navigates to Questions → New Question.
2. Examiner selects the question type: Single Choice, Multiple Choice, True/False, Likert Scale, or Short Text.
3. Examiner fills in the question stem in the default locale.
4. Examiner adds answer options:
   - For choice types: adds at least 2 options and marks the correct one(s).
   - For Likert: adds options with weights and polarities.
   - For Short Text: no options required; optionally provides a model answer for AI grading.
5. Examiner selects difficulty level (Easy / Medium / Hard).
6. Examiner assigns the question to a category and adds relevant tags.
7. Examiner optionally adds an explanation (shown to employees after the exam if allowed).
8. Examiner saves the question as Draft.
9. **Expected outcome**: A new draft question exists in the bank with all required fields populated.

### Step 3 — Add Translations (Examiner)

1. Examiner opens the question in the editor.
2. Examiner clicks the language tab for each additional configured locale (e.g. Russian, Kazakh).
3. Examiner enters the translated stem and all translated answer option texts.
4. Examiner saves each locale.
5. The locale coverage indicator on the question list shows ✓ for each completed locale.
6. **Expected outcome**: The question is fully translated into all organisation-configured languages.

### Step 4 — Submit for Review (Examiner)

1. Examiner opens the question in Draft status.
2. Examiner clicks "Submit for Review".
3. The question status changes to "In Review".
4. **Expected outcome**: The question is visible in the review queue for the approver.

### Step 5 — Review and Approve (Department Admin / Super Admin)

1. Reviewer navigates to the question bank and filters by status "In Review".
2. Reviewer opens each question, reads the stem, checks options, and verifies the correct answer.
3. **If approved**: Reviewer clicks "Approve". Status changes to "Active". The question is now available for exam rule selection.
4. **If rejected**: Reviewer returns the question to Draft (with a note in the comment field if supported) for revision.
5. **Expected outcome**: Only quality-verified questions reach Active status and become eligible for exam selection.

### Step 6 — Edit an Active Question (Examiner)

1. Examiner opens an Active question.
2. Examiner makes corrections (e.g. fixes a typo, adjusts weights).
3. The system creates a new version of the question and archives the old one.
4. The new version enters Draft status and must pass through Steps 4–5 again.
5. **Expected outcome**: The bank always contains the authoritative current version; historical versions are preserved for session audit purposes.

### Step 7 — Bulk Import (Examiner)

1. Examiner prepares a CSV file following the template (type, difficulty, category path, locale, stem, options, correct indices, tags).
2. Examiner navigates to Questions → Import and uploads the file.
3. The system shows a preview with any validation errors highlighted per row.
4. Examiner corrects the source file if errors exist and re-uploads.
5. Examiner confirms the import.
6. The system creates all valid questions as Drafts.
7. **Expected outcome**: Large batches of questions can be loaded without manual form entry; all imported questions start in Draft and require review.

### Step 8 — AI-Assisted Generation (Examiner, Phase 7)

1. Examiner navigates to Questions → Generate with AI.
2. Examiner selects a category, difficulty, and optionally pastes a context passage.
3. Examiner specifies the number of questions to generate.
4. The system calls the AI and returns draft question suggestions.
5. Examiner reviews each suggestion, edits if needed, and saves accepted questions as Drafts.
6. AI-generated questions enter the same review workflow (Steps 4–5) before becoming Active.
7. **Expected outcome**: Examiners can accelerate initial content creation with AI suggestions; no AI-generated content reaches Active status without human review.

### Step 9 — Archive Obsolete Questions (Examiner)

1. Examiner identifies questions that are no longer relevant (outdated regulation, retired topic).
2. Examiner clicks "Archive" on the question.
3. The question status changes to Archived and it is excluded from all future exam rule selections.
4. Existing sessions that already used this question version are unaffected.
5. **Expected outcome**: The active question bank remains current and relevant.

## Business Rules

- A question cannot become Active without at least one complete translation in the default locale.
- For choice questions, at least one answer option must be marked correct.
- For multiple-choice questions, at least two options must be marked correct (otherwise it is functionally a single-choice question).
- Editing an Active question always creates a new version; the original version is archived and must not be modified.
- A question cannot be hard-deleted if it has already been used in any session (even a completed one).
- Draft questions are the only ones eligible for hard deletion.
- Duplicate detection warns (but does not block) if a new question stem is > 80% similar to an existing active question.
- Category and tag deletion is blocked if any active questions reference them.
- Bulk import validation runs on all rows before committing any; a partial import is not allowed.

## Acceptance Criteria (business language)

1. An examiner can create a Single Choice question with a stem, four options, and one correct answer, save it as Draft, and see it in the question list filtered to "Draft" status.
2. An examiner can add a Russian translation to a question and the locale coverage column shows ✓ for Russian.
3. An examiner can submit a Draft question for review; after submission the status badge shows "In Review".
4. An approver can change a question from "In Review" to "Active" with one click; the question then appears in exam rule candidate lists.
5. Editing an Active question creates a new Draft version; the original remains visible in the question's version history.
6. A bulk CSV upload with two invalid rows and ten valid rows shows two error rows in the preview; after correction and re-upload all ten valid rows become Draft questions.
7. An examiner can filter the question bank by category, difficulty, type, status, and locale coverage simultaneously.
8. Archiving a question removes it from the exam rule candidate list but does not affect any in-progress or completed sessions.

## Out of Scope

- Employee-authored questions or peer review by non-admin roles.
- Question scoring configuration at the bank level (scoring rules are part of the Exam Configuration process).
- Storing media attachments (images, audio) in question stems.

---
slug: question-authoring
title: "Question Authoring — UAT Scenario"
feature: question-authoring (FR-BB21, FR-BB22, FR-BB23, FR-BB24, FR-BB25, FR-BB26, FR-BB27, FR-BB28, FR-BB29)
version: 1
created: 2026-06-09
author: Business Analyst
---

Target: local | qa (default: local; never the production-class demo instance, see DEC-001)

## Preconditions

- The platform is running at `http://localhost:5173`.
- A Super Admin account exists: email `admin@test.com`, password `Admin1234!`.
- An Examiner account exists: email `examiner@test.com`, password `Examiner1234!`. If it does not exist, create it via admin before starting.
- A category named "UAT Security" exists (or will be created in Scenario 1).
- A tag named "uat-tag" exists (or will be created in Scenario 1).

---

## Scenario 1: Create Category and Tag

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Super Admin | Log in as `admin@test.com` / `Admin1234!` | Admin Dashboard visible | |
| 2 | Super Admin | Click "Categories" in the sidebar | Categories page is visible | |
| 3 | Super Admin | Click "Add category", fill name `UAT Security`, save | "UAT Security" appears in the category list or tree | |
| 4 | Super Admin | Click "Tags" in the sidebar | Tags page is visible | |
| 5 | Super Admin | Click "Add tag", fill name `uat-tag`, save | "uat-tag" appears in the tags list | |

---

## Scenario 2: Author a Single Choice Question, Submit for Review, Approve

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Examiner | Log in as `examiner@test.com` / `Examiner1234!` | Admin shell visible | |
| 2 | Examiner | Navigate to Questions, click "New Question" (or equivalent) | Question editor is visible | |
| 3 | Examiner | Select type "Single Choice" | Answer options section appears with radio-style option inputs | |
| 4 | Examiner | Fill stem with `What is the primary purpose of a firewall?` | Stem field contains the text | |
| 5 | Examiner | Add option 1: `To block unauthorised network traffic` and mark it correct | Option 1 is listed and marked correct | |
| 6 | Examiner | Add option 2: `To speed up internet connections` | Option 2 is listed (not correct) | |
| 7 | Examiner | Add option 3: `To manage user passwords` | Option 3 is listed (not correct) | |
| 8 | Examiner | Add option 4: `To compress data` | Option 4 is listed (not correct) | |
| 9 | Examiner | Select difficulty "Medium" | Difficulty shows "Medium" | |
| 10 | Examiner | Select category "UAT Security" | Category field shows "UAT Security" | |
| 11 | Examiner | Add tag `uat-tag` | Tag is shown on the question | |
| 12 | Examiner | Click "Save as Draft" | Success; question appears in the question bank with status "Draft" | |
| 13 | Examiner | In the question bank, filter by status "Draft" | The new question is visible in the filtered list | |
| 14 | Examiner | Open the question, click "Submit for Review" | Question status changes to "In Review"; confirmation shown | |
| 15 | Super Admin | Log in as `admin@test.com` / `Admin1234!`, navigate to Questions, filter by status "In Review" | The question appears in the "In Review" list | |
| 16 | Super Admin | Open the question, click "Approve" | Question status changes to "Active" | |
| 17 | Examiner | Navigate to Questions, filter by status "Active" | The approved question appears in the "Active" list | |

---

## Scenario 3: Add a Translation

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Examiner | Open the Active question from Scenario 2 in the editor | Question editor is visible; language tabs are shown (English + configured locales) | |
| 2 | Examiner | Click the "Russian" (or "KK") language tab | Russian translation form is visible; stem and options fields are empty | |
| 3 | Examiner | Fill Russian stem with `Какова основная цель межсетевого экрана?` | Russian stem field contains the text | |
| 4 | Examiner | Fill Russian option 1 with `Блокировать несанкционированный сетевой трафик` | Option 1 Russian text is filled | |
| 5 | Examiner | Fill Russian options 2–4 with any Russian text | Options 2–4 Russian text are filled | |
| 6 | Examiner | Click "Save" | Success; in the question list the locale coverage column shows ✓ for Russian | |

---

## Scenario 4: Edit an Active Question Creates a New Version

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Examiner | Open the Active question from Scenario 2 | Question editor is visible; status shows "Active" | |
| 2 | Examiner | Change stem to `What is the PRIMARY purpose of a firewall?` and save | System creates a new version; the new version has status "Draft"; the original version is visible in version history | |
| 3 | Examiner | Navigate to Questions, filter by status "Active" | The original question is now "Archived"; the new Draft version is listed separately | |

---

## Scenario 5: Filter Question Bank by Multiple Criteria

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Examiner | Navigate to Questions | Questions list is visible | |
| 2 | Examiner | Apply filter: Category = "UAT Security", Difficulty = "Medium", Type = "Single Choice", Status = "Active" | Only questions matching all four filters are shown | |
| 3 | Examiner | Clear all filters | All questions are shown again | |

---

## Scenario 6: Archive a Question

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Examiner | Ensure at least one Active question exists (use the one from Scenario 2 if re-approved, or create a new one) | Active question is visible | |
| 2 | Examiner | Open the Active question and click "Archive" | Confirmation prompt appears | |
| 3 | Examiner | Confirm archiving | Question status changes to "Archived"; it no longer appears in the "Active" filter | |

---

## Acceptance Criteria Coverage

| AC# | Criterion | Covered by Scenario |
|-----|-----------|---------------------|
| 1 | Examiner creates Single Choice question, saves as Draft, sees it in filtered list | Scenario 2, Steps 1–13 |
| 2 | Examiner adds Russian translation; locale coverage shows ✓ | Scenario 3, Steps 1–6 |
| 3 | Examiner submits Draft for review; status shows "In Review" | Scenario 2, Step 14 |
| 4 | Approver changes "In Review" to "Active" with one click | Scenario 2, Steps 15–16 |
| 5 | Editing Active question creates new Draft; original in version history | Scenario 4, Steps 1–3 |
| 6 | Bulk CSV upload with invalid rows shows errors; valid rows become Drafts | Not covered in this run (requires CSV file preparation) |
| 7 | Filter by category, difficulty, type, status simultaneously | Scenario 5, Steps 1–3 |
| 8 | Archiving removes from Active list; does not affect sessions | Scenario 6, Steps 1–3 |

---
slug: grading-pending-experience
title: "Grading-Pending Result Experience — UAT Scenario"
feature: grading-pending-experience (FR-BB322, amends FR-BB45 AC-8, FR-BB313, FR-BB46, FR-BB61)
version: 1
created: 2026-10-11
author: Developer (issue #545)
---

Target: local | qa (default: local; never the production-class demo instance, see DEC-001)

## Preconditions

- The platform is running at `http://localhost:5173` (frontend) and `http://localhost:8080` (API), with the mail server reachable or its log readable.
- A Super Admin account exists: email `admin@test.com`, password `Admin1234!`.
- An employee account exists and is in a department: email `uat.employee@test.com`, password `NewPass123!`, role Employee.
- Exam A, "Fire safety (grading)": active, assigned to the employee, `max_attempts = 3`, `show_answers = after_completion`, certificate enabled. It has one single-choice question and one short-text question with auto-grade on and a model answer.
- Exam B, "Fire safety (one attempt)": as Exam A but `max_attempts = 1`.
- The AI grading scenario (Scenario 2) makes paid provider calls. Run it only after the owner orders it in the bb-uat session. Scenarios 1, 3 to 7 need no AI provider and make no paid calls.
- Mail: `SMTP` is configured and the mail log (`email_log`) can be read by the UAT runner.

---

## Scenario 1: Examiner grading — the page waits, then flips to the result

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Employee | Log in as `uat.employee@test.com`, open Exam A, click "Start exam" | The exam opens | |
| 2 | Employee | Answer the single-choice question, answer the short-text question with "Use the north stairs", click "Submit" | The submit succeeds and the browser opens the result page for the session | |
| 3 | Employee | Read the result page | The page shows "Your answers are waiting for an examiner. You will get an email when the result is ready." and "1 of 1 answers are still being checked"; there is no score dial and no pass/fail banner; no certificate, retake or "Download" action is shown | |
| 4 | Employee | Read the question review | The short-text question shows "Awaiting grading" in the Points cell, and its correct answer and explanation cells are empty | |
| 5 | Employee | Open the Employee Portal | The Exam A card shows the badge "Awaiting grading" and the line "Submitted <date and time>"; it has no "Start exam" button; the button reads "View status" | |
| 6 | Employee | Click "View status" on the Exam A card | The browser opens the result page of the waiting session | |
| 7 | Employee | Keep the result page open for 1 minute, then leave the tab in the background for 1 minute and come back | The page updates itself while visible; it does not poll while the tab is hidden | |
| 8 | Examiner | Log in as `admin@test.com`, open the grading queue, open the waiting session, grade the short-text question 100% and submit | The grading succeeds | |
| 9 | Employee | Stay on the result page | Within 10 seconds the page shows the normal result: the score dial, the pass/fail banner and the question review with the correct answer; the notice disappears; the text "Your result is ready." is announced by the screen reader | |
| 10 | Employee | Open the Employee Portal | The Exam A card shows "Passed" (or "Failed" per the score) and the "View result" button; the "Submitted" line is gone | |

---

## Scenario 2: AI grading — the notice names the AI, and the result arrives within the job time

**Requires the owner's go** (paid provider calls). Preconditions: an AI provider is configured for the API, `AI_GRADING_DAILY_LIMIT` is greater than 0.

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Employee | Start Exam A and submit it with a short-text answer | The submit succeeds and the result page opens | |
| 2 | Employee | Read the result page | The notice reads "Your answers are being checked automatically. This usually takes a few minutes." and "1 of 1 answers are still being checked" | |
| 3 | Employee | Wait on the result page | Within about 2 minutes (the background job runs every 15 seconds) the normal result appears with the AI's score and reasoning; "Your result is ready." is announced | |
| 4 | Employee | Open the Employee Portal | The card shows the final status, not "Awaiting grading" | |

---

## Scenario 3: Attempt cap — a waiting session blocks a new attempt with max_attempts = 1

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Employee | Start Exam B, submit it with a short-text answer | The result page shows the waiting notice | |
| 2 | Employee | Open the Employee Portal | The Exam B card shows "Awaiting grading", the attempt count is "Attempt 1 of 1", and there is no "Start exam" button | |
| 3 | Employee | Call `POST /api/v1/portal/exams/<Exam B id>/start` with the employee's token (for example from the browser console with `fetch`) | HTTP 409 with the error code `GRADING_PENDING` | |
| 4 | Examiner | Grade the waiting session (as in Scenario 1, step 8) | The session becomes final | |
| 5 | Employee | Call `POST /api/v1/portal/exams/<Exam B id>/start` again | HTTP 422 with the error code `ATTEMPTS_EXHAUSTED` (the attempt is finished and there is no attempt left) | |

---

## Scenario 4: Mail — one submitted mail, and no result mail until the result is final

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Employee | Start Exam A, submit it with a short-text answer | The mail log has one `exam_submitted` entry for the employee, in the employee's preferred locale | |
| 2 | Employee | Open the `exam_submitted` mail | It names the exam, the submit time, and a link `<PUBLIC_APP_URL>/portal/sessions/<session id>/result` | |
| 3 | Employee | Submit Exam A again from the same session (a repeated submit) | No second `exam_submitted` entry appears in the mail log | |
| 4 | Examiner | Grade the waiting session to a final result | One `exam_passed` or `exam_failed` entry appears in the mail log, after the grading; the mail links to `<PUBLIC_APP_URL>/portal/sessions/<session id>/result` and contains no `/certificate` link | |
| 5 | Employee | Repeat the step-1 submit in a second session of Exam A that is left to expire (the expiry job runs at its deadline) | Exactly one `exam_submitted` entry for that session, written by the expiry job | |

---

## Scenario 5: Mail link while signed out — login returns to the result page

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Employee | Sign out | The login page is shown | |
| 2 | Employee | Open the `exam_submitted` link from Scenario 4 in the same browser | The browser opens the login page, not an error | |
| 3 | Employee | Sign in as `uat.employee@test.com` | The browser returns to `/portal/sessions/<session id>/result` (the requested page), not to the Employee Portal home | |

---

## Scenario 6: Results list and exam-level link

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Employee | With a waiting session in Exam A, open "My results" | The waiting session is listed; its score cell reads "Awaiting grading" and it shows no pass or fail badge | |
| 2 | Employee | In the Employee Portal, open the Exam A card's "View status" (while waiting) | The browser opens the waiting session, even if an older attempt of Exam A exists | |
| 3 | Employee | With no waiting session, open the Exam A "View result" button | The browser opens the newest attempt of Exam A | |

---

## Scenario 7: Question marker and the stopped-polling state

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Employee | Open a waiting result page and leave it untouched for 15 minutes | The page shows "We stopped checking automatically. Refresh to see the latest state." and a "Refresh" button; it makes no further requests | |
| 2 | Employee | Click "Refresh" | The page fetches the result once, and if the result is still waiting it polls again for 15 minutes | |

---

## Cleanup

- Delete the test sessions created in Exam A and Exam B, or leave them for the next UAT run if the runner keeps the test data.
- Do not leave `AI_GRADING_DAILY_LIMIT` above 0 on a shared stack after Scenario 2.

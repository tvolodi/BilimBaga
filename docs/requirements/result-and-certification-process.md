---
slug: result-and-certification
title: "Result Review and Certification"
type: process-description
status: uat-verified
created: 2026-06-09
related_requirements: [FR-BB41, FR-BB43, FR-BB44, FR-BB45, FR-BB46]
---

## Business Goal

After an employee completes an exam, both the employee and the organisation need a clear, trustworthy record of the outcome. For passing employees, a verifiable certificate must be issued that can be presented to third parties. This process covers the flow from session submission through result visibility, certificate generation, and ongoing access to the employee's exam history. Success means every passing employee can download a certificate within seconds of their result, every certificate can be publicly verified, and admins can view any employee's full exam record at any time.

## Actors

| Actor | Role |
|-------|------|
| Employee | Views their result, downloads their certificate, reviews their exam history |
| Examiner / Admin | Views results and certificates for any employee; uses data for compliance reporting |
| External Party | Verifies a certificate's authenticity using the public verification URL or QR code |

## Process Steps

### Step 1 — View Immediate Result (Employee)

1. Immediately after the exam is submitted and graded, the employee is redirected to the Result Screen.
2. The Result Screen shows:
   - Score as a percentage dial/display.
   - Pass/Fail banner in brand colours.
   - Time taken.
   - Attempt number.
   - Per-section score breakdown (if the exam has sections).
   - Per-question review table (if the exam's `show_answers` policy allows): question stem, the employee's answer, the correct answer, points earned, and explanation text if available.
3. If the session status is `grading_pending` (short-text questions not yet graded): the screen shows "Grading in progress" and the certificate/retake options are not yet shown.
4. **Expected outcome**: The employee immediately knows whether they passed and can review their performance.

### Step 2 — Download Certificate (Employee, if passed)

1. After a passing result is confirmed, a "Download Certificate" button appears on the Result Screen.
2. Employee clicks the button.
3. The system generates the certificate on first request (lazy generation):
   - Fetches company logo and name from tenant config.
   - Builds the certificate layout: employee full name, exam title, score percentage, issue date, signatory name and title.
   - Generates a unique verification code (UUID).
   - Embeds a QR code pointing to the public verification URL.
   - Renders as a PDF.
4. The PDF is delivered to the browser as a download.
5. The certificate record is saved in the database; subsequent download requests return the same certificate without regenerating it.
6. **Expected outcome**: The employee has a PDF certificate with a unique verification code. Repeat downloads return the same certificate.

### Step 3 — Verify a Certificate (External Party)

1. External party scans the QR code on the certificate or navigates to the verification URL (`/verify/{code}`).
2. The system responds with:
   - Valid: employee name, exam title, issue date. Displayed as a simple confirmation page.
   - Invalid / not found: a clear message that the certificate could not be verified.
3. No authentication is required for this endpoint.
4. **Expected outcome**: Any third party can confirm the certificate's authenticity without contacting the organisation.

### Step 4 — View Exam History (Employee)

1. Employee navigates to the "My Results" tab in the Employee Portal.
2. The tab shows a chronological table of all their past sessions:
   - Exam name, date taken, score, pass/fail badge, certificate download link (for passing sessions where enabled).
3. Employee can sort by date or score.
4. **Expected outcome**: The employee has a complete personal record of all their assessments.

### Step 5 — Admin Views Employee Result (Examiner / Admin)

1. Admin navigates to Users → selects an employee → "View record".
2. The Employee Record page shows a session history table with exam name, date, score, pass/fail, certificate link, and time taken.
3. Admin can download the certificate for any passing session.
4. Admin can also view per-session detail: the full answer breakdown if `show_answers` is enabled.
5. **Expected outcome**: Admins can perform compliance audits and identify employees who need follow-up.

### Step 6 — Retake an Exam (Employee, if attempts remain)

1. On the Result Screen after a failed attempt, if attempts remain, a "Retake exam" button appears.
2. Employee clicks "Retake exam".
3. The system starts a new session (same flow as Employee Exam Taking, Step 1 onward).
4. Previous attempts are visible in the employee's history alongside the new attempt.
5. **Expected outcome**: The employee can try again; each attempt is independently recorded.

## Business Rules

- Certificate generation is only triggered for sessions where: `passed = true` AND `exam.certificate_enabled = true`.
- A certificate is generated once (lazily on first download request) and never regenerated; the verification code never changes.
- The public certificate verification endpoint is available without authentication and must not reveal any data beyond employee name, exam title, and issue date.
- The result screen's per-question review is only shown if `exam.show_answers` is set to `after_completion` or `after_all_attempts` (and the appropriate condition is met).
- If `show_answers = after_all_attempts`, the full answer review is only visible after the employee has used all their allowed attempts.
- For sessions in `grading_pending` status, no pass/fail result, certificate, or retake option is shown until manual grading is complete.
- All certificates include the tenant's logo and branding at the time of generation; branding changes after generation do not affect existing certificates (the template snapshot is stored).
- An admin can access the result and certificate for any session regardless of `show_answers` policy.

## Acceptance Criteria (business language)

1. After submitting an exam with only objective questions, the employee sees their score and pass/fail result within 5 seconds.
2. A passing employee with a certificate-enabled exam sees a "Download Certificate" button on the result screen.
3. The downloaded PDF contains the employee's name, exam title, score, issue date, QR code, and the tenant's logo.
4. Scanning the QR code on the certificate displays the employee name, exam title, and issue date without requiring a login.
5. Downloading the certificate a second time returns the same PDF with the same verification code.
6. The employee's "My Results" tab shows all past sessions sorted by date, with pass/fail badges and certificate links where applicable.
7. An admin can open an employee's record and see all sessions with scores; they can download certificates for any passing session.
8. For an exam with `show_answers = never`, the employee does not see the correct answers on the result screen.

## Out of Scope

- Certificate revocation or invalidation.
- Digital signatures on PDF certificates.
- Sending certificates automatically via email (email notifications are a separate process).
- Printing or embossing physical certificates.

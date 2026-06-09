---
slug: exam-assignment
title: "Exam Assignment"
type: process-description
status: uat-verified
created: 2026-06-09
related_requirements: [FR-BB33, FR-BB34, FR-BB61]
---

## Business Goal

The organisation needs to direct specific exams to the right employees or groups, with optional deadlines, so that compliance and competency requirements are met on schedule. This process covers assigning an active exam to individuals, entire departments, or all employees, and revoking assignments when they are no longer applicable. Success means every employee who should take an exam can see it in their portal before the deadline, and admins have visibility into assignment completion status.

## Actors

| Actor | Role |
|-------|------|
| Super Admin | Can assign any exam to any user, department, or all employees; can remove any assignment |
| Department Admin | Can assign exams to employees within their own department |
| Employee | Receives the exam in their portal; sees deadline and attempt status |

## Process Steps

### Step 1 — Select the Exam and Target (Super Admin / Department Admin)

1. Admin navigates to Exams and opens an Active exam.
2. Admin clicks "Assign".
3. Admin selects the assignment scope:
   - **Individual**: selects a specific employee by name or email.
   - **Department**: selects a department; the assignment covers all current members.
   - **All employees**: assigns to every active user in the system.
4. Admin optionally sets a deadline date and time.
5. Admin confirms the assignment.
6. **Expected outcome**: An assignment record is created. The exam immediately appears in the targeted employees' portals.

### Step 2 — Employee Receives the Exam (Employee)

1. Employee logs in and navigates to the Employee Portal.
2. The assigned exam appears as an exam card with:
   - Status: "Not started".
   - Deadline (if set) with a countdown.
   - Time limit and attempt limit information.
3. **Expected outcome**: The employee can see the exam and all the information they need to decide when to take it.

### Step 3 — Deadline Reminder (Automated)

1. The system background job checks daily at 08:00 (tenant timezone) for assignments with deadlines within 48 hours.
2. A reminder email is sent to each affected employee in their configured locale.
3. The email includes the exam name, deadline, and a direct link to the portal.
4. **Expected outcome**: Employees receive a timely reminder and have a chance to complete the exam before the deadline.

### Step 4 — Monitor Completion (Admin)

1. Admin navigates to Exams → Assignments for the exam.
2. Admin sees a completion table: each assignment row shows assignee name/department, deadline, and status (Not started / In progress / Passed / Failed / Expired).
3. Admin can identify overdue employees and, if needed, extend deadlines by removing and re-creating the assignment with a new deadline.
4. **Expected outcome**: Admin has real-time visibility into who has and has not completed the exam.

### Step 5 — Remove an Assignment (Admin)

1. Admin opens the Assignments view for the exam.
2. Admin clicks "Remove" next to an assignment.
3. The assignment is deleted. The exam disappears from the affected employees' portals.
4. **Note**: Sessions that were already started or completed before the assignment was removed are retained in full.
5. **Expected outcome**: The exam is no longer visible to the employee; historical session data is unaffected.

## Business Rules

- Only Active exams can be assigned. Draft and Archived exams are not assignable.
- The same exam can be assigned to the same employee via multiple overlapping assignment types (e.g. individually and also via department). The employee sees the exam once; the most permissive deadline applies.
- An assignment with no deadline means no expiry; the employee can start the exam at any time while it remains active.
- Removing an assignment does not cancel or delete any sessions already started by the affected employees.
- Department assignments are resolved at session-start time: if an employee joins a department after the assignment was created, they will still see the exam.
- After the deadline passes, the exam card shows "Expired" and the employee cannot start new sessions for that exam via that assignment.
- An employee who has exhausted their maximum attempts cannot start a new session regardless of the assignment being active.

## Acceptance Criteria (business language)

1. Assigning an active exam to a department makes the exam card visible to all employees in that department within their portal on next page load.
2. An assignment with a deadline shows a countdown timer on the employee's exam card.
3. After the deadline passes, the exam card shows "Expired" and the "Start" button is disabled.
4. The assignment completion table shows the correct status (Not started / Passed / Failed / etc.) for each assignee.
5. Removing an assignment removes the exam card from the affected employee's portal; completed sessions remain visible in the employee's history.
6. An employee who has used all allowed attempts sees "Attempts exhausted" on the exam card even if the deadline has not passed.

## Out of Scope

- Automated assignment based on role or onboarding date (assignments are always triggered manually by an admin).
- Prerequisite chaining (exam B can only be assigned after exam A is passed).
- Assignment approval workflows.

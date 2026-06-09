---
slug: user-onboarding
title: "User Onboarding"
type: process-description
status: uat-verified
created: 2026-06-09
related_requirements: [FR-BB14, FR-BB16, FR-BB17, FR-BB18, FR-BB110, FR-BB111]
---

## Business Goal

The organisation needs to enrol employees into the platform so they can receive and complete assigned exams. This process covers everything from the moment an administrator creates a user account to the moment the employee successfully logs in, changes their temporary password, and lands in their personal exam portal. Success means every employee has a working account with the correct role and department assignment, and no employee is active in the system without a valid identity.

## Actors

| Actor | Role |
|-------|------|
| Super Admin | Creates user accounts, manages departments, assigns roles, resets passwords |
| Department Admin | Creates and manages users within their own department |
| Employee | Receives credentials, completes first login, changes temporary password |

## Process Steps

### Step 1 — Prepare the Department Structure (Super Admin)

1. Super Admin logs in to the admin shell.
2. Super Admin navigates to Departments in the sidebar.
3. Super Admin creates the required department tree (root departments first, then child departments).
4. Each department is saved with a unique name.
5. **Expected outcome**: The department tree reflects the organisation's structure and is available for user assignment.

### Step 2 — Create User Accounts (Super Admin or Department Admin)

**Option A — Single user creation**

1. Admin navigates to Users in the sidebar.
2. Admin clicks "Create user".
3. Admin fills in: full name, email address, department (from tree), role.
4. Admin submits the form.
5. The system creates the account with a temporary password and sets `force_password_change = true`.
6. Admin communicates the temporary credentials to the employee (e.g. by email or in person).
7. **Expected outcome**: User account exists, is active, and appears in the user list.

**Option B — Bulk CSV import**

1. Admin navigates to Users → Import.
2. Admin uploads a CSV file with columns: full name, email, department path, role.
3. The system shows a preview table with validation errors highlighted.
4. Admin corrects the source file if errors are shown and re-uploads.
5. Admin confirms the import.
6. The system creates all valid accounts with temporary passwords.
7. **Expected outcome**: All valid rows in the CSV become active user accounts.

### Step 3 — First Login (Employee)

1. Employee navigates to the platform login page.
2. Employee enters email and temporary password.
3. The system authenticates and detects `force_password_change = true`.
4. The system redirects the employee to the Forced Password Change screen.
5. Employee enters the temporary password as "current password" and chooses a new password.
6. Employee submits the form.
7. The system clears `force_password_change`, issues a new session, and redirects to the Employee Portal.
8. **Expected outcome**: Employee is logged in with a personal password and sees their assigned exams.

### Step 4 — Account Maintenance (Super Admin)

- **Deactivate**: Admin deactivates a user who has left the organisation. All historical records are preserved. The user can no longer log in.
- **Password reset**: If an employee forgets their password, Admin resets it. A new temporary password is set and `force_password_change` is set to true again, repeating Step 3.
- **Role / department change**: Admin edits the user's profile to update role or department. Changes take effect on the employee's next authenticated request.

## Business Rules

- Every user must belong to exactly one department at all times.
- Every user must have exactly one role (`super_admin`, `department_admin`, `examiner`, or `employee`).
- A deactivated user cannot log in but their exam history, answers, and certificates are permanently retained.
- Department deletion is blocked while any user is assigned to it.
- After 5 consecutive failed login attempts the account is locked and must be unlocked by an admin.
- Temporary passwords must be changed on first login; no other portal action is available until the change is complete.
- Email addresses are unique across the system; duplicate emails are rejected at creation and import time.
- Bulk import validates all rows before committing any; a partial import is not allowed — either all valid rows are committed together or none.

## Acceptance Criteria (business language)

1. An admin can create a single user and the user appears immediately in the user list with status "Active".
2. An admin can upload a CSV; invalid rows are flagged with a human-readable error before any data is saved; valid rows are committed when the admin confirms.
3. An employee with `force_password_change = true` cannot access any page other than the password change screen after login.
4. After a successful forced password change, the employee lands on the Employee Portal and their exam list is visible.
5. Deactivating a user prevents that user from logging in; all their historical data remains visible to admins.
6. A password reset by an admin results in the employee being required to change their password on next login.
7. An admin can view all users filtered by department, role, or status.
8. Attempting to delete a department with assigned users results in a visible error message, not a silent failure.

## Out of Scope

- Self-registration by employees (accounts are always created by admins).
- Email delivery of temporary passwords (email notifications are a separate process).
- Single Sign-On (SSO) or external identity provider integration.
- Multi-factor authentication.

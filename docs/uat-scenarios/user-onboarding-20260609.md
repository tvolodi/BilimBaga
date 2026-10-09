---
slug: user-onboarding
title: "User Onboarding — UAT Scenario"
feature: user-onboarding (FR-BB14, FR-BB16, FR-BB17, FR-BB18, FR-BB110, FR-BB111)
version: 1
created: 2026-06-09
author: Business Analyst
---

Target: local | qa (default: local; never the production-class demo instance, see DEC-001)

## Preconditions

- The platform is running at `http://localhost:5173` (frontend) and `http://localhost:8080` (API).
- A Super Admin account exists: email `admin@test.com`, password `Admin1234!`.
- No department named "UAT Engineering" exists yet.
- No user with email `uat.employee@test.com` exists yet.

---

## Scenario 1: Create Department and Single User, Employee First Login

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Super Admin | Navigate to `http://localhost:5173` | Login page is visible with company logo and login form | |
| 2 | Super Admin | Fill "Email" with `admin@test.com`, fill "Password" with `Admin1234!`, click "Sign in" | Admin is redirected to the Admin Dashboard; sidebar is visible | |
| 3 | Super Admin | Click "Departments" in the sidebar | Departments page is visible with the department tree | |
| 4 | Super Admin | Click "Add department" (or equivalent create button) | A form or modal appears to enter a department name | |
| 5 | Super Admin | Fill department name with `UAT Engineering`, confirm/save | "UAT Engineering" appears in the department tree | |
| 6 | Super Admin | Click "Users" in the sidebar | Users list page is visible | |
| 7 | Super Admin | Click "Create user" | A user creation form or drawer appears | |
| 8 | Super Admin | Fill "Full name" with `UAT Employee`, fill "Email" with `uat.employee@test.com`, select department "UAT Engineering", select role "Employee" | All fields are populated; form is valid | |
| 9 | Super Admin | Submit the user creation form | Success message is shown; the Users list now contains "UAT Employee" with status "Active" | |
| 10 | Super Admin | Note the temporary password shown (or copy from the response) | Temporary password is visible to the admin | |
| 11 | Super Admin | Click "Logout" | Admin is returned to the login page | |
| 12 | Employee | Fill "Email" with `uat.employee@test.com`, fill "Password" with the temporary password, click "Sign in" | Employee is redirected to the Forced Password Change screen — NOT the portal | |
| 13 | Employee | Attempt to navigate to `http://localhost:5173/portal` directly | Still shows the Forced Password Change screen; navigation is blocked | |
| 14 | Employee | Fill "Current password" with the temporary password, fill "New password" with `NewPass123!`, fill "Confirm password" with `NewPass123!`, click "Save" | Employee is redirected to the Employee Portal; exam card grid is visible (may be empty) | |
| 15 | Employee | Click "Logout" | Employee is returned to the login page | |
| 16 | Employee | Log in with `uat.employee@test.com` and `NewPass123!` | Employee lands directly on the Employee Portal (no forced password change screen) | |

---

## Scenario 2: Deactivate User — Login Blocked

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Super Admin | Log in as `admin@test.com` / `Admin1234!` | Admin Dashboard visible | |
| 2 | Super Admin | Navigate to Users, find "UAT Employee" in the list | Row for UAT Employee is visible with status "Active" | |
| 3 | Super Admin | Open UAT Employee's record and click "Deactivate" (or equivalent) | Confirmation prompt appears | |
| 4 | Super Admin | Confirm deactivation | UAT Employee status changes to "Inactive" in the users list | |
| 5 | Employee | Attempt to log in with `uat.employee@test.com` / `NewPass123!` | Login is rejected; an error message is shown (e.g. "Account is inactive") | |

---

## Scenario 3: Duplicate Email Rejected

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Super Admin | Log in as `admin@test.com` / `Admin1234!` and navigate to Create User | User creation form is visible | |
| 2 | Super Admin | Fill "Full name" with `Duplicate User`, fill "Email" with `uat.employee@test.com`, select any department and role, submit | Error message is shown stating the email is already in use; no new user is created | |

---

## Scenario 4: Admin Filters Users by Department

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Super Admin | Log in as `admin@test.com` / `Admin1234!`, navigate to Users | Users list is visible | |
| 2 | Super Admin | Apply filter: Department = "UAT Engineering" | Only users in "UAT Engineering" are shown in the list | |
| 3 | Super Admin | Clear the filter | All users are shown again | |

---

## Acceptance Criteria Coverage

| AC# | Criterion | Covered by Scenario |
|-----|-----------|---------------------|
| 1 | Admin creates a single user; user appears in list with status "Active" | Scenario 1, Steps 7–9 |
| 2 | CSV import: invalid rows flagged, valid rows committed on confirm | Not covered (CSV import requires a prepared file; tested separately) |
| 3 | `force_password_change` employee cannot access portal pages | Scenario 1, Steps 12–13 |
| 4 | After forced password change, employee lands on portal | Scenario 1, Step 14 |
| 5 | Deactivating prevents login; historical data remains | Scenario 2, Steps 3–5 |
| 6 | Password reset causes employee to change on next login | Not covered in this run (requires admin reset flow) |
| 7 | Admin filters users by department, role, or status | Scenario 4, Steps 2–3 |
| 8 | Deleting a department with users shows an error | Not covered (requires department with assigned user; tested in extended run) |

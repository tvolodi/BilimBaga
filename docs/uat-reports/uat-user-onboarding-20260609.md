---
run_id: uat-user-onboarding-20260609
scenario_path: docs/uat-scenarios/user-onboarding-20260609.md
executed: 2026-06-09T00:00:00Z
executor: UAT Runner
result: PARTIAL (1 step failed — confirmed defect)
---

# UAT Report — User Onboarding

## Summary
- Total scenarios: 4
- Total Playwright test blocks: 5
- Passed: 4
- Failed: 1 (confirmed defect — not a test error)
- Blocked: 0
- Screenshots taken: multiple (in test-results/ directory)

## Precondition Setup

| Precondition | Status | Notes |
|-------------|--------|-------|
| Platform running at localhost:80 (frontend via Nginx) | PASS | Frontend served via Nginx on port 80; port 5173 not directly exposed |
| Backend API at localhost:8080 | PASS | Health check returns `{"status":"ok","db_ok":true}` |
| Super Admin exists | PASS | `admin@bilimbaga.local` / `Admin1234!` — NOTE: scenario document specifies `admin@test.com` which does not exist; actual seed email is `admin@bilimbaga.local` |
| No "UAT Engineering" department | PASS (idempotent) | Department created on first run, verified on subsequent runs |
| No `uat.employee@test.com` | PASS (idempotent) | User created on first run; scenario handles subsequent runs by verifying existing state |

**Precondition discrepancy**: The scenario script specifies admin credentials as `admin@test.com / Admin1234!`. The actual seeded admin is `admin@bilimbaga.local / Admin1234!`. Scenario script should be updated.

---

## Scenario Results

### Scenario 1 — Create Department and Single User, Employee First Login

| Step | Actor | Action | Expected | Actual | Method | Status | Screenshot |
|------|-------|--------|----------|--------|--------|--------|------------|
| 1 | Super Admin | Navigate to `http://localhost:5173` | Login page visible with form | Login page visible at http://localhost/login | Playwright | PASS | none |
| 2 | Super Admin | Fill email `admin@bilimbaga.local`, password `Admin1234!`, click Sign in | Admin Dashboard visible; sidebar visible | Redirected to /admin/dashboard; admin layout rendered | Playwright | PASS | none |
| 3 | Super Admin | Click Departments in sidebar | Departments page with heading | Navigated to /admin/departments; heading "Departments" visible | Playwright | PASS | none |
| 4 | Super Admin | Click "New Department" | Form/modal appears | Dialog opened with dept-name input | Playwright | PASS | none |
| 5 | Super Admin | Fill "UAT Engineering", save | Dept appears in tree | "UAT Engineering" visible in department tree | Playwright | PASS | none |
| 6 | Super Admin | Click Users in sidebar | Users list with table | Table visible at /admin/users | Playwright | PASS | none |
| 7 | Super Admin | Click "New User" | Create user Sheet opens | Sheet heading "Create User" visible | Playwright | PASS | none |
| 8 | Super Admin | Fill Full Name, Email, select Department "UAT Engineering", Role "Employee" | All fields populated | All fields filled via placeholders and combobox/select | Playwright | PASS | none |
| 9 | Super Admin | Submit user creation | Success; user "UAT Employee" in list with status "Active" | User visible in table with "Active" status | Playwright | PASS | none |
| 10 | Super Admin | Note temporary password | Temporary password visible to admin | PasswordResetModal dialog opened after creation with "Temporary Password" label | Playwright | PASS | none |
| 11 | Super Admin | Logout | Returns to login page | Navigated to /login via Sign out button (aria-label="Sign out") | Playwright | PASS | none |

### Scenario 1 (Employee) — Forced Password Change on First Login

| Step | Actor | Action | Expected | Actual | Method | Status | Screenshot |
|------|-------|--------|----------|--------|--------|--------|------------|
| 12 | Employee | Login with temp password | Redirected to Forced Password Change screen, NOT portal | URL after login was not /portal; change password heading visible | Playwright | PASS | none |
| 13 | Employee | Navigate directly to /portal | Still shows Forced Password Change; navigation blocked | Redirect away from /portal confirmed; change password still shown | Playwright | PASS | none |
| 14 | Employee | Fill current password, new password `NewPass123!`, confirm, Save | Redirected to Employee Portal | Redirected to /portal after successful password change | Playwright | PASS | none |
| 15 | Employee | Logout | Returns to login page | **FAIL (DEFECT-1)**: PortalLayout has no Sign Out button. Logout via API call workaround used to continue test. | Playwright | FAIL | test-results/... |
| 16 | Employee | Login with `NewPass123!` | Lands directly on portal (no forced change screen) | Login succeeded; redirected to /portal without change-password screen | Playwright | PASS | none |

---

### Scenario 2 — Deactivate User, Login Blocked

| Step | Actor | Action | Expected | Actual | Method | Status | Screenshot |
|------|-------|--------|----------|--------|--------|--------|------------|
| 1 | Super Admin | Login as admin | Admin Dashboard visible | Login succeeded | Playwright | PASS | none |
| 2 | Super Admin | Navigate to Users, find "UAT Employee" | Row visible with status "Active" | Row found; "Active" badge visible | Playwright | PASS | none |
| 3 | Super Admin | Click "Deactivate" | Confirmation prompt appears | Deactivate button clicked; DeactivateConfirmDialog appeared | Playwright | PASS | none |
| 4 | Super Admin | Confirm deactivation | Status changes to "Inactive" | Status badge changed to "Inactive" in table row | Playwright | PASS | none |
| 5 | Employee | Login with `uat.employee@test.com` / `NewPass123!` | Login rejected; error "Account is inactive" or equivalent | **FAIL (DEFECT-2)**: Login succeeded and employee landed on /portal. Backend does not check `users.status` during authentication. | Playwright | FAIL | test-results/uat-user-onboarding-202606-4c3a7-activate-User-Login-Blocked-chromium-uat/test-failed-1.png |

---

### Scenario 3 — Duplicate Email Rejected

| Step | Actor | Action | Expected | Actual | Method | Status | Screenshot |
|------|-------|--------|----------|--------|--------|--------|------------|
| 1 | Super Admin | Login, navigate to Create User | Form visible | Sheet opened with "Create User" heading | Playwright | PASS | none |
| 2 | Super Admin | Submit with `uat.employee@test.com` (existing email) | Error: email already in use | Error message displayed (red text / destructive class) in form | Playwright | PASS | none |

---

### Scenario 4 — Admin Filters Users by Department

| Step | Actor | Action | Expected | Actual | Method | Status | Screenshot |
|------|-------|--------|----------|--------|--------|--------|------------|
| 1 | Super Admin | Login, navigate to Users | Users list visible | Table visible | Playwright | PASS | none |
| 2 | Super Admin | Filter by Department = "UAT Engineering" | Only UAT Engineering users shown | Filter applied via native select; filtered rows all contained "UAT Engineering" | Playwright | PASS | none |
| 3 | Super Admin | Clear filter | All users shown again | Blank option selected; row count returned to ≥ filtered count | Playwright | PASS | none |

---

## Failed Steps Detail

### Step 15 (Scenario 1) — Employee Portal has no Logout button

**Expected**: Employee clicks Logout; returned to login page
**Actual**: PortalLayout (`src/layouts/PortalLayout.tsx`) contains no logout button. Only a LocaleSwitcher is in the nav. There is no mechanism for an employee to log out from the portal.
**Error**: `locator('header').getByRole('button').last()` timeout — no button matching "Sign out" found
**Screenshot**: test-results/uat-user-onboarding-202606-be0d7-sword-change-on-first-login-chromium-uat/test-failed-1.png
**Possible cause**: Logout was not implemented in the PortalLayout during development. AdminLayout correctly has `TopBar` with a Sign Out button; PortalLayout has no equivalent.

---

### Step 5 (Scenario 2) — Inactive account can log in

**Expected**: Login rejected; error shown (e.g., "Account is inactive")
**Actual**: POST `/api/v1/auth/login` with `uat.employee@test.com` / `NewPass123!` succeeded. User landed on `/portal` with HTTP 200. The database shows `users.status = 'inactive'` for this user.
**Error**: `expect(wasBlocked).toBeTruthy()` — page.url() was `http://localhost/portal` not `/login`
**Screenshot**: test-results/uat-user-onboarding-202606-4c3a7-activate-User-Login-Blocked-chromium-uat/test-failed-1.png
**Possible cause**: The login handler (`backend/internal/auth/handler.go`) does not query or check the `users.status` field before issuing a JWT. The check needs to be added to the authentication service.

---

## Acceptance Criteria Coverage

| AC# | Criterion | Steps | Status |
|-----|-----------|-------|--------|
| 1 | Admin creates a single user; user appears in list with status "Active" | Sc1, Steps 7–9 | PASS |
| 2 | CSV import: invalid rows flagged, valid rows committed | Not executed | NOT COVERED (noted in scenario) |
| 3 | `force_password_change` employee cannot access portal pages | Sc1, Steps 12–13 | PASS |
| 4 | After forced password change, employee lands on portal | Sc1, Step 14 | PASS |
| 5 | Deactivating prevents login; historical data remains | Sc2, Steps 3–5 | **FAIL** — DEFECT-2: deactivated user can still log in |
| 6 | Password reset causes employee to change on next login | Not executed | NOT COVERED (noted in scenario) |
| 7 | Admin filters users by department, role, or status | Sc4, Steps 2–3 | PASS |
| 8 | Deleting a department with users shows an error | Not executed | NOT COVERED (noted in scenario) |

---

## Defects Found

### DEFECT-1: Employee Portal has no logout mechanism
- **Severity**: Medium
- **Location**: `frontend/src/layouts/PortalLayout.tsx`
- **Observed**: No Sign Out button exists in the employee portal navigation
- **Expected**: Employee should be able to log out from the portal
- **AC affected**: Scenario 1 Step 15

### DEFECT-2: Inactive users can authenticate
- **Severity**: Critical
- **Location**: `backend/internal/auth/` — login handler / authentication service
- **Observed**: `POST /api/v1/auth/login` issues a valid JWT for a user with `users.status = 'inactive'`. Employee lands on `/portal` and has full access.
- **Expected**: Login should fail with HTTP 401 and an appropriate error code (e.g., `ACCOUNT_INACTIVE`) when `users.status != 'active'`
- **AC affected**: FR-BB14/FR-BB18 AC#5 — deactivation must prevent login

---

## Additional Finding

**Precondition discrepancy in scenario script**: The scenario specifies admin credentials `admin@test.com / Admin1234!`. The actual seeded admin account is `admin@bilimbaga.local / Admin1234!`. The scenario script should be updated.

---

## Environment
- Frontend: http://localhost (Nginx port 80; internal Vite on port 5173 not exposed)
- Backend: http://localhost:8080
- Browser: Chromium (Playwright)
- Stack started by: already running (Infrastructure Configuration confirmed)
- DB state: PostgreSQL 16 in bilimbaga-db-1 container

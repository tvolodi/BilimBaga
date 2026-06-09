# UAT Report — User Onboarding (Rerun)

**Run ID**: uat-user-onboarding-20260609-rerun  
**Date**: 2026-06-09  
**Scenario file**: [docs/uat-scenarios/user-onboarding-20260609.md](../uat-scenarios/user-onboarding-20260609.md)  
**Executor**: UAT Runner (automated Playwright)  
**Stack**: Docker Compose @ http://localhost (Nginx port 80)  
**Purpose**: Verify ISS-030 and ISS-031 fixes; confirm full User Onboarding UAT PASS

---

## Summary

| Result | Scenarios | Steps |
|--------|-----------|-------|
| PASS   | 5/5       | 21/21 |

**Overall verdict: PASS**

---

## Defects Verified (Resolved)

| ID | Title | Verification |
|----|-------|-------------|
| ISS-030 | Inactive user can authenticate | PASS — `POST /api/v1/auth/login` returns `ACCOUNT_INACTIVE` / 401 for deactivated user |
| ISS-031 | Employee Portal has no logout button | PASS — Sign Out button present in portal nav; navigates to `/login` on click |

---

## Scenario Results

### Scenario 1 — Create Department and Single User, Employee First Login

**Result**: PASS (7.8s)

| Step | Description | Outcome |
|------|-------------|---------|
| 1–2 | Login as admin (`admin@bilimbaga.local`) | PASS — redirected to `/admin` |
| 3 | Navigate to Departments page | PASS — heading visible |
| 4–5 | Create "UAT Engineering" department (skip if exists) | PASS — department listed |
| 6 | Navigate to Users page | PASS — users table visible |
| 7 | Open New User sheet | PASS — "Create User" heading visible |
| 8 | Fill form: email, name, department (tree select), role (employee) | PASS |
| 9 | Submit Create | PASS |
| 10 | Temporary password dialog shown | PASS |
| 11 | Admin Sign Out → redirected to `/login` | PASS |

### Scenario 1 (Employee) — Forced Password Change + Portal Logout (ISS-031 verify)

**Result**: PASS (6.4s)

| Step | Description | Outcome |
|------|-------------|---------|
| 12 | Employee logs in with temp password → forced-change flow | PASS — not redirected to `/portal` |
| 13 | Direct nav to `/portal` blocked while password not changed | PASS — redirected away |
| 14 | Complete password change form → redirected to `/portal` | PASS |
| 15 | Sign Out button visible in Employee Portal | PASS — ISS-031 verified |
| 15 | Click Sign Out → redirected to `/login` | PASS |
| 16 | Employee logs in with new password → no forced-change, lands on `/portal` | PASS |

### Scenario 2 — Deactivate User, Login Blocked (ISS-030 verify)

**Result**: PASS (5.3s)

| Step | Description | Outcome |
|------|-------------|---------|
| 1–2 | Admin login, navigate to Users, confirm employee is Active | PASS |
| 3 | Click Deactivate button on employee row | PASS |
| 4 | Confirm deactivation dialog | PASS — status shows "Inactive" |
| 5 | Employee login attempt → blocked (ISS-030 fix) | PASS — URL remains `/login`, error shown: `account is inactive` |

### Scenario 3 — Duplicate Email Rejected

**Result**: PASS (5.0s)

| Step | Description | Outcome |
|------|-------------|---------|
| 1–3 | Admin opens New User form, fills existing employee email | PASS |
| 4 | Submit → duplicate error shown | PASS — error message visible |

### Scenario 4 — Filter Users by Department

**Result**: PASS (4.6s)

| Step | Description | Outcome |
|------|-------------|---------|
| 1 | Admin logs in, navigates to Users | PASS |
| 2 | Filter by "UAT Engineering" department | PASS — only matching rows shown |
| 3 | Clear filter → all users shown | PASS — row count ≥ filtered count |

---

## Infrastructure Notes

- Docker containers rebuilt to deploy fixes: `bilimbaga-api-1` (ISS-030, Go binary rebuilt), `bilimbaga-frontend-1` (ISS-031, Vite dist rebuilt locally and injected)
- UAT employee `uat.employee@test.com` reactivated (`status = 'active'`) before rerun

---

## Decision

**PASS** — all 5 scenarios pass, both defects (ISS-030, ISS-031) confirmed resolved. User Onboarding business process is `uat-verified`.

**Next step**: Release Finalizer — mark requirement as `uat-verified`, commit UAT artifacts.

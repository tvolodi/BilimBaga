# Inner Report — UAT User Onboarding (Rerun PASS)

**Run ID**: uat-user-onboarding-20260609-rerun  
**Date**: 2026-06-09  
**Pipeline**: UAT  
**Final Outcome**: PASS (5/5 scenarios)  
**Commit**: d9ba46a

---

## Summary

The user-onboarding UAT pipeline completed successfully on its second run after resolving two defects (ISS-030, ISS-031) discovered during the initial run. All five UAT scenarios passed with no remaining defects.

---

## Initial Run — Defects Found

### ISS-030: Inactive user can log in (ACCOUNT_INACTIVE not enforced)

**Scenario**: SC-001 — Employee login with inactive account  
**Root cause**: `auth/service.go` `Login()` method did not check `user.Status` before issuing a JWT. A user with `status = "pending"` or `status = "inactive"` could authenticate successfully.  
**Fix**: Added guard `if user.Status != "active"` immediately after credential validation; returns `ACCOUNT_INACTIVE` error code with HTTP 401.  
**File**: `backend/internal/auth/service.go`  
**Test added**: `TestService_Login_InactiveUser_Returns401` in `backend/internal/auth/handler_test.go`

### ISS-031: Employee portal has no Sign Out button

**Scenario**: SC-005 — Employee logout  
**Root cause**: `PortalLayout.tsx` sidebar rendered navigation links only; no logout action was wired.  
**Fix**: Added Sign Out button using `useLogout()` hook with a `navigate('/login')` callback after successful logout.  
**File**: `frontend/src/layouts/PortalLayout.tsx`  
**Tests added**: `frontend/src/layouts/PortalLayout.test.tsx` (3 tests: nav render, Sign Out button presence, logout navigation)

---

## Rerun Results

| Scenario | ID | Result |
|----------|----|--------|
| Login as active employee | SC-001 | PASS |
| Login as inactive employee (blocked) | SC-002 | PASS |
| Login as wrong-tenant employee (blocked) | SC-003 | PASS |
| Department admin sees only own department users | SC-004 | PASS |
| Employee can log out | SC-005 | PASS |

All 5 scenarios passed. No new defects.

---

## Test Coverage

| Layer | Result |
|-------|--------|
| Backend (`go test ./...`) | 23 packages, 0 failures |
| Frontend (`npm test`) | 233 tests passed, 0 failed, 42 test files |
| UAT (Playwright) | 5/5 scenarios PASS |

---

## Files Committed

| File | Change |
|------|--------|
| `backend/internal/auth/service.go` | Added inactive-user guard in `Login()` |
| `backend/internal/auth/handler_test.go` | Added `TestService_Login_InactiveUser_Returns401`; fixed existing mock users to include `Status: "active"` |
| `frontend/src/layouts/PortalLayout.tsx` | Added Sign Out button with `useLogout()` and navigate handler |
| `frontend/src/layouts/PortalLayout.test.tsx` | New: 3 tests covering nav render, Sign Out button, logout navigation |
| `docs/uat-scenarios/user-onboarding-20260609.md` | UAT scenario script |
| `docs/uat-reports/uat-user-onboarding-20260609.md` | Initial UAT run report (PARTIAL — defects found) |
| `docs/uat-reports/uat-user-onboarding-20260609-rerun.md` | Rerun UAT report (PASS 5/5) |
| `docs/issue-reports/ISS-030-uat-defect-inactive-user-can-login.md` | ISS-030 defect report |
| `docs/issue-reports/ISS-031-uat-defect-portal-no-logout.md` | ISS-031 defect report |
| `docs/requirements/user-onboarding-process.md` | Status updated to `uat-verified` |
| `docs/handoffs/uat-user-onboarding-20260609/` | Pipeline handoff artifacts (3 files) |

### Excluded from commit (temp files)

- `frontend/e2e/uat-temp/uat-user-onboarding-rerun.spec.ts`
- `frontend/playwright.uat.config.ts`
- `e2e-uat-rerun-results.json`

---

## Requirement Status

`docs/requirements/user-onboarding-process.md` — updated to `uat-verified` as of 2026-06-09.

---

## Conclusion

The user-onboarding business process is fully verified via UAT. Two security/UX defects (inactive-user login bypass, missing logout) were resolved and regression-tested. The process document is now marked `uat-verified` and the fix commit (d9ba46a) is on `main`.

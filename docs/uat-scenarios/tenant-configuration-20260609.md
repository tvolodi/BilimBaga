---
slug: tenant-configuration
title: "Tenant Configuration — UAT Scenario"
feature: tenant-configuration (FR-BB13, FR-BB112, FR-BB62)
version: 1
created: 2026-06-09
author: Business Analyst
---

Target: local | qa (default: local; never the production-class demo instance, see DEC-001)

## Preconditions

- The platform is running at `http://localhost:5173`.
- Super Admin: `admin@test.com` / `Admin1234!`.
- A Department Admin account `dept.admin@test.com` / `DeptAdmin1!` exists (or create it before starting).
- A small PNG/JPG image file is available locally for upload as a new logo.

---

## Scenario 1: Update Branding — Logo and Colours Reflected Immediately

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Super Admin | Log in as `admin@test.com` / `Admin1234!` | Admin Dashboard visible | |
| 2 | Super Admin | Navigate to Settings → Branding | Branding settings page is visible with current logo, application name, and colour fields | |
| 3 | Super Admin | Note the current application name | Current name noted | |
| 4 | Super Admin | Change "Application name" to `BilimBaga UAT` | Field updated | |
| 5 | Super Admin | Upload a new logo image (drag-and-drop or file picker) | Logo preview updates in the live preview panel | |
| 6 | Super Admin | Change "Primary colour" to `#1A73E8` | Colour picker or hex field updated; live preview reflects the new colour | |
| 7 | Super Admin | Click "Save" | Success message shown | |
| 8 | Super Admin | Click "Logout" | Returned to login page | |
| 9 | Super Admin | Assert (visual): login page shows the new logo and new primary colour | New logo and colour are visible on the login page without requiring a hard refresh | |
| 10 | Super Admin | Log in again as `admin@test.com` / `Admin1234!` | Admin shell reflects the new application name "BilimBaga UAT" in the page title or header | |

---

## Scenario 2: WCAG Contrast Warning

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Super Admin | Navigate to Settings → Branding | Branding page visible | |
| 2 | Super Admin | Set "Primary colour" to `#FFFFFF` (white) | Colour updated | |
| 3 | Super Admin | Assert: a WCAG contrast warning is visible (e.g. "This colour fails WCAG AA contrast") | Warning message appears | |
| 4 | Super Admin | Assert: the "Save" button is still enabled (warning only, not a block) | Save button is clickable | |
| 5 | Super Admin | Restore "Primary colour" to a valid colour (e.g. `#1A73E8`) and save | Warning disappears; save succeeds | |

---

## Scenario 3: Department Admin Cannot Access Branding Settings

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Dept Admin | Log in as `dept.admin@test.com` / `DeptAdmin1!` | Admin shell visible | |
| 2 | Dept Admin | Attempt to navigate to Settings → Branding (click in sidebar if visible, or navigate to `http://localhost:5173/settings/branding` directly) | Settings → Branding is not visible in the sidebar OR navigation to the URL redirects to a "Not found" or "Access denied" page | |

---

## Scenario 4: Restore Original Application Name

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Super Admin | Navigate to Settings → Branding | Branding page visible showing "BilimBaga UAT" as current name | |
| 2 | Super Admin | Change "Application name" back to `BilimBaga` (or the original name) and save | Application name restored; page title updates on next navigation | |

---

## Acceptance Criteria Coverage

| AC# | Criterion | Covered by Scenario |
|-----|-----------|---------------------|
| 1 | After saving new logo, login page displays new logo on fresh load | Scenario 1, Steps 7–9 |
| 2 | After saving new colours, UI reflects new colours for all users on next navigation | Scenario 1, Steps 6–10 |
| 3 | WCAG contrast warning visible for failing colour; save button remains enabled | Scenario 2, Steps 2–4 |
| 4 | After changing default locale, new questions must have that locale translation | Not covered in this run (locale change tested in extended run) |
| 5 | Certificate downloaded before branding change retains old branding | Not covered in this run (requires two certificates at different branding states) |
| 6 | Department Admin cannot access Branding settings page | Scenario 3, Steps 1–2 |

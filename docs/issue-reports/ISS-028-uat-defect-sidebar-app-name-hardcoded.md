---
id: ISS-028
title: "UAT Defect: Tenant Configuration — Admin sidebar app name hardcoded, does not reflect tenant config"
status: resolved
severity: medium
layer: frontend
module: admin/Sidebar
tags: [uat, tenant-config, branding, sidebar]
created: 2026-06-09
resolved: 2026-06-09
recurrence_count: 1
related_issues: []
regression_test: "src/components/admin/Sidebar.test.tsx — 'displays app_name from tenant config instead of hardcoded literal'"
---

## Symptom
UAT Scenario: `Scenario 1: Update Branding — Logo and Colours Reflected Immediately`, Step 10
Actor: Super Admin
Action: Re-login as admin; assert admin shell reflects "BilimBaga UAT" in header
Expected: Admin sidebar displays the saved application name "BilimBaga UAT"
Actual: Admin sidebar still shows the hardcoded literal "BilimBaga" regardless of the saved tenant config value. The login page title correctly updated to "Sign in to BilimBaga UAT" but the post-login admin sidebar did not.
Screenshot: playwright-report-live/uat-s1-login-branding.png

## Root Cause
`src/components/admin/Sidebar.tsx` rendered the application name as a hardcoded string literal (`BilimBaga`) in the sidebar header. It never imported or called `useTenantConfig`, so it was completely disconnected from the tenant configuration API that other components (e.g. `LoginPage.tsx`) already use to read `app_name`.

## Fix Applied
1. Added `import { useTenantConfig } from '@/api/useTenantConfig'` to `Sidebar.tsx`.
2. Added `const { data: tenantConfig } = useTenantConfig()` inside the `Sidebar` component.
3. Replaced the hardcoded literal with `{tenantConfig?.app_name ?? 'BilimBaga'}` — the fallback ensures graceful handling during the loading state.

No styling or layout changes were made.

## Files Changed
- `frontend/src/components/admin/Sidebar.tsx` — import + hook call + dynamic app name
- `frontend/src/components/admin/Sidebar.test.tsx` — added `vi.mock` for `useTenantConfig`; added two new regression tests (tenant name displayed, fallback to default)

## Regression Test
`src/components/admin/Sidebar.test.tsx`:
- **"displays app_name from tenant config instead of hardcoded literal"** — mocks `useTenantConfig` returning `{ app_name: 'AcmeCorp' }`, renders Sidebar, asserts `AcmeCorp` is visible and `BilimBaga` is absent.
- **"falls back to 'BilimBaga' when tenant config is not yet loaded"** — mocks `useTenantConfig` returning `{ data: undefined }`, renders Sidebar, asserts `BilimBaga` is shown.

## Resolution Results
All 41 test files, 230 tests pass (`npm test -- --run`). No regressions introduced.

---

## Evidence

`src/components/admin/Sidebar.tsx` contains the literal:
```tsx
<span className="text-lg font-bold tracking-tight truncate">BilimBaga</span>
```

The login page correctly reads `app_name` from `/api/v1/tenant/config` (login page title passes). The Sidebar component never consumes tenant config — it renders a hardcoded string.

## Expected Behaviour

`Sidebar.tsx` should read `app_name` from the tenant config query (already fetched by the app for the login page) and display it in place of the hardcoded "BilimBaga" literal.

## Acceptance Criteria Impacted

- AC#2: "After saving new colours, UI reflects new colours for all users on next navigation"
- Requirement: FR-BB13

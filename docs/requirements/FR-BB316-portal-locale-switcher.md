# FR-BB316 — Portal Locale Switcher

**Status**: draft  
**Phase**: 3.16  
**Depends on**: FR-BB313 (Frontend: Employee Portal), FR-BB62 (Full i18n Coverage)

## Summary

Employees using the exam portal currently have no way to switch the UI language after logging in. This requirement adds the existing `LocaleSwitcher` component to the `PortalLayout` header bar so that exam-taking users can change the display language (Kazakh, Russian, English) from within the portal — mirroring the functionality already available to admins in the admin `TopBar`.

## Scope

| Layer | Items |
|-------|-------|
| Database | None |
| API endpoints | None (tenant config already served via `GET /api/v1/tenant/config`) |
| Frontend pages/components | `frontend/src/layouts/PortalLayout.tsx` (add switcher to header) |
| i18n keys | None new — existing `LocaleSwitcher` reused as-is |

## Acceptance Criteria

- **AC1**: The portal header bar (`PortalLayout`) contains a visible locale switcher control rendered to the right of the navigation tabs.
- **AC2**: Selecting a locale from the switcher changes all visible portal UI strings immediately without a page reload.
- **AC3**: After switching locales, navigating between portal tabs (My Exams, My Results) retains the selected locale.
- **AC4**: The selected locale persists across browser refreshes (stored in `localStorage` under key `i18n-lang`, matching the existing `LocaleSwitcher` behaviour).
- **AC5**: The switcher is only visible on the portal layout — it does not appear on the admin layout or login screen (those have their own switchers).
- **AC6**: The switcher renders without horizontal overflow on viewport widths ≥ 375 px (mobile-first).

## Technical Notes

### Database

None.

### API Contract

None. `LocaleSwitcher` reads and writes `localStorage` and calls `i18n.changeLanguage()` directly; no API calls are needed.

### Go Implementation Notes

None — this is a purely frontend change.

### Frontend Implementation Notes

**File to change**: `frontend/src/layouts/PortalLayout.tsx`

1. Import `LocaleSwitcher` from `@/components/LocaleSwitcher`.
2. Wrap the sticky header `<nav>` and the new switcher in a flex row inside the sticky `<div>`:

```tsx
<div className="border-b bg-background sticky top-0 z-10">
  <div className="max-w-7xl mx-auto flex items-center justify-between px-6">
    <nav className="flex" aria-label="Portal navigation">
      <NavLink to="/portal" end className={tabClass}>
        {t('portal.tab_exams', 'My Exams')}
      </NavLink>
      <NavLink to="/portal/results" className={tabClass}>
        {t('portal.tab_results', 'My Results')}
      </NavLink>
    </nav>
    <LocaleSwitcher />
  </div>
</div>
```

- `LocaleSwitcher` already handles `i18n.changeLanguage`, `localStorage`, and `document.lang` updates — no additional wiring needed.
- No new i18n keys are required; the component labels (`Қазақша`, `Русский`, `English`) are hard-coded in the component itself by design.
- The `<Select>` inside `LocaleSwitcher` has a fixed `w-[120px]` class; on the portal header it should align naturally with `ml-auto` or `justify-between` layout.

## Out of Scope

- Filtering the locale list by `tenant.available_locales` — the existing `LocaleSwitcher` shows all three locales unconditionally; restricting by tenant config is a separate enhancement.
- Adding a locale switcher to the exam-taking screen (`/portal/exam/:sessionId`) — that page has its own full-screen layout and is out of scope here.
- Any backend changes.
- Admin layout changes (already handled).
- Login page changes (already has `LanguageSelector`).

## Test Strategy

- **Component test** (`PortalLayout`): assert that `LocaleSwitcher` is rendered inside the layout header.
- **Integration / E2E**: navigate to `/portal`, click the locale switcher, select a different locale, assert that the tab labels change to the selected language without reload.
- No backend tests needed.

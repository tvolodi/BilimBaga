---
id: ISS-014
title: Axe accessibility violations on admin pages — missing main landmark, h1, region, skip-link
status: open
severity: medium
layer: frontend
module: auth
tags: [axe, accessibility, landmark, skip-link, h1, region, main]
created: 2026-05-19
resolved: null
recurrence_count: 1
related_issues: []
regression_test: null
---

## Symptom
Browser console (via @axe-core/react) reports the following violations on the question edit
page (and on other pages) in development mode:

1. **landmark-one-main** — Document should have one main landmark
2. **page-has-heading-one** — Page should contain a level-one heading (`<h1>`)
3. **region** — All page content should be contained by landmarks
4. **skip-link** — The skip-link target should exist and be focusable

## Root Cause

Three separate but related issues:

### 1. LoginPage and ChangePasswordPage use `<div>` instead of `<main>`
`LoginPage.tsx` and `ChangePasswordPage.tsx` both render their page content as:
```jsx
<div id="main-content" tabIndex={-1} className="...">
```
A `<div>` is not a landmark element. This causes `landmark-one-main` (no `<main>` exists)
and `region` (content is not in a landmark region) violations whenever these pages are displayed.

### 2. ResultPage has no `<main>` and no `id="main-content"` at all
`ResultPage.tsx` wraps its content in a plain `<div className="min-h-screen bg-background ...">`.
There is no `<main>` landmark and no element with `id="main-content"`. This triggers
`landmark-one-main`, `region`, AND `skip-link` (no target for the global SkipLink).

### 3. AppRoutes loading state renders FullPageSpinner with no landmark or heading
In `App.tsx`, `AppRoutes` returns `<FullPageSpinner />` while `useRefreshToken` is loading.
`FullPageSpinner` is a plain `<div>`. Since `@axe-core/react` fires 1 second after each React
render, if the token refresh takes >1 second (common on first load or slow networks), axe
runs against a DOM that has:
- Only the `<a>` skip link (no `<main>`, no `<h1>`, no `id="main-content"`)
This causes all four violations to appear on any page the user is navigating to, including the
admin question edit page.

## Fix Applied

### 1. LoginPage.tsx
Changed `<div id="main-content" tabIndex={-1}>` → `<main id="main-content" tabIndex={-1}>`.
Fixes `landmark-one-main` and `region` violations on the login page.

### 2. ChangePasswordPage.tsx
Same change: `<div id="main-content" tabIndex={-1}>` → `<main id="main-content" tabIndex={-1}>`.

### 3. ResultPage.tsx
Changed the outer wrapper from a plain `<div className="min-h-screen ...">` to
`<main id="main-content" tabIndex={-1} className="min-h-screen ... outline-none">`.
Fixes `landmark-one-main`, `region`, and `skip-link` violations on the result page.

### 4. App.tsx — AppRoutes loading state
Changed the `isLoading` early return from:
```jsx
return <FullPageSpinner />
```
to:
```jsx
return (
  <main id="main-content" tabIndex={-1} className="outline-none">
    <h1 className="sr-only">{/* Loading */}</h1>
    <FullPageSpinner />
  </main>
)
```
Ensures that during the token-refresh loading phase, the DOM has a `<main>` landmark,
an `<h1>` (screen-reader only), and a valid skip-link target.

## Files Changed

| File | Change |
|------|--------|
| `frontend/src/pages/auth/LoginPage.tsx` | `<div id="main-content">` → `<main id="main-content">` |
| `frontend/src/pages/auth/ChangePasswordPage.tsx` | `<div id="main-content">` → `<main id="main-content">` |
| `frontend/src/pages/ResultPage.tsx` | Outer div → `<main id="main-content" tabIndex={-1}>` |
| `frontend/src/App.tsx` | Loading state wrapped in `<main>` with sr-only `<h1>` |
| `docs/issue-reports/ISS-014-axe-accessibility-violations-admin.md` | This file |

## Regression Test
None added — the violations are observable via @axe-core/react in the browser console in dev mode.
The build check (`npm run build`) confirms no TypeScript errors were introduced.

## Resolution Results
- Tests: pending
- Migration applied: no
- Build clean: pending

## Recurrence Log
| Date | Trigger | Action Taken |
|------|---------|--------------|
| 2026-05-19 | ISS-014 initial report | Applied fixes to LoginPage, ChangePasswordPage, ResultPage, App.tsx |

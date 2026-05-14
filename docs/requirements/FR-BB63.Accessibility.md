# FR-BB63 — Accessibility

## Metadata
| Field | Value |
|-------|-------|
| ID | FR-BB63 |
| Phase | 6 — Polish & Hardening |
| Priority | 2 |
| Status | Draft |
| Depends On | FR-BB110, FR-BB111, FR-BB112, FR-BB26, FR-BB27, FR-BB312, FR-BB313, FR-BB314, FR-BB45, FR-BB46, FR-BB47, FR-BB56, FR-BB57, FR-BB58 |

## Description
Brings the entire frontend to WCAG 2.1 Level AA conformance. All interactive elements are keyboard-navigable and screen-reader-friendly. The exam timer makes live time announcements. Color contrast ratios meet AA thresholds, and the branding settings screen warns when a chosen color combination fails. Forms use proper label associations and error linkage via ARIA attributes.

## Acceptance Criteria
- [ ] AC-1: Every interactive element (buttons, links, form controls, modal dialogs, dropdown menus) is reachable and operable via keyboard alone; tab order follows visual reading order; no keyboard focus traps exist except inside open modal dialogs (where focus is intentionally constrained until dismissed).
- [ ] AC-2: A visible focus ring is present on all interactive elements when focused via keyboard; no `outline: none` or `outline: 0` rule is applied without a fully visible custom replacement style.
- [ ] AC-3: All icon-only buttons, status badges, progress indicators, and chart elements carry `aria-label` or `aria-labelledby` attributes describing their purpose; buttons with both icon and text do not duplicate the label in ARIA.
- [ ] AC-4: The exam countdown timer announces remaining time via an `aria-live="assertive"` region at the 5-minute mark and again when 1 minute remains; outside those thresholds the live region is silent to avoid constant interruptions.
- [ ] AC-5: All text elements meet WCAG 2.1 AA color-contrast ratios (minimum 4.5:1 for body text, 3:1 for large text ≥18pt or ≥14pt bold, and 3:1 for UI component boundaries such as input borders).
- [ ] AC-6: Every `<form>` input has a programmatically associated `<label>` element (via `htmlFor`/`id` pairing or `aria-labelledby`); placeholder text alone is never used as the only label.
- [ ] AC-7: Inline validation error messages are programmatically linked to their input via `aria-describedby`; error messages are not communicated by color alone.
- [ ] AC-8: A "Skip to main content" anchor link is rendered as the first focusable element on every page; it becomes visible on focus and moves keyboard focus to the `<main>` landmark.
- [ ] AC-9: All non-decorative images and logos have descriptive `alt` text; purely decorative images use `alt=""` so screen readers skip them.
- [ ] AC-10: The full exam-taking flow (reading question, selecting answer, navigating to next question, submitting) is navigable and operable using VoiceOver on macOS and NVDA on Windows.

## Technical Specification

### Frontend Components

**Skip link** (`src/components/SkipLink.tsx`):
```tsx
export function SkipLink() {
  return (
    <a
      href="#main-content"
      className="sr-only focus:not-sr-only focus:absolute focus:top-2 focus:left-2
                 focus:z-50 focus:rounded focus:bg-primary focus:px-4 focus:py-2 focus:text-white"
    >
      {t('common.skipToMain')}
    </a>
  );
}
```
Placed as the very first child of `<body>` in `App.tsx`. Target `<main>` element has `id="main-content"` and `tabIndex={-1}`.

**Focus ring** — Tailwind `tailwind.config.ts`:
```ts
theme: {
  extend: {
    ringWidth: { DEFAULT: '2px' },
    ringColor: { DEFAULT: 'hsl(var(--primary))' },
  }
}
```
Global CSS resets `outline: none` only on elements that replace it with a Tailwind `ring` class. ESLint rule `jsx-a11y/no-outline-none` catches violations.

**Live timer announcer** (`src/components/ExamTimer.tsx`):
```tsx
const [announcement, setAnnouncement] = useState('');

useEffect(() => {
  if (secondsRemaining === 300) setAnnouncement(t('session.timerFiveMinutes'));
  if (secondsRemaining === 60)  setAnnouncement(t('session.timerOneMinute'));
  if (secondsRemaining <= 0)    setAnnouncement(t('session.timerExpired'));
}, [secondsRemaining]);

return (
  <>
    <div aria-live="assertive" aria-atomic="true" className="sr-only">
      {announcement}
    </div>
    <div aria-hidden="true">{formatTime(secondsRemaining)}</div>
  </>
);
```

**Icon-only button pattern**:
```tsx
// Correct
<Button variant="ghost" size="icon" aria-label={t('common.deleteQuestion')}>
  <TrashIcon className="h-4 w-4" aria-hidden="true" />
</Button>
```

**Form label / error pattern**:
```tsx
<div>
  <Label htmlFor="email">{t('auth.emailLabel')}</Label>
  <Input
    id="email"
    aria-describedby={errors.email ? 'email-error' : undefined}
    aria-invalid={!!errors.email}
  />
  {errors.email && (
    <p id="email-error" role="alert" className="text-destructive text-sm mt-1">
      {errors.email.message}
    </p>
  )}
</div>
```

**Chart accessibility** (analytics pages):
- Recharts / similar: each chart wrapped with `role="img"` and `aria-label` describing the chart title and data range.
- Data tables provided as visually hidden alternatives for bar/line charts.

**Status badges**:
```tsx
<Badge aria-label={t(`status.${status}.label`)} variant={badgeVariant(status)}>
  {t(`status.${status}.short`)}
</Badge>
```

**Modal / Dialog focus trap**: shadcn/ui `<Dialog>` already implements focus trap via Radix UI. Verify `aria-modal="true"` is present and `aria-labelledby` points to the dialog title.

**Color contrast tooling**:
- `eslint-plugin-jsx-a11y` installed and enabled for all `.tsx` files.
- `@axe-core/react` added in development only (`import` guarded by `if (process.env.NODE_ENV !== 'production')`); logs violations to the browser console.
- WCAG AA check in branding settings (see FR-BB112): uses `wcag-contrast` npm package formula:
  ```ts
  import { hex } from 'wcag-contrast';
  const ratio = hex(primaryColor, backgroundColor); // returns number
  const passes = ratio >= 4.5; // AA normal text
  ```

**Recommended ESLint plugins**:
- `eslint-plugin-jsx-a11y` — catches missing `alt`, missing labels, improper ARIA roles.

## Notes
- All shadcn/ui primitives (Button, Input, Select, Dialog, etc.) are built on Radix UI which provides ARIA semantics by default; custom styling must not override Radix UI's ARIA attributes.
- RTL keyboard navigation (arrow keys in menus, etc.) is handled by Radix UI automatically when `dir="rtl"` is set on the `<html>` element (see FR-BB62).
- Screen reader testing with VoiceOver + Safari and NVDA + Chrome/Firefox should be performed as a manual QA step before the Phase 6 release. Automated tools catch ~30-40% of accessibility issues; manual testing is required for AC-10.
- The `@axe-core/react` development overlay should be removed or kept behind a feature flag and never shipped to production.

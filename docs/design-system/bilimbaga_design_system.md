# BilimBaga — Design System

> Agent-facing reference. Use this file when generating any frontend code for BilimBaga.
> Stack: React 18 + TypeScript + Vite + Tailwind CSS + shadcn/ui

---

## 1. Brand identity

### 1.1 Application name

- Full name: **BilimBaga**
- Tagline: Corporate Learning & Assessment
- Origin: Kazakh — "Bilim" (knowledge) + "Baға" (grade/score)

### 1.2 Logo files

| File | Usage |
|---|---|
| `logo-light.svg` | On white or light backgrounds |
| `logo-dark.svg` | On dark/navy backgrounds |
| `icon.svg` | App icon, standalone mark |
| `favicon.svg` | Browser tab, 16–32px |

Logo mark: hexagonal frame with inner 4-pointed star pattern (Central Asian geometric motif) and a gold center dot.
Wordmark: "Bilim" bold weight + "BAGA" light spaced caps, stacked two lines.

**Never** stretch, recolor, or place the logo on a busy background.
**Minimum size:** 120px wide for full logo, 24px for icon mark.

---

## 2. Color system

### 2.1 Brand palette

```css
:root {
  --color-navy:         #1B3A6B;  /* deep navy — primary brand, dark surfaces */
  --color-blue:         #2E6DB4;  /* primary blue — main interactive color */
  --color-blue-light:   #5AA0E0;  /* light blue — hover states, icon fills */
  --color-blue-pale:    #D8E8F5;  /* pale blue — subtle backgrounds, badges */
  --color-gold:         #C8A84B;  /* gold accent — certificates, highlights */
  --color-gold-light:   #F0E4B8;  /* gold pale — gold badge backgrounds */
}
```

### 2.2 Semantic tokens (Tailwind CSS custom properties)

Define in `tailwind.config.ts` and `globals.css`:

```css
:root {
  /* Backgrounds */
  --bg-page:            #F4F6FA;
  --bg-surface:         #FFFFFF;
  --bg-surface-raised:  #FFFFFF;
  --bg-subtle:          #EEF2F8;
  --bg-navy:            #1B3A6B;

  /* Text */
  --text-primary:       #1A1F2E;
  --text-secondary:     #4A5568;
  --text-muted:         #8A94A6;
  --text-on-navy:       #FFFFFF;
  --text-on-blue:       #FFFFFF;
  --text-link:          #2E6DB4;

  /* Borders */
  --border-default:     #DDE2ED;
  --border-strong:      #B8C4D8;
  --border-focus:       #2E6DB4;

  /* Status */
  --color-success:      #1E7E4E;
  --bg-success:         #E8F7EF;
  --color-warning:      #92610A;
  --bg-warning:         #FEF3DC;
  --color-danger:       #C0392B;
  --bg-danger:          #FDECEA;
  --color-info:         #1B5FA8;
  --bg-info:            #E6F1FB;
}

.dark {
  --bg-page:            #0F1623;
  --bg-surface:         #1A2133;
  --bg-surface-raised:  #222D42;
  --bg-subtle:          #1E2A3D;
  --bg-navy:            #0D1E3A;

  --text-primary:       #E8ECF4;
  --text-secondary:     #9AAABB;
  --text-muted:         #5A6A7E;
  --text-on-navy:       #E8ECF4;
  --text-link:          #5AA0E0;

  --border-default:     #2A3550;
  --border-strong:      #3D4F6E;
  --border-focus:       #5AA0E0;

  --color-success:      #4ABA7E;
  --bg-success:         #0F2E1E;
  --color-warning:      #E0A030;
  --bg-warning:         #2A1E08;
  --color-danger:       #E06050;
  --bg-danger:          #2A0F0D;
  --color-info:         #5AA0E0;
  --bg-info:            #0C1E35;
}
```

### 2.3 Color usage rules

- **Navy** (`--color-navy`) — sidebar background, top nav on dark mode, login screen background panel
- **Blue** (`--color-blue`) — primary buttons, links, active nav items, focus rings, form borders on focus
- **Gold** (`--color-gold`) — certificates only, achievement badges, the logo center dot. Never use for UI actions.
- **Status colors** — always use semantic pairs: `--color-success` text on `--bg-success` background, etc.
- **Never** use raw hex values in component code. Always reference CSS variables or Tailwind tokens.

---

## 3. Typography

### 3.1 Font stack

```css
--font-sans: 'Inter', 'Segoe UI', system-ui, sans-serif;
--font-mono: 'JetBrains Mono', 'Fira Code', monospace;
```

Import Inter from Google Fonts or bundle locally:
```html
<link rel="preconnect" href="https://fonts.googleapis.com">
<link href="https://fonts.googleapis.com/css2?family=Inter:wght@300;400;500;600;700&display=swap" rel="stylesheet">
```

### 3.2 Type scale

| Token | Size | Weight | Line height | Usage |
|---|---|---|---|---|
| `text-display` | 32px / 2rem | 700 | 1.2 | Page titles on landing/login |
| `text-h1` | 24px / 1.5rem | 600 | 1.3 | Main page heading |
| `text-h2` | 20px / 1.25rem | 600 | 1.35 | Section heading |
| `text-h3` | 16px / 1rem | 600 | 1.4 | Card heading, panel title |
| `text-body` | 15px / 0.9375rem | 400 | 1.6 | Default body text |
| `text-body-sm` | 13px / 0.8125rem | 400 | 1.5 | Secondary body, table cells |
| `text-label` | 12px / 0.75rem | 500 | 1.4 | Form labels, column headers |
| `text-caption` | 11px / 0.6875rem | 400 | 1.4 | Timestamps, helper text |
| `text-mono` | 13px / 0.8125rem | 400 | 1.5 | Code, IDs, verification codes |

### 3.3 Tailwind config

```ts
// tailwind.config.ts
fontSize: {
  'display': ['2rem',    { lineHeight: '1.2',  fontWeight: '700' }],
  'h1':      ['1.5rem',  { lineHeight: '1.3',  fontWeight: '600' }],
  'h2':      ['1.25rem', { lineHeight: '1.35', fontWeight: '600' }],
  'h3':      ['1rem',    { lineHeight: '1.4',  fontWeight: '600' }],
  'body':    ['0.9375rem', { lineHeight: '1.6', fontWeight: '400' }],
  'body-sm': ['0.8125rem', { lineHeight: '1.5', fontWeight: '400' }],
  'label':   ['0.75rem', { lineHeight: '1.4',  fontWeight: '500' }],
  'caption': ['0.6875rem', { lineHeight: '1.4', fontWeight: '400' }],
}
```

---

## 4. Spacing and layout

### 4.1 Spacing scale

Use Tailwind's default 4px base unit. Key values:

```
4px   — gap between icon and label, tight internal spacing
8px   — padding inside badges, chips
12px  — gap between form elements, table cell padding
16px  — card internal padding (compact), list item padding
20px  — standard form field height padding
24px  — card internal padding (default)
32px  — section spacing within a page
48px  — section spacing between major page areas
64px  — page top padding
```

### 4.2 Page layouts

**Admin back-office:**
```
┌─────────────────────────────────────────┐
│  Top bar (64px height)                  │
├──────────┬──────────────────────────────┤
│ Sidebar  │  Content area               │
│ (240px)  │  max-width: 1200px          │
│          │  padding: 32px              │
│          │                             │
└──────────┴──────────────────────────────┘
```

**Employee portal:**
```
┌─────────────────────────────────────────┐
│  Top bar (56px, minimal)                │
├─────────────────────────────────────────┤
│  Content — centered, max-width: 960px   │
│  padding: 40px 24px                     │
└─────────────────────────────────────────┘
```

**Exam taking (full-focus):**
```
┌─────────────────────────────────────────┐
│  Exam bar (48px) — title + timer        │
├───────────────────────┬─────────────────┤
│  Question area        │ Navigator panel │
│  max-width: 720px     │ (220px)         │
│  padding: 40px 48px   │ collapsible     │
└───────────────────────┴─────────────────┘
```

**Auth screens:**
```
Full-page navy background
Centered card: width 420px, padding 40px
Logo above card, centered
```

### 4.3 Breakpoints

```ts
screens: {
  'sm':  '640px',
  'md':  '768px',
  'lg':  '1024px',
  'xl':  '1280px',
}
```

On mobile (< 768px): sidebar collapses to bottom nav or hamburger menu; exam navigator becomes a modal drawer; card padding reduces to 16px.

---

## 5. Components

### 5.1 Buttons

**Variants:**

```tsx
// Primary — blue fill
<Button variant="primary">Save changes</Button>
// bg: --color-blue, text: white, hover: darken 8%

// Secondary — outlined
<Button variant="secondary">Cancel</Button>
// bg: transparent, border: --border-default, text: --text-primary, hover: --bg-subtle

// Danger — red fill
<Button variant="danger">Delete question</Button>
// bg: --color-danger, text: white

// Ghost — no border
<Button variant="ghost">View details</Button>
// bg: transparent, text: --color-blue, hover: --bg-info

// Navy — for dark surfaces
<Button variant="navy">Start exam</Button>
// bg: --color-navy, text: white
```

**Sizes:**

```tsx
size="sm"   // height 32px, px-3, text-label
size="md"   // height 40px, px-4, text-body-sm   ← default
size="lg"   // height 48px, px-6, text-body
```

**Rules:**
- Always include a loading state with a spinner for async actions
- Destructive actions always require a confirmation dialog before executing
- Icon buttons must have an `aria-label`
- Full-width buttons on mobile auth screens only

### 5.2 Form fields

```tsx
// Text input
<FormField label="Full name" required hint="As shown on company ID">
  <Input placeholder="e.g. Aizat Bekova" />
</FormField>

// Select
<FormField label="Department">
  <Select options={departments} />
</FormField>

// Textarea
<FormField label="Question stem" required characterCount maxLength={500}>
  <Textarea rows={4} />
</FormField>
```

Field anatomy:
```
Label (text-label, --text-primary, mb-1.5)
  [optional] Required indicator — red asterisk * inline after label
Input (height 40px, border --border-default, radius 8px, px-3)
  focus: border --border-focus, ring 3px --color-blue/20
  error: border --color-danger, ring 3px --color-danger/20
Hint text (text-caption, --text-muted, mt-1)
Error text (text-caption, --color-danger, mt-1) — replaces hint on error
```

Never show both hint and error simultaneously — error replaces hint.

### 5.3 Cards

```tsx
// Default surface card
<Card>
  <CardHeader title="Security awareness" subtitle="12 questions · 30 min" />
  <CardContent>...</CardContent>
  <CardFooter>...</CardFooter>
</Card>
```

Styles:
```css
background: var(--bg-surface);
border: 1px solid var(--border-default);
border-radius: 12px;
padding: 24px;
```

**Exam card (employee portal):**
- Status pill top-right
- Title h3
- Meta row: time limit + passing score + attempts
- Deadline row (warning color if < 48h)
- CTA button full-width at bottom

**Question card (question bank):**
- Type badge + difficulty badge top row
- Question stem (2-line clamp)
- Category breadcrumb + tags
- Status badge + locale coverage dots
- Last edited caption

### 5.4 Badges and status pills

```tsx
<Badge variant="success">Passed</Badge>
<Badge variant="danger">Failed</Badge>
<Badge variant="warning">Overdue</Badge>
<Badge variant="info">In progress</Badge>
<Badge variant="neutral">Draft</Badge>
<Badge variant="gold">Certified</Badge>  // certificates only
```

Anatomy: `text-caption, font-weight: 500, px-2 py-0.5, border-radius: 999px`

Colors (always use semantic pairs, never invert):

| Variant | Background | Text |
|---|---|---|
| success | `--bg-success` | `--color-success` |
| danger | `--bg-danger` | `--color-danger` |
| warning | `--bg-warning` | `--color-warning` |
| info | `--bg-info` | `--color-info` |
| neutral | `--bg-subtle` | `--text-secondary` |
| gold | `--color-gold-light` | `#7A5E1A` |

**Status mapping:**

| Entity | Status | Badge variant |
|---|---|---|
| Exam session | Not started | neutral |
| Exam session | In progress | info |
| Exam session | Passed | success |
| Exam session | Failed | danger |
| Exam session | Overdue | warning |
| Question | Draft | neutral |
| Question | Review | warning |
| Question | Active | success |
| Question | Archived | neutral (muted) |
| Exam | Draft | neutral |
| Exam | Active | success |
| Exam | Archived | neutral (muted) |
| User | Active | success |
| User | Deactivated | neutral (muted) |

### 5.5 Data tables

```tsx
<DataTable
  columns={columns}
  data={rows}
  pagination
  filters
  selectable     // optional bulk actions
  onRowClick     // optional row navigation
/>
```

Rules:
- Column headers: `text-label, --text-secondary, uppercase, letter-spacing: 0.04em`
- Cell text: `text-body-sm, --text-primary`
- Row height: 48px default, 40px compact
- Alternating row backgrounds: even rows `--bg-subtle`
- Hover: row background `--bg-subtle` + cursor pointer if clickable
- Sticky header on scroll
- Empty state: centered illustration + message + CTA if applicable
- Loading state: skeleton rows (not spinner)

### 5.6 Navigation — sidebar (admin)

```
┌──────────────────────┐
│ [Logo]  BilimBaga    │  — 64px header
├──────────────────────┤
│ ▸ Dashboard          │
│ ▸ Users              │  — nav items: 40px height, px-4
│   ▸ All users        │  — sub-items: 36px, px-8, shown when parent active
│   ▸ Departments      │
│ ▸ Questions          │
│ ▸ Exams              │
│ ▸ Reports            │
│                      │
│ ─────────────────    │
│ ▸ Audit log          │
│ ▸ Settings           │
├──────────────────────┤
│ [Avatar] Name        │  — 56px footer: user + logout
│          Role badge  │
└──────────────────────┘
```

Active nav item: `background: --bg-info, color: --color-blue, border-left: 3px solid --color-blue`
Inactive: `color: --text-secondary, hover: --bg-subtle`

### 5.7 Top bar (employee portal)

```
┌──────────────────────────────────────────────────┐
│ [Icon] BilimBaga    ·  My exams  My results      │  left
│                                 [Lang] [Avatar]  │  right
└──────────────────────────────────────────────────┘
```

Height 56px, background `--bg-surface`, border-bottom `--border-default`.

### 5.8 Exam timer

```tsx
<ExamTimer remainingSeconds={1800} warningThreshold={0.2} />
```

- Normal state: `--text-secondary` clock icon + `MM:SS`
- Warning state (≤ 20% remaining): `--color-warning` background pill, pulsing animation
- Critical state (≤ 5 min): `--color-danger`, bolder weight
- Announces remaining time via `aria-live="polite"` at 5-min intervals and at 1 min

### 5.9 Question navigator (exam taking)

Grid of numbered buttons, one per question:

```css
/* unanswered */
background: --bg-subtle;
border: 1px solid --border-default;

/* answered */
background: --bg-info;
border: 1px solid --color-blue;
color: --color-blue;

/* flagged */
background: --bg-warning;
border: 1px solid --color-warning;

/* current */
border: 2px solid --color-blue;
font-weight: 600;
```

### 5.10 Progress bar

```tsx
<ProgressBar value={7} max={20} label="7 of 20 answered" />
```

- Height 6px, border-radius 999px
- Track: `--border-default`
- Fill: `--color-blue`
- Animated fill transition: `transition: width 300ms ease`

### 5.11 Score display (result screen)

```tsx
<ScoreDial score={84} passing={80} passed={true} />
```

- Large circular dial (SVG), 160px diameter
- Score percentage centered, `text-display`
- Arc fill: `--color-success` if passed, `--color-danger` if failed
- Passing threshold marker on the arc
- "PASSED" / "FAILED" label below in matching color

### 5.12 Locale coverage indicator

Used in question editor and question bank list:

```tsx
<LocaleCoverage locales={['kk', 'ru', 'en']} covered={['kk', 'ru']} />
```

- Small colored dots per locale: filled = translated, hollow = missing
- Tooltip on hover: locale name + status

### 5.13 Modals and dialogs

```tsx
<Dialog title="Delete question" description="This cannot be undone." destructive>
  <Button variant="danger">Delete</Button>
  <Button variant="secondary">Cancel</Button>
</Dialog>
```

- Overlay: `rgba(0,0,0,0.4)`, backdrop-blur none (flat design)
- Dialog card: max-width 480px, padding 32px, border-radius 16px
- Destructive dialogs: danger icon in header, danger primary button
- Always trap focus, close on Escape, close on overlay click (except destructive)

### 5.14 Empty states

Every list and table must define an empty state:

```tsx
<EmptyState
  icon="ti-file-off"
  title="No questions yet"
  description="Add your first question to start building this exam."
  action={<Button variant="primary">Add question</Button>}
/>
```

- Centered vertically in container
- Icon: 48px, `--text-muted`
- Title: `text-h3, --text-primary`
- Description: `text-body-sm, --text-secondary`
- Action button: optional

---

## 6. Icons

> **Superseded (2026-10-10, FR-BB320 AC-5):** the app uses `lucide-react` (outline, stroke 1.5, 20px for UI icons), not Tabler. The Tabler instructions and imports below are historical; see `foundations.md` in this folder.

Use [Tabler Icons](https://tabler.io/icons) throughout — outline style only.

```tsx
import { IconShieldCheck, IconUsers, IconBook2 } from '@tabler/icons-react'

<IconShieldCheck size={20} stroke={1.5} />
```

Standard stroke width: `1.5` for UI icons, `1.25` for large decorative icons.

### Key icon mapping

| Concept | Icon |
|---|---|
| Security track | `IconShieldCheck` |
| Safety track | `IconHardHat` |
| Loyalty track | `IconStar` |
| Question | `IconQuestionMark` |
| Exam | `IconClipboardList` |
| Certificate | `IconCertificate` |
| Timer | `IconClock` |
| Score / grade | `IconChartBar` |
| Department | `IconBuildingCommunity` |
| User | `IconUser` |
| Users / bulk | `IconUsers` |
| Settings | `IconSettings` |
| Audit log | `IconHistory` |
| Language | `IconLanguage` |
| Flag (review) | `IconFlag` |
| Pass | `IconCircleCheck` |
| Fail | `IconCircleX` |
| Warning | `IconAlertTriangle` |
| Export | `IconDownload` |
| Import | `IconUpload` |
| Add | `IconPlus` |
| Edit | `IconPencil` |
| Delete | `IconTrash` |
| Archive | `IconArchive` |
| Search | `IconSearch` |
| Filter | `IconFilter` |

---

## 7. Motion and interaction

Keep animation minimal and purposeful. This is a corporate tool, not a marketing site.

```css
/* Standard transition for all interactive elements */
transition: background-color 150ms ease, border-color 150ms ease, color 150ms ease;

/* Entry animation for cards and modals */
@keyframes fade-in-up {
  from { opacity: 0; transform: translateY(8px); }
  to   { opacity: 1; transform: translateY(0); }
}
animation: fade-in-up 200ms ease;

/* Score dial arc fill */
transition: stroke-dashoffset 600ms ease-out;

/* Progress bar fill */
transition: width 300ms ease;
```

**Never animate:** table row contents, form validation messages, status badges.
**Respect:** `prefers-reduced-motion` — wrap all animations in the media query.

```css
@media (prefers-reduced-motion: reduce) {
  *, *::before, *::after {
    animation-duration: 0.01ms !important;
    transition-duration: 0.01ms !important;
  }
}
```

---

## 8. Two application modes

### 8.1 Admin back-office

Character: dense, efficient, data-rich. Assume the user knows the system.

- Information density: high. Tables show 15–20 rows default.
- Sidebar always visible on ≥ 1024px.
- Actions accessible via table row hover menus and bulk selection.
- No onboarding tooltips or guided tours.
- Keyboard shortcuts for frequent actions (documented in settings).

### 8.2 Employee exam portal

Character: calm, focused, reassuring. Assume the user may be anxious.

- Information density: low. One primary action per screen.
- No sidebar. Top bar only, minimal links.
- Generous whitespace around question text.
- Clear progress indication at all times.
- Confirmations before irreversible actions (submit exam).
- Friendly language: "Your answer has been saved" not "POST 200 OK".

### 8.3 Exam taking — full focus mode

This is the most sensitive screen. Additional rules:

- Remove all navigation elements except the exam bar.
- Question text: `text-h3` minimum, `--text-primary`, high contrast.
- Answer options: large tap targets minimum 44px height, generous spacing.
- Selected answer: `--bg-info` background, `--border-focus` left border 4px.
- Do not use red/green to indicate selection — it implies right/wrong before submission.
- Auto-save indicator always visible, never intrusive (small, bottom corner).
- Timer always visible, never obscures question content.

---

## 9. Accessibility

- All interactive elements keyboard-navigable; visible focus ring `3px solid --color-blue, offset 2px`
- Minimum touch target: 44×44px
- Color is never the sole indicator of meaning — always pair with icon, label, or pattern
- All images and icons have `alt` text or `aria-label`; purely decorative icons use `aria-hidden="true"`
- Form errors announced via `aria-live="polite"`
- Exam timer announced via `aria-live="polite"` at 5-min intervals and 1-min mark
- Modal dialogs trap focus and restore on close
- Table columns are `<th scope="col">`, row headers `<th scope="row">` where applicable
- Minimum contrast ratio: 4.5:1 for body text, 3:1 for large text and UI components (WCAG AA)

---

## 10. Tailwind config summary

```ts
// tailwind.config.ts
import type { Config } from 'tailwindcss'

export default {
  content: ['./src/**/*.{ts,tsx}'],
  darkMode: 'class',
  theme: {
    extend: {
      colors: {
        brand: {
          navy:       '#1B3A6B',
          blue:       '#2E6DB4',
          'blue-light': '#5AA0E0',
          'blue-pale':  '#D8E8F5',
          gold:       '#C8A84B',
          'gold-light': '#F0E4B8',
        },
      },
      fontFamily: {
        sans: ['Inter', 'Segoe UI', 'system-ui', 'sans-serif'],
        mono: ['JetBrains Mono', 'Fira Code', 'monospace'],
      },
      fontSize: {
        'display': ['2rem',      { lineHeight: '1.2',  fontWeight: '700' }],
        'h1':      ['1.5rem',    { lineHeight: '1.3',  fontWeight: '600' }],
        'h2':      ['1.25rem',   { lineHeight: '1.35', fontWeight: '600' }],
        'h3':      ['1rem',      { lineHeight: '1.4',  fontWeight: '600' }],
        'body':    ['0.9375rem', { lineHeight: '1.6',  fontWeight: '400' }],
        'body-sm': ['0.8125rem', { lineHeight: '1.5',  fontWeight: '400' }],
        'label':   ['0.75rem',   { lineHeight: '1.4',  fontWeight: '500' }],
        'caption': ['0.6875rem', { lineHeight: '1.4',  fontWeight: '400' }],
      },
      borderRadius: {
        'sm':   '6px',
        'md':   '8px',
        'lg':   '12px',
        'xl':   '16px',
        'pill': '999px',
      },
      spacing: {
        '18': '4.5rem',
        '22': '5.5rem',
      },
      boxShadow: {
        'card':  '0 1px 3px rgba(0,0,0,0.06), 0 1px 2px rgba(0,0,0,0.04)',
        'modal': '0 8px 32px rgba(0,0,0,0.12)',
        'focus': '0 0 0 3px rgba(46,109,180,0.25)',
      },
      maxWidth: {
        'content': '960px',
        'admin':   '1200px',
        'question': '720px',
      },
    },
  },
  plugins: [],
} satisfies Config
```

---

## 11. File and folder structure (frontend)

```
src/
├── assets/
│   ├── logo-light.svg
│   ├── logo-dark.svg
│   ├── icon.svg
│   └── favicon.svg
├── components/
│   ├── ui/               # shadcn/ui base components (Button, Input, Select…)
│   ├── common/           # shared app components (Badge, Card, DataTable, EmptyState…)
│   ├── exam/             # ExamCard, ExamTimer, QuestionNavigator, ScoreDial…
│   ├── questions/        # QuestionEditor, QuestionCard, LocaleCoverage…
│   └── layout/           # Sidebar, TopBar, ExamBar, AuthLayout…
├── pages/
│   ├── auth/             # Login, ChangePassword
│   ├── portal/           # EmployeePortal, ExamTaking, ExamResult, MyResults
│   └── admin/            # Dashboard, Users, Questions, Exams, Reports, Settings, AuditLog
├── hooks/                # useAuth, useExamSession, useTenantConfig…
├── lib/
│   ├── api.ts            # axios instance, interceptors
│   ├── auth.ts           # JWT helpers
│   └── i18n.ts           # react-i18next init
├── locales/
│   ├── kk.json           # Kazakh (default)
│   ├── ru.json           # Russian
│   └── en.json           # English
├── store/                # Zustand or React Query state
├── types/                # shared TypeScript interfaces
└── main.tsx
```

---

## 12. i18n conventions

- Library: `react-i18next`
- Default locale: `kk` (Kazakh)
- Supported: `kk`, `ru`, `en` (expandable)
- Locale files: flat JSON with dot-namespaced keys

```json
{
  "nav.dashboard": "Басты бет",
  "nav.questions": "Сұрақтар",
  "exam.start": "Бастау",
  "exam.timer.warning": "Уақыт аяқталуда",
  "status.passed": "Өтті",
  "status.failed": "Өтпеді"
}
```

Rules:
- Zero hardcoded user-visible strings in component code — all go through `t()`
- Date and number formatting via `Intl` API, not manual string concatenation
- RTL: add `dir={i18n.dir()}` to the `<html>` element; use CSS logical properties (`margin-inline-start` not `margin-left`) throughout

---

## 13. Logo SVG source

### Icon mark (use as `icon.svg`)

```svg
<svg width="56" height="56" viewBox="0 0 56 56" xmlns="http://www.w3.org/2000/svg">
  <title>BilimBaga icon</title>
  <!-- Hexagonal frame -->
  <polygon points="28,4 48,16 48,40 28,52 8,40 8,16" fill="#1B3A6B"/>
  <!-- Inner star pattern — 4 triangular facets -->
  <polygon points="28,14 33,22 28,20 23,22" fill="#2E6DB4"/>
  <polygon points="38,28 30,28 32,23 36,22" fill="#2E6DB4"/>
  <polygon points="28,42 23,34 28,36 33,34" fill="#2E6DB4"/>
  <polygon points="18,28 26,28 24,33 20,34" fill="#2E6DB4"/>
  <!-- Gold center dot -->
  <circle cx="28" cy="28" r="4.5" fill="#C8A84B"/>
</svg>
```

### Full logo light (`logo-light.svg`)

```svg
<svg width="200" height="56" viewBox="0 0 200 56" xmlns="http://www.w3.org/2000/svg">
  <title>BilimBaga</title>
  <!-- Mark -->
  <polygon points="28,4 48,16 48,40 28,52 8,40 8,16" fill="#1B3A6B"/>
  <polygon points="28,14 33,22 28,20 23,22" fill="#2E6DB4"/>
  <polygon points="38,28 30,28 32,23 36,22" fill="#2E6DB4"/>
  <polygon points="28,42 23,34 28,36 33,34" fill="#2E6DB4"/>
  <polygon points="18,28 26,28 24,33 20,34" fill="#2E6DB4"/>
  <circle cx="28" cy="28" r="4.5" fill="#C8A84B"/>
  <!-- Wordmark -->
  <text x="62" y="24"
    font-family="Inter, Arial, sans-serif"
    font-weight="700" font-size="22"
    fill="#1B3A6B" letter-spacing="-0.3">Bilim</text>
  <text x="62" y="46"
    font-family="Inter, Arial, sans-serif"
    font-weight="300" font-size="17"
    fill="#2E6DB4" letter-spacing="3.5">BAGA</text>
</svg>
```

### Full logo dark (`logo-dark.svg`)

```svg
<svg width="200" height="56" viewBox="0 0 200 56" xmlns="http://www.w3.org/2000/svg">
  <title>BilimBaga</title>
  <!-- Mark -->
  <polygon points="28,4 48,16 48,40 28,52 8,40 8,16" fill="#2E6DB4"/>
  <polygon points="28,14 33,22 28,20 23,22" fill="#5AA0E0"/>
  <polygon points="38,28 30,28 32,23 36,22" fill="#5AA0E0"/>
  <polygon points="28,42 23,34 28,36 33,34" fill="#5AA0E0"/>
  <polygon points="18,28 26,28 24,33 20,34" fill="#5AA0E0"/>
  <circle cx="28" cy="28" r="4.5" fill="#C8A84B"/>
  <!-- Wordmark -->
  <text x="62" y="24"
    font-family="Inter, Arial, sans-serif"
    font-weight="700" font-size="22"
    fill="#FFFFFF" letter-spacing="-0.3">Bilim</text>
  <text x="62" y="46"
    font-family="Inter, Arial, sans-serif"
    font-weight="300" font-size="17"
    fill="#C8A84B" letter-spacing="3.5">BAGA</text>
</svg>
```

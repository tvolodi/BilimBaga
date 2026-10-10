# BilimBaga design foundations

> Source: the owner's Design System (Claude design-system artifact, 58 components with live previews, synced from `main@2a4ad2d` on 2026-10-09). This file is its brand book, kept in the repo so agents and developers read it as a project rule. Machine-readable tokens: `tokens.json` in this folder. Where this file conflicts with `bilimbaga_design_system.md`, THIS file wins (see `README.md` for the order of authority).

BilimBaga is a corporate exam and learning platform with two faces: a dense admin back-office and a calm, focused employee exam portal. Build in React with Tailwind and shadcn-style primitives; every value below is a token, never a raw hex.

## Content fundamentals

- Tagline: "Corporate Learning & Assessment". Name: **BilimBaga** (Kazakh *bilim* "knowledge" + *baға* "grade"), written as one word, capital B twice.
- Kazakh (`kk`) is the default locale, then `ru` and `en`. Put no user-visible string in component code; route it through `t()`. Format dates and numbers with `Intl`. Use logical CSS properties (`margin-inline-start`), not left/right.
- Employee portal copy is friendly and reassuring: "Your answer has been saved", never "POST 200 OK". Admin copy is terse and assumes an expert.
- No emoji. Status is always a word plus colour.

## Visual foundations

- **Colour.** `color-navy` for sidebar, auth panel and dark surfaces; `color-blue` (alias `color-primary`) for buttons, links, active nav and focus; `color-gold` only for certificates and achievement badges, never for an action; the logo carries its own colours and does not use the palette tokens. Status colours come in pairs: text `color-success` on `color-bg-success`, likewise warning, danger and info. Never invert a pair.
- **Surfaces.** Page `color-bg-page`, cards `color-bg-surface` with a 1px `color-border-default` and `shadow-card`, hover and alternate rows `color-bg-subtle`. Dark theme swaps the same names; switch with `class="dark"` on `<html>` (the code's mechanism; the original design system text said `data-theme="dark"`). The theme switch is specified in FR-BB321 (light, dark, system; stored in `localStorage` key `bb-theme`).
- **Type.** Inter (`font-sans`, loaded from Google Fonts, 300 to 700), JetBrains Mono for codes and IDs. Page title `text-h1`, section `text-h2`, card title `text-h3`, body `text-body`, tables `text-body-sm`, labels `text-label`, helper text `text-caption`.
- **Spacing.** 4px base: `space-6` (24px) card padding, `space-8` between sections in a page, `space-12` between major areas. Admin content is `container-admin` wide with `space-8` padding; the portal is `container-content`; an exam question column is `container-question`.
- **Radii.** `radius-md` buttons and inputs, `radius-lg` cards, `radius-xl` dialogs, `radius-pill` badges and progress bars.
- **Focus.** A 2px ring in `color-ring` with a 2px offset on every control; fields add `shadow-focus`. Tap targets are at least 44px in exam mode.
- **Motion.** 150ms colour transitions only; `fade-in-up` 200ms for cards and dialogs; the score dial arc eases 600ms. Never animate table rows, validation messages or badges. Honour `prefers-reduced-motion`.
- **Exam mode.** Remove all navigation except the exam bar. Selected answers use `color-bg-info` with a 4px `color-border-focus` left edge; never green or red, which would imply right or wrong before submission.

## Iconography

The design doc names Tabler outline icons at stroke 1.5, but the shipped frontend imports `lucide-react`. Use lucide in code; keep stroke 1.5 and 20px for UI icons. Icons that carry meaning get an `aria-label`; decorative ones `aria-hidden`.

## Logos

`assets/Logos/`: the mark is `logo-mark.svg` (shield, graduation cap, open book, orange check mark and cog; colours #204f77, #599bbf, #eda136; transparent background). In UI use `LogoMark`, which draws the same artwork. The name has no wordmark file: set "BilimBaga" beside the mark in Inter 700. On navy or dark grounds put the mark on a white tile (`LogoMark plate`), because its dark blue parts vanish there. `apple-touch-icon.png` (180x180) is for iOS home screens; the browser-tab icon is the SVG. Never stretch, recolour or crop the mark; 24px high at minimum.

## Components

The bundle (`window.BilimBaga`, React 18) holds 59 ready-made components. Copy the example from a component's README, change the data and handlers, and keep the structure and class names: every part is already wired to the tokens and works in both themes. Compose screens from these pieces rather than writing new markup.

- **Foundations:** Icon, LogoMark.
- **Actions and status:** Button; Badge, StatusBadge (entity + status to variant), RoleBadge, Avatar.
- **Surfaces:** Card (+ CardHeader, CardTitle, CardContent, CardFooter), KpiCard (+ KpiRow).
- **Forms:** FormField (label, hint, error, counter; wire every control through it), Input, Label, Textarea, Select, Checkbox, Switch, RadioGroup, SearchInput, PasswordInput, FormSection (+ FormActions), LoginForm.
- **Feedback:** Alert, Spinner, Skeleton, EmptyState, ProgressBar.
- **Overlays:** Dialog (+ DialogBody, DialogFooter), ConfirmDialog, Sheet (drawer for create/edit forms), Popover, DropdownMenu (row actions).
- **Navigation:** Sidebar, PortalTopBar, AdminTopBar, ExamBar, Breadcrumb, Tabs, Stepper, Pagination, PageHeader.
- **Layouts:** AdminLayout, PortalLayout, AuthLayout, ExamLayout.
- **Data:** Table (primitives), DataTable (sort, select, row actions, paging, skeleton, empty), BulkBar, FilterBar, TreeView, TreeSelect, MasterDetail (+ DetailHeader, DescriptionList).
- **Exam:** ExamCard, CountdownTimer, QuestionNavigator, AnswerOption, QuestionCard, LocaleCoverage, ScoreDial, PassFailBanner.

Screen recipes: an admin list page is `AdminLayout` > `PageHeader` > `FilterBar` > `BulkBar` > `DataTable`, with create/edit in a `Sheet` and deletes behind a destructive `ConfirmDialog`. A hierarchy screen (departments, categories) is `MasterDetail` with a `TreeView` search in the master pane, or a `TreeView` beside a `DetailHeader` panel. A form is `FormSection`s of `FormField`s ending in `FormActions`. The employee portal is `PortalLayout` with `ExamCard`s; taking an exam is `ExamLayout` with `AnswerOption`s, `QuestionNavigator` and `CountdownTimer`. Sign-in is `AuthLayout` + `LoginForm`.

Status mapping (StatusBadge): exam session Not started = neutral, In progress = info, Passed = success, Failed = danger, Overdue = warning; question and exam Draft = neutral, Review = warning, Active = success, Archived = neutral; Certified = gold.

Rules that apply to every screen: all text goes through `t()`; sorting and paging are server-side, so pass DataTable the current page; show Skeleton rows while loading and an EmptyState when empty; confirm every destructive action; give icon-only buttons an `aria-label`; never use red/green to mark an exam answer as selected.

## Accessibility

Body text meets 4.5:1 in both themes. The source values of `color-text-muted` (3.06:1 on white light, 2.90:1 on surface dark) and `color-gold` on white (2.3:1) failed; FR-BB320 AC-1 corrected them: muted text is `#5f6b7d` light and `#8fa0b4` dark (4.81 to 5.40:1 and 5.16 to 6.78:1 on every ground), the dark destructive foreground is `#0f1623` (5.14:1 on the dark destructive fill, white gave 3.52:1). `color-gold` is never used as text; gold badge text is `#7a5e1a` on `color-gold-light` (4.79:1).

## Not synced

Built from `tvolodi/BilimBaga@2a4ad2d` (`docs/design-system/` and `frontend/src`). The components are hand-written to match the repo's `ui/*`, `admin/*`, `dashboard/*`, `results/*`, `EmployeePortal/*` and `ExamTaking/*` components and the design doc, not built from the repo. They are plain React with `bb-*` classes in `components/bundle.css` rather than Tailwind and Radix, so the markup and tokens carry over but the code does not run unchanged in the app. No font files: Inter and JetBrains Mono load from Google Fonts. The logos are the files you uploaded to this system, not the repo's older hexagon logos. Not brought in: the date-time picker, the colour picker, the logo uploader, grading screens and other feature-specific components. Differences from the repo: the repo's Sidebar uses gray-900 (this follows the doc's navy); the repo's Badge uses Tailwind green and yellow for success and warning (this uses the doc's semantic pairs); the repo's Dialog and Sheet are plain divs, here they add `aria-labelledby` and focus on open; the doc names Tabler icons but the code imports `lucide-react`. `TenantProvider` overrides `color-primary` and `color-accent` per tenant at runtime. The `fade-in-up` and pulse keyframes are not tokens. Overlays use `position: fixed` and are not portalled, so render them near the root of your page.

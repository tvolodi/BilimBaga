# BilimBaga design system: rules for every UI change

Read this page before any frontend work (pages, components, forms, layouts, locale strings in the UI). It is the project rule for how BilimBaga looks and behaves; `docs/requirements/FR-BB320.Design-system-conformance.md` makes it enforceable.

## Order of authority
1. `foundations.md` and `tokens.json` (the owner's design system, synced 2026-10-09): colour, type, spacing, radii, motion, exam mode, accessibility, screen recipes, status mapping.
2. `bilimbaga_design_system.md`: component details, Tailwind config summary, folder structure, i18n conventions. Where it conflicts with `foundations.md`, `foundations.md` wins (known case: icons are `lucide-react`, stroke 1.5, 20px, not Tabler).
3. `frontend/src/index.css` and `components/ui/*` are the implementation. If they disagree with 1, the code is wrong (see FR-BB320 for the known differences).

The owner also keeps the design system as a Claude artifact with live component previews (private; ask the owner for the link). It is a visual reference and a specification, not a code source: its components are plain React with `bb-*` classes and do not run unchanged in the app (the app uses Tailwind and shadcn/ui). Never copy its `bundle.js` into `frontend/`.

## One-page checklist (every UI change)
1. **Tokens only.** No raw hex, `rgb()` or Tailwind arbitrary colours (`bg-[#...]`) in `.tsx` or `.ts`; use the token classes (`bg-background`, `text-foreground`, `text-muted-foreground`, status pairs). New colours go into `index.css` and `tokens.json` first.
2. **Primitives only.** Use `components/ui/*` (Button, Input, Select, Dialog, Sheet, Table, Badge, Card, Label, Popover, Skeleton). No raw `<button>`, `<input>`, `<select>`, `<textarea>` outside `components/ui`. Compose screens from the recipes in `foundations.md` (list page, hierarchy page, form, portal, exam, sign-in).
3. **Forms.** Every control goes through label, hint and error with proper `id`, `aria-describedby`, `aria-invalid`. Server-side sorting and paging. Primary action on the right, destructive actions behind a confirm dialog.
4. **States.** Every data view has loading (Skeleton), empty (EmptyState) and error states. Buttons show a pending state; no double submit.
5. **Colour semantics.** Blue is for actions, gold only for certificates, status uses the success, warning, danger, info pairs (text colour on its own background, never inverted). Never red or green for a selected exam answer.
6. **Text.** All strings through `t()` in `kk`, `ru`, `en`; Kazakh and Russian are longer than English, so check overflow at 375px width. Body text must reach 4.5:1 contrast in both themes; `text-muted-foreground` for secondary text, never a lighter grey.
7. **Layout.** `container-admin` 1200px, `container-content` 960px, `container-question` 720px; 4px spacing scale; radii `md` for controls, `lg` cards, `xl` dialogs, pill for badges. Logical CSS properties.
8. **Focus and touch.** 2px ring with 2px offset on every control; tap targets at least 44px in exam mode; icon-only buttons have `aria-label`; dialogs label themselves (`aria-labelledby`) and restore focus.
9. **Motion.** 150ms colour transitions, 200ms fade for cards and dialogs; nothing on table rows, validation messages or badges; honour `prefers-reduced-motion`.
10. **Exam mode.** No navigation except the exam bar; selected answers use the info colour with a left edge.

## How to check a change
- `npm test` runs vitest and the i18n check; FR-BB320 adds `npm run check:design` (colours and primitives) and the design gallery screenshots.
- Look at the page in light and dark theme, at 375, 768 and 1280px, in `kk`, `ru` and `en`.
- A change that needs a new token, component or exception is a design-system change: update `foundations.md` or `tokens.json` in the same PR and say so in the PR text. Exceptions in code use `// design-ok: <reason>`.

## Who maintains what
- The owner: the design system itself (the artifact, and approval of new tokens and components).
- BA: this folder, `FR-BB320`, frontend sections of requirements, review of UI requirements against this page.
- Developers: follow the checklist; fix differences between code and the design system.
- Architect (event-driven, no loop): decisions on trade-offs between the design system and implementation.

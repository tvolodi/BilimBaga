# FR-BB26 — Frontend: Question Editor

## Metadata
| Field | Value |
|-------|-------|
| ID | FR-BB26 |
| Phase | 2 — Content Management |
| Priority | 2 |
| Status | Draft |
| Depends On | FR-BB23, FR-BB24 |

## Description
A rich two-panel page for creating and editing individual questions. The left panel contains the authoring form (metadata, per-locale text fields, answer options); the right panel renders a live preview of the question exactly as an exam taker would see it. Language tabs at the top allow translators to switch locale context; a coverage indicator on each tab shows whether the translation is complete or missing. Status workflow transitions are accessible via contextual action buttons without leaving the editor.

## Acceptance Criteria
- [ ] AC-1: The editor renders a two-panel layout (form left, preview right) on viewports ≥ 1024 px; on narrower viewports the preview collapses into a toggle-able drawer; both panels are visible simultaneously on desktop without horizontal scroll.
- [ ] AC-2: Language tabs display the tenant's configured locales; the default locale tab is first and cannot be removed; each non-default tab shows a ✓ (complete) or ✗ (missing) coverage indicator based on whether a translation row exists and has a non-empty stem.
- [ ] AC-3: The question type selector (`single`, `multiple`, `truefalse`, `likert`, `shorttext`) dynamically adjusts the answer options section: `truefalse` locks to exactly two options with preset labels; `shorttext` hides the options section entirely; `likert` shows `weight` (numeric) and `polarity` (`positive`/`negative`) fields per option.
- [ ] AC-4: For `single` and `multiple` types, answer options support drag-to-reorder (using a drag handle); the `sort_order` values are sent in the updated order on save; a "Mark correct" toggle appears per option (radio for `single`, checkbox for `multiple`).
- [ ] AC-5: The difficulty selector (`easy`, `medium`, `hard`) and category tree picker (collapsible tree sourced from `GET /api/v1/categories`) are present in the metadata section; the category picker shows the full ancestor path of the selected category.
- [ ] AC-6: The tag input field provides autocomplete sourced from `GET /api/v1/tags`; users can add existing tags or type a new tag name (which calls `POST /api/v1/tags` before associating); tags are displayed as dismissible chips.
- [ ] AC-7: The status workflow section displays the current status badge and conditionally renders action buttons: draft → "Submit for Review"; review → "Approve" (admin+) and "Return to Draft"; active → "Archive"; each button calls `POST /api/v1/questions/:id/status` and uses optimistic UI update via React Query mutation with rollback on error.
- [ ] AC-8: Auto-save fires every 30 seconds when the form is dirty (has unsaved changes); it calls `PUT /api/v1/questions/:id` silently; a subtle "Saved" / "Saving…" indicator is shown in the toolbar; auto-save is cancelled when the user manually saves or navigates away.
- [ ] AC-9: The explanation field (per-locale) is collapsible; it is collapsed by default and expands when the user clicks "Add explanation"; it supports plain text input with a character counter (max 1000 chars).
- [ ] AC-10: All user-visible strings (labels, placeholders, tooltips, status names, error messages) are sourced from `src/locales/{locale}.json`; zero hardcoded English strings appear in component code; the editor is fully usable in Kazakh (`kk`), Russian (`ru`), and English (`en`).

## Technical Specification

### Frontend Components

#### Page: `QuestionEditorPage`
- Route: `/questions/new` and `/questions/:id/edit`
- Fetches: `GET /api/v1/questions/:id` (edit mode), `GET /api/v1/categories`, `GET /api/v1/tags`
- Mutations: `POST /api/v1/questions`, `PUT /api/v1/questions/:id`, `POST /api/v1/questions/:id/status`, `PUT /api/v1/questions/:id/translations/:locale`

```
QuestionEditorPage
├── EditorToolbar
│   ├── StatusBadge
│   ├── StatusActionButtons
│   ├── AutoSaveIndicator
│   └── SaveButton / CancelButton
├── LocaleTabs                       ← shadcn Tabs
│   └── LocaleTab (per configured locale)
│       └── CoverageIndicator (✓/✗)
└── EditorLayout (two-column)
    ├── EditorFormPanel (left)
    │   ├── MetadataSection
    │   │   ├── QuestionTypeSelector  ← shadcn Select
    │   │   ├── DifficultySelector    ← shadcn Select
    │   │   ├── CategoryTreePicker    ← custom popover + tree
    │   │   └── TagInput              ← autocomplete + chips
    │   ├── TranslationSection (active locale)
    │   │   ├── StemTextarea
    │   │   └── ExplanationCollapsible
    │   └── AnswerOptionsSection
    │       ├── AnswerOptionList      ← react-beautiful-dnd or @dnd-kit
    │       │   └── AnswerOptionRow
    │       │       ├── DragHandle
    │       │       ├── OptionTextInput (active locale)
    │       │       ├── CorrectToggle (radio/checkbox)
    │       │       └── LikertFields (weight + polarity, conditional)
    │       └── AddOptionButton
    └── PreviewPanel (right)
        └── QuestionPreview
            ├── StemRenderer
            └── AnswerOptionsRenderer
```

#### State Management
- React Query (`useQuery`) for all read data (question, categories, tags).
- React Query (`useMutation`) for all writes; cache invalidation on success: `['questions', id]`.
- Local `useForm` state (React Hook Form or controlled state) for form fields; dirty tracking triggers auto-save.
- Auto-save implemented with `useEffect` + `setInterval`; clears on unmount and on explicit save.

#### Key i18n Keys (representative)
```json
{
  "questionEditor.title.new": "Жаңа сұрақ",
  "questionEditor.title.edit": "Сұрақты өңдеу",
  "questionEditor.locale.coverage.complete": "Аударма толық",
  "questionEditor.locale.coverage.missing": "Аударма жоқ",
  "questionEditor.type.single": "Бір дұрыс жауап",
  "questionEditor.type.multiple": "Бірнеше дұрыс жауап",
  "questionEditor.type.truefalse": "Рас / Жалған",
  "questionEditor.type.likert": "Лайкерт шкаласы",
  "questionEditor.type.shorttext": "Қысқа жауап",
  "questionEditor.difficulty.easy": "Оңай",
  "questionEditor.difficulty.medium": "Орташа",
  "questionEditor.difficulty.hard": "Қиын",
  "questionEditor.status.draft": "Жоба",
  "questionEditor.status.review": "Тексерілуде",
  "questionEditor.status.active": "Белсенді",
  "questionEditor.status.archived": "Мұрағатталған",
  "questionEditor.action.submitForReview": "Тексеруге жіберу",
  "questionEditor.action.approve": "Бекіту",
  "questionEditor.action.archive": "Мұрағаттау",
  "questionEditor.autosave.saving": "Сақталуда…",
  "questionEditor.autosave.saved": "Сақталды",
  "questionEditor.explanation.add": "Түсіндірме қосу",
  "questionEditor.addOption": "Жауап нұсқасын қосу"
}
```

## Notes
- The live preview in the right panel reads directly from the local form state (no API call) to ensure zero-latency updates as the user types.
- Drag-to-reorder should use `@dnd-kit/sortable` (already lighter than react-beautiful-dnd and maintained); wrap `AnswerOptionList` in `DndContext` + `SortableContext`.
- The category tree picker does not use a native `<select>` but a Popover with a recursive `CategoryTreeNode` component; selected path is displayed as a breadcrumb.
- Auto-save must debounce rapid keystrokes (500 ms) before starting the 30 s interval to avoid saving mid-word.
- On mobile the two-panel layout should stack vertically (form first, preview second) using a CSS grid `grid-template-columns: 1fr` breakpoint at `lg`.
- Optimistic status transitions must revert the badge UI if the mutation fails (React Query `onError` → `queryClient.setQueryData` with the previous snapshot).

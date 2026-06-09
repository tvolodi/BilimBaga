# FR-BB26 — Frontend: Question Editor

## Metadata
| Field | Value |
|-------|-------|
| ID | FR-BB26 |
| Phase | 2 — Content Management |
| Priority | 2 |
| Status | uat-verified |
| Depends On | FR-BB21, FR-BB23, FR-BB24 |

## Description
A rich two-panel page for creating and editing individual questions. The left panel contains the authoring form (metadata, per-locale text fields, answer options); the right panel renders a live preview of the question exactly as an exam taker would see it. Language tabs at the top allow translators to switch locale context; a coverage indicator on each tab shows whether the translation is complete or missing. Status workflow transitions are accessible via contextual action buttons without leaving the editor.

## Scope

| Layer | Items |
|-------|-------|
| Frontend | New page `QuestionEditorPage` (`src/pages/admin/questions/QuestionEditorPage.tsx`); components: `LocaleTabBar`, `StemSection`, `AnswerOptionsSection`, `CoverageIndicator`, `TagCombobox` |
| Router | New routes `questions/new` and `questions/:id/edit` nested in `App.tsx` under `/admin` |
| API hooks | New hooks: `useQuestion`, `useCreateQuestion`, `useUpdateQuestion`, `useUpdateQuestionStatus`, `useUpdateTranslation`, `useCreateTag` added to `frontend/src/api/questions.ts` |
| i18n | New key namespace `questionEditor.*` in `src/locales/{kk,ru,en}.json` |
| Backend | No changes — consumes FR-BB23 and FR-BB24 endpoints |
| Database | No changes |

## Acceptance Criteria
- [ ] AC-1: The editor renders a two-panel layout (form left, preview right) on viewports ≥ 1024 px; on narrower viewports the preview collapses into a toggle-able drawer; both panels are visible simultaneously on desktop without horizontal scroll.
- [ ] AC-2: Language tabs display the tenant's configured locales; the default locale tab is first and cannot be removed; each non-default tab shows a ✓ (complete) or ✗ (missing) coverage indicator based on whether a translation row exists and has a non-empty stem.
- [ ] AC-3: The question type selector (`single`, `multiple`, `truefalse`, `likert`, `shorttext`) dynamically adjusts the answer options section: `truefalse` locks to exactly two options with preset labels; `shorttext` hides the options section entirely; `likert` shows `weight` (numeric) and `polarity` (`positive`/`negative`) fields per option.
- [ ] AC-4: For `single` and `multiple` types, answer options support drag-to-reorder (using a drag handle); the `sort_order` values are sent in the updated order on save; a "Mark correct" toggle appears per option (radio for `single`, checkbox for `multiple`).
- [ ] AC-5: The difficulty selector (`easy`, `medium`, `hard`) and category tree picker (collapsible tree sourced from `GET /api/v1/categories`) are present in the metadata section; the category picker shows the full ancestor path of the selected category.
- [ ] AC-6: The tag input field provides autocomplete sourced from `GET /api/v1/tags`; users can add existing tags or type a new tag name (which calls `POST /api/v1/tags` before associating); tags are displayed as dismissible chips.
- [ ] AC-7: The status workflow section displays the current status badge and conditionally renders action buttons: draft → "Submit for Review" (→review); review → "Approve" (→active, requires department_admin or super_admin role); active → "Archive" (→archived); each button calls `POST /api/v1/questions/:id/status` and uses optimistic UI update via React Query mutation with rollback on error.
- [ ] AC-8: Auto-save fires every 30 seconds when the form is dirty (has unsaved changes); it calls `PUT /api/v1/questions/:id` silently; a subtle "Saved" / "Saving…" indicator is shown in the toolbar; auto-save is cancelled when the user manually saves or navigates away. **In create mode (`/admin/questions/new`, no question ID exists), auto-save is disabled.** The editor toolbar shows a "Save Draft" button (`questionEditor.action.saveDraft`). Auto-save activates only after the first explicit save creates the question and the browser navigates to `/admin/questions/:id/edit`.
- [ ] AC-9: The explanation field (per-locale) is collapsible; it is collapsed by default and expands when the user clicks "Add explanation"; it supports plain text input with a character counter (max 1000 chars).
- [ ] AC-10: All user-visible strings (labels, placeholders, tooltips, status names, error messages) are sourced from `src/locales/{locale}.json`; zero hardcoded English strings appear in component code; the editor is fully usable in Kazakh (`kk`), Russian (`ru`), and English (`en`).
- [ ] AC-11: When `POST /api/v1/questions` or `PUT /api/v1/questions/:id` returns a non-2xx response, a toast notification displays the server error message (`questionEditor.error.saveFailed`). Auto-save is suspended for the current dirty session. The form retains its current state and the Save button remains enabled.

## Technical Specification

### Frontend Components

#### Page: `QuestionEditorPage`
- Route: `/admin/questions/new` (full path) and `/admin/questions/:id/edit` (full path)
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

#### TypeScript Interfaces

```typescript
// GET /api/v1/questions/:id response shape
interface QuestionDetail {
  id: string;           // UUID v4
  type: 'single' | 'multiple' | 'truefalse' | 'shorttext' | 'likert';
  difficulty: 'easy' | 'medium' | 'hard';
  status: 'draft' | 'review' | 'active' | 'archived';
  category_id: string;  // UUID v4
  category_name: string;
  default_locale: string;
  version: number;
  parent_id: string | null; // UUID v4
  locale_coverage: string[];
  translations: Record<string, QuestionTranslation>; // keyed by locale code
  answer_options: AnswerOption[];
  tag_ids: string[];        // UUID v4 array
  created_by: string;       // UUID v4
  created_by_name: string;
  created_at: string;       // UTC ISO 8601
  updated_at: string;       // UTC ISO 8601
}

interface QuestionTranslation {
  stem: string;
  explanation: string;
}

interface AnswerOption {
  id: string;               // UUID v4
  sort_order: number;
  is_correct: boolean;
  likert_weight: number | null;
  likert_polarity: string | null;
  translations: Record<string, AnswerOptionTranslation>; // keyed by locale
}

interface AnswerOptionTranslation {
  body: string;
}

// Create payload
interface QuestionCreatePayload {
  type: string;
  difficulty: string;
  category_id: string;     // UUID v4
  default_locale: string;
  stem: string;            // for default locale
  explanation?: string;
  answer_options?: AnswerOptionCreatePayload[];
}

interface AnswerOptionCreatePayload {
  sort_order: number;
  is_correct: boolean;
  likert_weight?: number;
  likert_polarity?: string;
  body: string; // for default locale
}

// Update payload (PUT /api/v1/questions/:id)
interface QuestionUpdatePayload {
  difficulty?: string;
  category_id?: string;
  translations?: Record<string, Partial<QuestionTranslation>>;
  answer_options?: AnswerOptionUpdatePayload[];
}

interface AnswerOptionUpdatePayload extends AnswerOptionCreatePayload {
  id?: string; // UUID v4 — if present, updates existing option; if absent, creates new
}
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
  "questionEditor.action.saveDraft": "Жобаны сақтау",
  "questionEditor.autosave.saving": "Сақталуда…",
  "questionEditor.autosave.saved": "Сақталды",
  "questionEditor.explanation.add": "Түсіндірме қосу",
  "questionEditor.addOption": "Жауап нұсқасын қосу",
  "questionEditor.error.saveFailed": "Сақтау кезінде қате орын алды"
}
```

## Technical Notes
- The live preview in the right panel reads directly from the local form state (no API call) to ensure zero-latency updates as the user types.
- Drag-to-reorder should use `@dnd-kit/sortable` (already lighter than react-beautiful-dnd and maintained); wrap `AnswerOptionList` in `DndContext` + `SortableContext`.
- The category tree picker does not use a native `<select>` but a Popover with a recursive `CategoryTreeNode` component; selected path is displayed as a breadcrumb.
- Auto-save must debounce rapid keystrokes (500 ms) before starting the 30 s interval to avoid saving mid-word.
- On mobile the two-panel layout should stack vertically (form first, preview second) using a CSS grid `grid-template-columns: 1fr` breakpoint at `lg`.
- Optimistic status transitions must revert the badge UI if the mutation fails (React Query `onError` → `queryClient.setQueryData` with the previous snapshot).
- Routes are registered in `App.tsx` as `path='questions/new'` and `path='questions/:id/edit'` (relative paths) nested inside the `/admin` route group, producing full paths `/admin/questions/new` and `/admin/questions/:id/edit`. Required roles: `super_admin`, `department_admin`, `examiner`.

## Out of Scope

Question bank list page (FR-BB27), bulk import/export (FR-BB25), standalone translation management page (FR-BB24 UI), collaborative multi-user editing, mobile-native layout, exam assignment (Phase 3), and tag CRUD management page are out of scope for this requirement.

## Test Strategy

Unit tests for each named component: `LocaleTabBar`, `AnswerOptionsSection` (all 5 question types), `CoverageIndicator`. React Testing Library integration tests for AC-8 (auto-save timer with fake timers; must cover both create-mode disabled path and edit-mode 30s timer), AC-4 (answer option add/reorder/delete), AC-11 (save failure toast + auto-save suspension). E2E test for the create-and-save happy path and edit round-trip.

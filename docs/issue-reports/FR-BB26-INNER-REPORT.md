# FR-BB26 Inner Report — Frontend Question Editor

**Date**: 2026-05-15  
**Status**: implemented  
**Pipeline**: A — Requirement Implementation

---

## Summary

Implemented the `QuestionEditorPage` for creating and editing individual questions, including all supporting API hooks, routing, i18n strings, and drag-to-reorder answer options.

---

## What Was Implemented

### 1. API Hooks (`frontend/src/api/questions.ts`)

Added exported TypeScript interfaces:
- `QuestionTranslation`, `AnswerOptionTranslation`, `AnswerOption`, `QuestionDetail`
- `QuestionCreatePayload`, `QuestionUpdatePayload`, `AnswerOptionCreatePayload`, `AnswerOptionUpdatePayload`

Added new React Query hooks:
- `useQuestion(id)` — GET /api/v1/questions/:id, enabled only when id is non-null, staleTime: 0
- `useCreateQuestion()` — POST /api/v1/questions, invalidates `['questions']`
- `useUpdateQuestion()` — PUT /api/v1/questions/:id, invalidates `['questions', id]` and `['questions']`
- `useUpdateQuestionStatus()` — POST /api/v1/questions/:id/status, optimistic update with rollback, invalidates both
- `useCreateTag()` — POST /api/v1/tags, invalidates `['tags']`

### 2. `@dnd-kit` Packages

Installed `@dnd-kit/core`, `@dnd-kit/sortable`, `@dnd-kit/utilities` for drag-to-reorder answer options.

### 3. `QuestionEditorPage` (`frontend/src/pages/admin/questions/QuestionEditorPage.tsx`)

Two-panel layout (form left, live preview right) with:
- **Create mode** (`/admin/questions/new`): empty form, "Save Draft" button, no auto-save
- **Edit mode** (`/admin/questions/:id/edit`): loads question, auto-save every 30s when dirty, shows "Saving…"/"Saved" indicator
- **LocaleTabBar**: EN/KK/RU tabs with ✓/○ coverage indicators per tab based on stem content
- **Type selector**: only editable in create mode, locked in edit mode
- **Difficulty selector**, **CategoryPicker** (popover with full hierarchy), **TagCombobox** (autocomplete + inline tag creation)
- **StemTextarea** and collapsible **Explanation** with 1000-char counter per locale
- **AnswerOptionsSection** per type:
  - `single`/`multiple`: drag-to-reorder with DndKit, radio/checkbox correct toggles, add/delete
  - `truefalse`: locked 2 options, correct toggle only
  - `shorttext`: panel hidden
  - `likert`: weight (number) + polarity (positive/negative) fields per option
- **Status workflow buttons**: Submit for Review (draft→review), Approve (review→active), Archive (active→archived)
- **QuestionPreview**: live preview panel reading from local form state (zero-latency)
- Notification banner for save errors and status transition results

### 4. Routes (`frontend/src/App.tsx`)

Added:
- `path="questions/new"` → `<RequireRole roles={['super_admin','department_admin','examiner']}><QuestionEditorPage /></RequireRole>`
- `path="questions/:id/edit"` → `<RequireRole roles={['super_admin','department_admin','examiner']}><QuestionEditorPage /></RequireRole>`

### 5. i18n Keys

Added `questionEditor.*` namespace to all three locale files:
- `frontend/src/locales/en.json`
- `frontend/src/locales/kk.json`
- `frontend/src/locales/ru.json`

Keys cover: title, locale coverage, type names, difficulty, status, actions (saveDraft, save, submitForReview, approve, archive), autosave indicators, sections, fields, category/tags placeholders, explanation, answer option controls, preview, and error messages.

---

## Acceptance Criteria Coverage

| AC | Status | Notes |
|----|--------|-------|
| AC-1 | ✓ | `grid-cols-1 lg:grid-cols-2` — single column on mobile, two-column on ≥lg |
| AC-2 | ✓ | LocaleTabBar with ✓/○ based on stem content; EN is first tab |
| AC-3 | ✓ | AnswerOptionsSection conditionally renders per type; shorttext hides panel; truefalse locks to 2; likert shows weight+polarity |
| AC-4 | ✓ | DndKit SortableContext + drag handles for single/multiple; sort_order updated on drag end |
| AC-5 | ✓ | Difficulty select; CategoryPicker with full ancestor path display |
| AC-6 | ✓ | TagCombobox with autocomplete; creates new tag via POST /api/v1/tags; chips with dismiss |
| AC-7 | ✓ | Status buttons shown per current status; useUpdateQuestionStatus with optimistic update + rollback |
| AC-8 | ✓ | Auto-save every 30s in edit mode only; "Saving…"/"Saved" indicator; create mode shows "Save Draft" |
| AC-9 | ✓ | Explanation field collapsible (Add explanation / Hide); 1000-char counter |
| AC-10 | ✓ | All strings from i18n; zero hardcoded English in component |
| AC-11 | ✓ | Non-2xx responses show notification banner with `questionEditor.error.saveFailed` |

---

## Build Verification

`npm run build` — ✓ clean (0 TypeScript errors, 0 warnings)

---

## Files Changed

- `frontend/src/api/questions.ts` — new types + 5 new hooks
- `frontend/src/pages/admin/questions/QuestionEditorPage.tsx` — new file (591 lines)
- `frontend/src/App.tsx` — import + 2 new routes
- `frontend/src/locales/en.json` — questionEditor.* namespace added
- `frontend/src/locales/kk.json` — questionEditor.* namespace added
- `frontend/src/locales/ru.json` — questionEditor.* namespace added
- `frontend/package.json` + `frontend/package-lock.json` — @dnd-kit/* installed
- `docs/requirements/FR-BB26.Frontend-question-editor.md` — status: validated → implemented

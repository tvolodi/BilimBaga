# FR-BB312 — Frontend: Exam Configuration UI

## Metadata
| Field | Value |
|-------|-------|
| ID | FR-BB312 |
| Phase | 3 — Exam Engine |
| Priority | 2 |
| Status | Draft |
| Depends On | FR-BB32, FR-BB33 |

## Description
Implements a four-step wizard for creating and configuring exams, accessible to examiners and admins. Steps cover basic settings, question rule building, assignment configuration, and a review-and-publish screen with live validation. All state is persisted to the API incrementally so progress is not lost on refresh.

## Acceptance Criteria
- [ ] AC-1: The wizard has exactly four numbered steps: (1) Basic Settings, (2) Question Rules, (3) Assignments, (4) Review & Publish. Navigation between steps is blocked if the current step has validation errors.
- [ ] AC-2: Step 1 includes inputs for all top-level exam fields: title, description, time limit (minutes), passing score (%), max attempts, availability window (from/until date-time pickers), shuffle questions toggle, shuffle options toggle, show-answers select, on-tab-switch select, and certificate enabled toggle.
- [ ] AC-3: Step 2 renders a list of rule rows; each row includes mode toggle (manual/random), category picker (async-loaded from question bank categories), difficulty select, tag multi-select, count input, and a drag handle for reordering. A "manual" mode row shows an "Edit questions" button that opens a question picker modal.
- [ ] AC-4: The question picker modal (for manual rules) allows search/filter by keyword, category, and difficulty; shows paginated question list; supports checkbox multi-select; confirms selection with a count badge.
- [ ] AC-5: Step 3 renders the assignment panel with an "Add assignee" dropdown (users / departments / all), a deadline date picker per assignee, and a remove button. Existing assignments from the API are shown and can be removed.
- [ ] AC-6: Step 4 shows a read-only summary of all configuration; a live validation warning banner appears (using the publish-validation response from `POST /publish`) listing any rules that cannot be satisfied.
- [ ] AC-7: The "Publish" button in step 4 opens a confirmation modal before calling the publish API; on success, the user is redirected to the exam detail page and a success toast is shown.
- [ ] AC-8: All mutations use React Query `useMutation`; loading spinners are shown during in-flight requests and error toasts appear on API errors.
- [ ] AC-9: Zero hardcoded user-visible strings; all labels, placeholders, tooltips, and error messages reference `src/locales/{locale}.json` keys.
- [ ] AC-10: The wizard is accessible from the exam list page via a "Create exam" button and from the exam detail page via an "Edit" button (edit mode pre-populates all fields from the existing exam).

## Technical Specification

### Frontend Components

```
pages/
  ExamWizard/
    index.tsx                  — wizard shell with step state, Next/Back navigation
    Step1BasicSettings.tsx     — controlled form for top-level exam fields
    Step2QuestionRules.tsx     — rule list with DnD reorder
    Step2QuestionPickerModal.tsx — modal for manual question selection
    Step3Assignments.tsx       — assignee list with add/remove
    Step4Review.tsx            — read-only summary + publish validation

api/
  exams.ts                     — React Query hooks: useExams, useExam, useCreateExam,
                                  useUpdateExam, usePublishExam, useArchiveExam,
                                  useAddSection, useUpdateSection, useDeleteSection,
                                  useAddRule, useUpdateRule, useDeleteRule,
                                  useSetRuleQuestions,
                                  useAddAssignment, useDeleteAssignment,
                                  useExamAssignments
```

#### Wizard State Machine

```
Step 1 (Basic Settings)
  → [Next: auto-save exam via PUT or POST if new]
Step 2 (Question Rules)
  → [Next: rules already persisted per-mutation]
Step 3 (Assignments)
  → [Next: assignments already persisted per-mutation]
Step 4 (Review & Publish)
  → [Publish → confirm modal → POST /publish → redirect]
```

#### Key Component Props / Interfaces

```typescript
interface ExamWizardProps {
  examId?: string;   // undefined = create mode, set = edit mode
}

interface RuleRowProps {
  rule: ExamQuestionRule;
  onUpdate: (updated: Partial<ExamQuestionRule>) => void;
  onDelete: () => void;
  onEditQuestions: () => void;
  dragHandleProps: DragHandleProps;
}

interface AssigneeRowProps {
  assignment: ExamAssignment;
  onRemove: () => void;
}
```

#### i18n Key Examples

```json
{
  "exam.wizard.step1.title": "Basic Settings",
  "exam.wizard.step2.title": "Question Rules",
  "exam.wizard.step3.title": "Assignments",
  "exam.wizard.step4.title": "Review & Publish",
  "exam.wizard.step2.addRule": "Add rule",
  "exam.wizard.step2.mode.manual": "Manual",
  "exam.wizard.step2.mode.random": "Random",
  "exam.wizard.step4.publishConfirm": "Publishing this exam will make it available to assigned users. This cannot be undone. Are you sure?",
  "exam.wizard.validationWarning": "Some question rules cannot be satisfied. Fix them before publishing.",
  "exam.field.title": "Exam title",
  "exam.field.timeLimitMinutes": "Time limit (minutes)",
  "exam.field.passingScorePct": "Passing score (%)",
  "exam.field.maxAttempts": "Max attempts",
  "exam.field.availableFrom": "Available from",
  "exam.field.availableUntil": "Available until",
  "exam.field.shuffleQuestions": "Shuffle questions",
  "exam.field.shuffleOptions": "Shuffle answer options",
  "exam.field.showAnswers": "Show answers",
  "exam.field.onTabSwitch": "On tab switch",
  "exam.field.certificateEnabled": "Issue certificate on pass"
}
```

## Notes
- Drag-and-drop for rule reordering should use `@dnd-kit/sortable` (consistent with shadcn/ui guidance); `react-beautiful-dnd` is not recommended for React 18.
- The `ExamWizard` component should read from the existing exam on mount (edit mode) using `useExam(examId)` and pre-populate all step forms.
- Step validation should run client-side (required fields, numeric ranges) before making API calls; server-side errors are shown as field-level error messages below inputs using the shadcn/ui `FormMessage` component.
- The question picker modal in Step 2 should use `useInfiniteQuery` for paginated question loading with a keyword search debounce of 300 ms.
- Availability window inputs should use the shadcn/ui `DateTimePicker` with timezone displayed as user's local time but submitted as UTC ISO 8601.

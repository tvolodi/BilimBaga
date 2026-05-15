# FR-BB312 — Frontend: Exam Configuration UI

## Metadata
| Field | Value |
|-------|-------|
| ID | FR-BB312 |
| Phase | 3 — Exam Engine |
| Priority | 2 |
| Status | implemented |
| Depends On | FR-BB31, FR-BB32, FR-BB33 |

## Description
Implements a four-step wizard for creating and configuring exams, accessible to examiners and admins. Steps cover basic settings, question rule building, assignment configuration, and a review-and-publish screen. All state is persisted to the API incrementally so progress is not lost on refresh.

## Scope

| Layer | Items |
|-------|-------|
| Database | No new tables; reads from `exams`, `exam_question_rules`, `exam_assignments` |
| API endpoints | GET /exams, GET /exams/:id, POST /exams, PUT /exams/:id, POST /exams/:id/publish, POST /exams/:id/assign, DELETE /exams/:id/assign/:assignmentId, GET /exams/:id/assignments |
| Frontend pages/components | `pages/ExamWizard/` (index, Step1–Step4 components, QuestionPickerModal), `api/exams.ts` hooks |
| i18n keys | `exam.wizard.*`, `exam.field.*` |

## Acceptance Criteria
- [ ] AC-1: The wizard has exactly four numbered steps: (1) Basic Settings, (2) Question Rules, (3) Assignments, (4) Review & Publish. Navigation between steps is blocked if the current step has validation errors.
- [ ] AC-2: Step 1 includes inputs for all top-level exam fields: title, description, time limit (minutes), passing score (%), max attempts, availability window (from/until date-time pickers), shuffle questions toggle, shuffle options toggle, show-answers select, on-tab-switch select, and certificate enabled toggle.
- [ ] AC-3: Step 2 renders a list of rule rows; each row includes mode toggle (manual/random), category picker (async-loaded from question bank categories), difficulty select, tag multi-select, count input, and a drag handle for reordering. A "manual" mode row shows an "Edit questions" button that opens a question picker modal.
- [ ] AC-4: The question picker modal (for manual rules) allows search/filter by keyword, category, and difficulty; shows paginated question list; supports checkbox multi-select; confirms selection with a count badge.
- [ ] AC-5: Step 3 renders the assignment panel. For admin+ roles: an "Add assignee" dropdown (users / departments / all), a deadline date picker per assignee, and a "Remove" button are shown and functional. For examiner roles: Step 3 is read-only, displaying current assignments only — the "Add assignee" and "Remove" controls are hidden.
- [ ] AC-6: Step 4 shows a read-only summary of all configuration. When the user clicks "Publish" and the API returns HTTP 422, a validation warning banner appears listing the unsatisfied rules from the error payload. The banner is not shown unless the publish attempt has been made and returned a 422 response.
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
                                  useUpdateExam, usePublishExam,
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

#### Routes

| Route | Component | RequireRole |
|-------|-----------|-------------|
| `/admin/exams/new` | `ExamWizard` (create mode) | examiner+ |
| `/admin/exams/:id/edit` | `ExamWizard` (edit mode) | examiner+ |

Both routes must be registered in `App.tsx`, which currently lacks them.

#### Query Keys

| Key | Usage |
|-----|-------|
| `['exams']` | Exam list |
| `['exams', id]` | Single exam detail |
| `['exams', id, 'assignments']` | Exam assignment list |

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
- Availability window inputs should use shadcn/ui `Popover` + `Calendar` for date selection combined with a native `<input type="time">` for time input, assembled into a local `DateTimePicker` component at `components/ui/date-time-picker.tsx`. Timezone is displayed as user's local time but submitted as UTC ISO 8601.

## Out of Scope

- Exam taking / session screens (covered by separate FR)
- Manual grading of open-ended responses
- Certificate issuance and verification
- Analytics and performance dashboards

## Test Strategy

- **Unit tests**: wizard step validation logic (required fields, numeric range checks) tested in isolation.
- **React Testing Library**: each step component (`Step1BasicSettings`, `Step2QuestionRules`, `Step3Assignments`, `Step4Review`) has component tests covering render, user interaction, and error states.
- **MSW mocks**: all API calls (`useCreateExam`, `useUpdateExam`, `usePublishExam`, `useAddAssignment`, etc.) mocked via MSW handlers in the test environment.
- **E2E (Playwright)**: a happy-path test covering create-exam: fill Step 1 → add one random rule in Step 2 → skip Step 3 → publish from Step 4 → assert redirect to exam detail page.

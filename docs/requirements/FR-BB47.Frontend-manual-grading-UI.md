# FR-BB47 — Frontend: Manual Grading UI

## Metadata
| Field | Value |
|-------|-------|
| ID | FR-BB47 |
| Phase | 4 — Results & Certificates |
| Priority | 2 |
| Status | Ready |
| Depends On | FR-BB42 |

## Description
Provides the examiner-facing UI for reviewing and scoring short-text answers. A queue table lists all sessions awaiting manual grading. Clicking a session opens a detail page where the examiner navigates through answers one at a time, assigns a score on a slider, optionally adds feedback text, and submits all grades at once. When the final grade is submitted, the session is automatically finalized and the examiner is returned to the queue.

## Scope

| Layer | Items |
|-------|-------|
| Frontend pages | `GradingQueuePage` (`/admin/grading`), `GradingDetailPage` (`/admin/grading/:sessionId`) |
| Frontend components | `GradingQueueTable`, `QuestionGrader`, `GradingNavigation`, `SubmitAllGradesButton` |
| API hooks | `useGradingQueue`, `useGradingSession`, `useSubmitGrade` (in `src/api/grading.ts`) |
| i18n keys | `grading.*` namespace (~22 keys added to `src/locales/{kk,ru,en}.json`) |
| Backend dependencies | FR-BB42 endpoints consumed: `GET /admin/grading`, `GET /admin/grading/:sessionId`, `POST /admin/grading/:sessionId/answers/:questionId` |

## Acceptance Criteria
- [ ] AC-1: The grading queue table is accessible only to users with `role IN (examiner, department_admin, super_admin)`; employees are redirected with a 403 message.
- [ ] AC-2: Queue table columns: session ID (first 8 chars), employee name, exam name, submission date, count of pending questions; rows are sorted by submission date ascending (oldest first).
- [ ] AC-3: Clicking a queue row navigates to the grading detail page for that session.
- [ ] AC-4: The grading detail page shows one question at a time with a "Question X of Y" indicator; Previous/Next navigation buttons allow moving between questions.
- [ ] AC-5: Already-graded questions show their current score and feedback as pre-filled values so the examiner can revise them.
- [ ] AC-6: The score input is a combined slider (0–100) and numeric text field that stay in sync; entering a value outside 0–100 in the text field disables the Submit button and shows an inline error.
- [ ] AC-7: The "Submit All Grades" button is enabled only when every question has a score value assigned (no `null` scores).
- [ ] AC-8: Clicking "Submit All Grades" fires `POST /admin/grading/:sessionId/answers/:questionId` sequentially for each question via React Query mutations; a loading spinner is shown during submission.
- [ ] AC-9: On successful submission of all grades, a success toast is shown and the examiner is navigated back to the queue; the queue list is invalidated and refetched.
- [ ] AC-10: Zero hardcoded user-visible strings; all text uses `useTranslation` i18n keys.
- [ ] AC-11: If any grade submission mutation fails, an error toast is shown identifying the failed question; remaining mutations are not fired; the Submit button returns to enabled state for retry.

## Technical Specification

### Frontend Components

#### Page: `GradingQueuePage` (`src/pages/admin/GradingQueuePage.tsx`)
- Route: `/admin/grading`
- Guarded by `RequireRole(['examiner','department_admin','super_admin'])` wrapper
- Uses `useGradingQueue(page)` hook
- Renders `<GradingQueueTable>` + `<Pagination>`

#### Page: `GradingDetailPage` (`src/pages/admin/GradingDetailPage.tsx`)
- Route: `/admin/grading/:sessionId`
- Uses `useGradingSession(sessionId)` + local state for score/feedback per question

#### Hook: `useGradingQueue`
```ts
// src/api/grading.ts
export function useGradingQueue(page: number, examId?: string) {
  return useQuery({
    queryKey: ['grading-queue', page, examId],
    queryFn: () =>
      apiGet<GradingQueueResponse>(`/admin/grading?page=${page}&per_page=20${examId ? `&exam_id=${examId}` : ''}`),
  });
}
```

#### Hook: `useGradingSession`
```ts
export function useGradingSession(sessionId: string) {
  return useQuery({
    queryKey: ['grading-session', sessionId],
    queryFn: () => apiGet<GradingSessionDetail>(`/admin/grading/${sessionId}`),
  });
}
```

#### Hook: `useSubmitGrade`
```ts
export function useSubmitGrade(sessionId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ questionId, scorePct, feedback }: GradeSubmission) =>
      apiPost<GradeAnswerResponse>(`/admin/grading/${sessionId}/answers/${questionId}`, { score_pct: scorePct, feedback }),
    onSuccess: (data: GradeAnswerResponse) => {
      queryClient.invalidateQueries({ queryKey: ['grading-session', sessionId] });
      if (data.all_graded) {
        queryClient.invalidateQueries({ queryKey: ['grading-queue'] });
      }
    },
  });
}
```

#### Component: `GradingQueueTable` (`src/components/grading/GradingQueueTable.tsx`)
```tsx
// Props: { sessions: GradingQueueItem[]; }
// Renders shadcn Table; each row is clickable → navigates to /admin/grading/:sessionId
// Pending count shown as <Badge variant="outline">
```

#### Component: `QuestionGrader` (`src/components/grading/QuestionGrader.tsx`)
```tsx
// Props: { question: GradingQuestion; score: number|null; feedback: string; onChange: fn }
// Layout:
//   - Question stem (read-only, large text)
//   - Employee answer (read-only Textarea, bg-muted)
//   - Score: <Slider min=0 max=100 step=1> + <Input type="number" min=0 max=100>
//   - Feedback: <Textarea placeholder={t('grading.feedback_placeholder')} optional />
// Slider and input stay in sync via shared controlled state
```

#### Component: `GradingNavigation`
```tsx
// Props: { current: number; total: number; onPrev: fn; onNext: fn }
// Renders "< Question X of Y >" with disabled states at boundaries
```

#### Component: `SubmitAllGradesButton`
```tsx
// Props: { grades: GradeMap; questionCount: number; onSubmit: fn; isLoading: boolean }
// Enabled only when Object.keys(grades).length === questionCount && all scores !== null
// Shows shadcn Button with Loader2 spinner when isLoading
```

### Local State Shape (GradingDetailPage)

```ts
interface GradeMap {
  [questionId: string]: { score_pct: number | null; feedback: string };
}
// Initialised from API response: graded questions pre-fill their current values
```

### i18n Keys Required

```json
{
  "grading.queue_title": "Manual Grading Queue",
  "grading.col_session": "Session",
  "grading.col_employee": "Employee",
  "grading.col_exam": "Exam",
  "grading.col_submitted": "Submitted",
  "grading.col_pending": "Pending Questions",
  "grading.queue_empty": "No sessions are awaiting manual grading.",
  "grading.detail_title": "Grade Session",
  "grading.question_indicator": "Question {{current}} of {{total}}",
  "grading.answer_label": "Employee's Answer",
  "grading.score_label": "Score (0–100%)",
  "grading.score_error": "Score must be between 0 and 100",
  "grading.feedback_label": "Feedback (optional)",
  "grading.feedback_placeholder": "Add comments for the employee...",
  "grading.prev_question": "Previous",
  "grading.next_question": "Next",
  "grading.submit_all": "Submit All Grades",
  "grading.submitting": "Submitting...",
  "grading.success_toast": "All grades submitted. Session has been finalized.",
  "grading.error_toast": "Failed to submit grade for question {{question}}. Please retry.",
  "grading.already_graded": "Already graded"
}
```

### TypeScript Types

```ts
export interface GradingQueueItem {
  session_id: string;
  employee_name: string;
  exam_id: string;
  exam_title: string;
  submitted_at: string;
  pending_question_count: number;
}

export interface GradingQuestion {
  question_id: string;
  stem: string;
  text_answer: string;
  grading_status: 'pending_manual' | 'graded';
  current_score_pct: number | null;
  manual_feedback: string | null;
}

export interface GradingSessionDetail {
  session_id: string;
  employee_name: string;
  exam_title: string;
  submitted_at: string | null;
  questions: GradingQuestion[];
}

export interface GradeSubmission {
  questionId: string;
  scorePct: number;
  feedback: string;
}

export interface GradingQueueResponse {
  items: GradingQueueItem[];
  meta: { page: number; per_page: number; total: number };
}

export interface GradeAnswerResponse {
  question_id: string;
  grading_status: string;
  score_pct: number;
  session_status: string;
  all_graded: boolean;
  final_score_pct?: number | null;
  passed?: boolean | null;
}
```

## Notes
- Grades are submitted sequentially (not in parallel) to avoid race conditions in the server-side "all graded" check.
- The optimistic update approach is not used here because the final-grade detection logic lives on the server; the client waits for the last mutation to succeed and reads `all_graded` from the response to decide whether to navigate away.
- The queue page should support an `exam_id` filter dropdown (populated from active exams) to let examiners focus on one exam at a time.

## Out of Scope
- Bulk import of grades via CSV or spreadsheet upload.
- Grading via mobile-optimised or native mobile interface.
- AI-assisted scoring suggestions or automated short-text evaluation.
- Editing or overriding auto-graded (non-manual) question scores.
- Real-time collaboration (multiple examiners grading the same session simultaneously).

## Test Strategy

### Unit Tests
- **GradeMap initialisation**: given an API response containing a mix of `pending_manual` and `graded` questions, verify that `GradeMap` is initialised with `null` scores for pending questions and with existing `current_score_pct` / `manual_feedback` values for already-graded questions.
- **SubmitAllGradesButton enablement**: given a `GradeMap` with one `null` score, verify the button is disabled; given all scores filled, verify enabled.
- **Score input validation**: entering a value outside 0–100 in the numeric field renders the inline error and disables Submit.

### Role-Guard Tests
- Rendering `GradingQueuePage` with an `employee` role mock redirects to the 403 page (does not render the queue table).
- Rendering with `examiner`, `department_admin`, or `super_admin` roles renders the queue table without redirect.

### Integration Tests
- **Sequential mutation flow**: mock all three endpoints; trigger "Submit All Grades" for a session with two questions; assert that the second `POST` is only fired after the first resolves; assert that `['grading-queue']` is invalidated after the final response returns `all_graded: true`.
- **Partial failure**: mock the first question's `POST` to succeed and the second to fail; assert that the second mutation is not fired; assert that an error toast appears; assert that the Submit button re-enables.

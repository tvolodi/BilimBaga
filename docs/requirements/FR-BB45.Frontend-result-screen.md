# FR-BB45 — Frontend: Result Screen

## Metadata
| Field | Value |
|-------|-------|
| ID | FR-BB45 |
| Phase | 4 — Results & Certificates |
| Priority | 2 |
| Status | Draft |
| Depends On | FR-BB41, FR-BB43 |

## Description
Displays the exam result to an employee immediately after submission. The screen shows a score dial with pass/fail status, time taken, per-section scores (if applicable), a question-by-question review table (if the exam permits it), and action buttons for certificate download, exam retake, and navigation back to the portal. All data is fetched via React Query from the result retrieval API.

## Scope

| Layer | Items |
|-------|-------|
| Pages | `ResultPage.tsx` (`src/pages/ResultPage.tsx`), `App.tsx` (route registration for `/portal/sessions/:sessionId/result`) |
| Components | `ScoreDial`, `PassFailBanner`, `TimeTakenBadge`, `SectionScores`, `QuestionBreakdownTable`, `ResultActions` |
| API hooks | `useSessionResult(sessionId)`, `usePortalExam(examId)` |
| TypeScript types | `SessionResult`, `SectionScore`, `QuestionBreakdownItem`, `PortalExamDetail` |
| i18n keys | `result.*` namespace (added to `src/locales/{kk,ru,en}.json`) |

## Acceptance Criteria
- [ ] AC-1: The score is displayed as a circular progress dial showing the percentage; the dial fill colour uses the tenant's `primary_color` from branding context.
- [ ] AC-2: A pass/fail banner is clearly visible: green with text "Passed" (i18n key `result.passed`) or red with text "Failed" (i18n key `result.failed`).
- [ ] AC-3: Time taken is displayed in the format `Xm Ys` (e.g. "23m 43s"); if `time_taken_seconds` is null (grading pending), the field is hidden.
- [ ] AC-4: When `per_section_scores` is non-empty, a row of score cards renders one card per section showing section title and percentage.
- [ ] AC-5: When `per_question_breakdown` is present, a table renders each row with: question stem (truncated to 120 chars), employee's answer, correct answer highlighted in green, points earned / max points, and explanation text (collapsible if longer than 200 chars).
- [ ] AC-6: The "Download Certificate" button is shown **only** when `passed = true` AND `PortalExam.certificate_enabled` (from `usePortalExam(examId)`) is `true`; clicking it calls `GET /api/v1/portal/sessions/:id/certificate` and triggers a browser file download.
- [ ] AC-7: The "Retake Exam" button is shown only when `attempt_number < PortalExam.max_attempts` (from `usePortalExam(examId)`) AND `PortalExam.user_status` is not `'expired'`; it navigates to the exam start page.
- [ ] AC-8: When `score_pct` is `null`, the score dial and pass/fail banner are replaced with a "Results Pending Manual Review" notice (i18n key `result.pending_grading`); certificate and retake buttons are hidden.
- [ ] AC-9: The page uses `useQuery` from TanStack Query with `queryKey: ['session-result', sessionId]`; loading and error states render appropriate skeleton/alert components.
- [ ] AC-10: Zero hardcoded user-visible strings; all text uses `useTranslation` i18n keys.

## Out of Scope

- Admin result view (separate admin UI — not part of this requirement)
- PDF generation or server-side certificate rendering
- Manual grading UI and grading queue management
- Push notifications for result availability

## Technical Specification

### Frontend Components

#### Page: `ResultPage` (`src/pages/ResultPage.tsx`)
- Route: `/portal/sessions/:sessionId/result`
- Fetches result with `useSessionResult(sessionId)` hook
- Fetches exam metadata with `usePortalExam(examId)` hook (provides `certificate_enabled` and `max_attempts`)
- `examId` is derived from `sessionResult.data?.exam_id`. The `usePortalExam` hook must use `enabled: !!examId` to prevent firing with an undefined/empty string while the first query is loading.
- Renders `<ScoreDial>`, `<PassFailBanner>`, `<TimeTakenBadge>`, `<SectionScores>`, `<QuestionBreakdownTable>`, `<ResultActions>`

#### Hook: `useSessionResult`
```ts
// src/api/sessions.ts
// Authentication: requires a valid employee JWT cookie (the `/api/v1/portal/` route prefix enforces this server-side).
export function useSessionResult(sessionId: string) {
  return useQuery<SessionResult, Error>({
    queryKey: ['session-result', sessionId],
    queryFn: () => apiFetch<SessionResult>(`/api/v1/portal/sessions/${sessionId}/result`),
    staleTime: Infinity, // results don't change once submitted
  });
}
```

#### Hook: `usePortalExam`
```ts
// src/api/portal.ts
// Authentication: requires a valid employee JWT cookie (the `/api/v1/portal/` route prefix enforces this server-side).
export function usePortalExam(examId: string | undefined) {
  return useQuery<PortalExamDetail, Error>({
    queryKey: ['portal', 'exams', examId],
    queryFn: () => apiFetch<PortalExamDetail>(`/api/v1/portal/exams/${examId}`),
    staleTime: Infinity,
    enabled: !!examId,
  });
}
```

#### Component: `ScoreDial` (`src/components/results/ScoreDial.tsx`)
```tsx
// Props: { scorePct: number; primaryColor: string }
// Uses SVG circle with stroke-dasharray/stroke-dashoffset for animation
// Shows percentage text centered inside the dial
```

#### Component: `PassFailBanner`
```tsx
// Props: { passed: boolean }
// Renders a full-width coloured banner with icon + localised text
// passed=true → bg-green-50 border-green-500 text-green-800
// passed=false → bg-red-50 border-red-500 text-red-800
```

#### Component: `SectionScores`
```tsx
// Props: { sections: Array<{ section_id, title, score_pct }> }
// Renders a horizontal flex row of shadcn Card components
// Hidden if sections array is empty
```

#### Component: `QuestionBreakdownTable`
```tsx
// Props: { breakdown: QuestionBreakdownItem[] | undefined }
// Renders shadcn Table; hidden if breakdown is undefined
// Correct answer cell: bg-green-50, bold
// Incorrect employee answer cell: bg-red-50
```

#### Component: `ResultActions`
```tsx
// Props: { sessionId, passed, certificateEnabled, canRetake, examId }
// Certificate button calls: GET /api/v1/portal/sessions/:id/certificate → triggers download
// Uses useMutation with onSuccess handler that creates <a> element and programmatically clicks it
// The mutationFn must use raw fetch() (not apiFetch) since the response is a binary PDF blob;
// parse via res.blob() and pass to window.URL.createObjectURL.
```

### i18n Keys Required

Add to `src/locales/{kk,ru,en}.json`:
```json
{
  "result": {
    "title": "Exam Result",
    "passed": "Passed",
    "failed": "Failed",
    "pending_grading": "Your answers are under review. Results will be available after manual grading.",
    "score_label": "Your Score",
    "time_taken": "Time Taken",
    "attempt_number": "Attempt",
    "section_scores": "Section Scores",
    "question_review": "Question Review",
    "your_answer": "Your Answer",
    "correct_answer": "Correct Answer",
    "points": "Points",
    "explanation": "Explanation",
    "download_certificate": "Download Certificate",
    "retake_exam": "Retake Exam",
    "back_to_portal": "Back to Portal"
  }
}
```

### TypeScript Types

```ts
// src/api/types.ts

export interface SessionResult {
  session_id: string;
  exam_id: string;
  exam_title: string;
  score_pct: number | null;
  passed: boolean;
  time_taken_seconds: number | null;
  attempt_number: number;
  submitted_at: string;
  show_answers_mode: 'never' | 'after_completion' | 'after_all_attempts';
  per_section_scores: SectionScore[];
  per_question_breakdown: QuestionBreakdownItem[] | undefined;
}

export interface SectionScore {
  section_id: string;
  title: string;
  score_pct: number;
}

export interface QuestionBreakdownItem {
  question_id: string;
  stem: string;
  employee_answer: string[];
  correct_answer: string[];
  points_earned: number;
  max_points: number;
  explanation: string | null;
}
```

```ts
// src/api/portal.ts (alongside PortalExam)

export interface PortalExamDetail extends PortalExam {
  certificate_enabled: boolean;
  // max_attempts is inherited from PortalExam
}
```

## Test Strategy

- **Vitest unit — `ScoreDial`**: Given `scorePct=75`, assert that the SVG `stroke-dashoffset` value equals `(1 - 0.75) * circumference`.
- **Vitest unit — `PassFailBanner`**: Assert the rendered element has class `bg-green-50` when `passed=true` and `bg-red-50` when `passed=false`.
- **Vitest unit — `QuestionBreakdownTable`**: Given a 3-item breakdown array, assert 3 table rows are rendered and the correct-answer cell has class `bg-green-50`.
- **Vitest integration — `ResultPage` loading state**: Mock `useSessionResult` to return `{ isLoading: true }`; assert skeleton component is present and no score content is rendered.
- **Vitest integration — `ResultPage` error state**: Mock `useSessionResult` to return `{ isError: true }`; assert an alert/error component is rendered.
- **Vitest integration — `ResultPage` pending grading**: Mock `useSessionResult` to return data with `score_pct: null`; assert pending-grading notice is visible and score dial is absent.
- **MSW integration — full result flow**: Use MSW to intercept `GET /api/v1/portal/sessions/:id/result` and `GET /api/v1/portal/exams/:id`; assert all result sections render correctly for a passing submission with breakdown enabled.

## Technical Notes
- The circular `ScoreDial` must be implemented with SVG, not an external chart library, to keep the bundle small and allow `primary_color` theming.
- Certificate download must not navigate away from the page; use `window.URL.createObjectURL` on the PDF blob response.
- The page is reachable via the exam-taking flow (automatic redirect after submission) and via `MyResults` history table.

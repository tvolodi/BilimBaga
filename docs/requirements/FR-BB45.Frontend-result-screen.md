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

## Acceptance Criteria
- [ ] AC-1: The score is displayed as a circular progress dial showing the percentage; the dial fill colour uses the tenant's `primary_color` from branding context.
- [ ] AC-2: A pass/fail banner is clearly visible: green with text "Passed" (i18n key `result.passed`) or red with text "Failed" (i18n key `result.failed`).
- [ ] AC-3: Time taken is displayed in the format `Xm Ys` (e.g. "23m 43s"); if `time_taken_seconds` is null (grading pending), the field is hidden.
- [ ] AC-4: When `per_section_scores` is non-empty, a row of score cards renders one card per section showing section title and percentage.
- [ ] AC-5: When `per_question_breakdown` is present, a table renders each row with: question stem (truncated to 120 chars), employee's answer, correct answer highlighted in green, points earned / max points, and explanation text (collapsible if longer than 200 chars).
- [ ] AC-6: The "Download Certificate" button is shown **only** when `passed = true` AND `certificate_enabled = true`; clicking it calls `GET /portal/sessions/:id/certificate` and triggers a browser file download.
- [ ] AC-7: The "Retake Exam" button is shown only when `attempt_number < exam.max_attempts` AND the exam is still active; it navigates to the exam start page.
- [ ] AC-8: When the session status is `grading_pending`, the score dial and pass/fail banner are replaced with a "Results Pending Manual Review" notice (i18n key `result.pending_grading`); certificate and retake buttons are hidden.
- [ ] AC-9: The page uses `useQuery` from TanStack Query with `queryKey: ['session-result', sessionId]`; loading and error states render appropriate skeleton/alert components.
- [ ] AC-10: Zero hardcoded user-visible strings; all text uses `useTranslation` i18n keys.

## Technical Specification

### Frontend Components

#### Page: `ResultPage` (`src/pages/ResultPage.tsx`)
- Route: `/portal/sessions/:sessionId/result`
- Fetches result with `useSessionResult(sessionId)` hook
- Renders `<ScoreDial>`, `<PassFailBanner>`, `<TimeTakenBadge>`, `<SectionScores>`, `<QuestionBreakdownTable>`, `<ResultActions>`

#### Hook: `useSessionResult`
```ts
// src/api/sessions.ts
export function useSessionResult(sessionId: string) {
  return useQuery({
    queryKey: ['session-result', sessionId],
    queryFn: () => apiGet<SessionResult>(`/portal/sessions/${sessionId}/result`),
    staleTime: Infinity, // results don't change once submitted
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
// Props: { breakdown: QuestionBreakdownItem[] | null }
// Renders shadcn Table; hidden if breakdown is null
// Correct answer cell: bg-green-50, bold
// Incorrect employee answer cell: bg-red-50
```

#### Component: `ResultActions`
```tsx
// Props: { sessionId, passed, certificateEnabled, canRetake, examId }
// Certificate button calls: GET /portal/sessions/:id/certificate → triggers download
// Uses useMutation with onSuccess handler that creates <a> element and programmatically clicks it
```

### i18n Keys Required

Add to `src/locales/{kk,ru,en}.json`:
```json
{
  "result.title": "Exam Result",
  "result.passed": "Passed",
  "result.failed": "Failed",
  "result.pending_grading": "Your answers are under review. Results will be available after manual grading.",
  "result.score_label": "Your Score",
  "result.time_taken": "Time Taken",
  "result.attempt_number": "Attempt",
  "result.section_scores": "Section Scores",
  "result.question_review": "Question Review",
  "result.your_answer": "Your Answer",
  "result.correct_answer": "Correct Answer",
  "result.points": "Points",
  "result.explanation": "Explanation",
  "result.download_certificate": "Download Certificate",
  "result.retake_exam": "Retake Exam",
  "result.back_to_portal": "Back to Portal"
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
  passed: boolean | null;
  time_taken_seconds: number | null;
  attempt_number: number;
  submitted_at: string;
  show_answers_mode: 'never' | 'after_submission' | 'always';
  per_section_scores: SectionScore[];
  per_question_breakdown: QuestionBreakdownItem[] | null;
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
  correct_answer: string[] | null;
  points_earned: number;
  max_points: number;
  explanation: string | null;
}
```

## Notes
- The circular `ScoreDial` must be implemented with SVG, not an external chart library, to keep the bundle small and allow `primary_color` theming.
- Certificate download must not navigate away from the page; use `window.URL.createObjectURL` on the PDF blob response.
- The page is reachable via the exam-taking flow (automatic redirect after submission) and via `MyResults` history table.

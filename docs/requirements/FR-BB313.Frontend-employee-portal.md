# FR-BB313 — Frontend: Employee Portal

## Metadata
| Field | Value |
|-------|-------|
| ID | FR-BB313 |
| Phase | 3 — Exam Engine |
| Priority | 2 |
| Status | implemented |
| Depends On | FR-BB34, FR-BB35 |

## Scope

| Layer | Items |
|-------|-------|
| API endpoints consumed | `GET /api/v1/portal/exams` (list assigned exams), `POST /api/v1/portal/exams/:id/sessions` (create session) |
| Frontend pages/components | `pages/EmployeePortal/index.tsx`, `ExamCard.tsx`, `ExamCardSkeleton.tsx`, `StartExamModal.tsx`, `EmptyPortal.tsx` |
| React Query hooks | `usePortalExams`, `useCreateSession` (in `api/portal.ts`) |
| Custom hooks | `hooks/useCountdown.ts` |
| i18n key namespaces | `portal.*` |

## Description
Implements the employee-facing exam portal page — the primary entry point for non-admin users. Displays all assigned exams as a card grid with computed status indicators, countdown timers, attempt counters, and contextual call-to-action buttons. Supports an empty state and auto-refreshes data every 30 seconds to keep deadlines current.

## Acceptance Criteria
- [ ] AC-1: The portal page loads all assigned exams via `GET /api/v1/portal/exams` using React Query with a `refetchInterval` of 30 000 ms (30 seconds).
- [ ] AC-2: Each exam card displays: exam title, description (truncated to 2 lines), a status pill, time limit, passing score %, attempts used vs max, and deadline countdown (or "No deadline" if absent).
- [ ] AC-3: Status pill colours and labels: `not_started` → grey "Not started"; `in_progress` → blue "In progress"; `passed` → green "Passed"; `failed` → red "Failed"; `expired` → orange "Expired".
- [ ] AC-4: The CTA button per card: `not_started` → "Start exam"; `in_progress` → "Continue"; `passed` → "View result"; `failed` → "View result" (disabled if `show_answers === 'never'`); `expired` → no action button.
- [ ] AC-5: Clicking "Start exam" opens a confirmation modal that displays: time limit, whether questions/options are shuffled, max attempts warning, and cannot-leave warning message; the user must explicitly confirm before the session is created.
- [ ] AC-6: The confirmation modal's "Begin" button calls `POST /api/v1/portal/exams/:id/sessions`; on success, navigates to `/portal/sessions/:sessionId` (exam-taking route defined in FR-BB37).
- [ ] AC-7: Clicking "Continue" navigates to `/portal/sessions/:sessionId` using the `open_session_id` (no new session created).
- [ ] AC-8: An empty state component is shown when the `data` array is empty, with an appropriate icon and localised message.
- [ ] AC-9: Deadline countdown displays `HH:MM:SS` when less than 24 hours remain; `X days Y hours` for longer durations; counts down in real-time using a `setInterval` on the client (updated every second).
- [ ] AC-10: Zero hardcoded user-visible strings; all text references `src/locales/{locale}.json` keys.

## Technical Specification

### Frontend Components

```
pages/
  EmployeePortal/
    index.tsx               — portal page, React Query data fetch + grid layout
    ExamCard.tsx            — individual exam card with status, CTA, countdown
    ExamCardSkeleton.tsx    — loading skeleton for card grid
    StartExamModal.tsx      — confirmation modal before starting
    EmptyPortal.tsx         — empty state component

api/
  portal.ts                 — React Query hooks: usePortalExams, usePortalExam,
                               useCreateSession
```

#### ExamCard Component Interface

```typescript
interface ExamCardProps {
  exam: PortalExam;   // from GET /api/v1/portal/exams response item
  onStart: (examId: string) => void;
  onContinue: (sessionId: string) => void;
  onViewResult: (examId: string) => void;
}

interface PortalExam {
  id: string;
  title: string;
  description: string | null;
  time_limit_minutes: number;
  passing_score_pct: number;
  max_attempts: number;
  attempts_used: number;
  deadline: string | null;   // ISO 8601
  user_status: 'not_started' | 'in_progress' | 'passed' | 'failed' | 'expired';
  open_session_id: string | null;
  show_answers: 'never' | 'after_completion' | 'after_all_attempts';  // enables "View result" CTA for passed/failed
  shuffle_questions: boolean;   // displayed in StartExamModal pre-start confirmation
  shuffle_options: boolean;     // displayed in StartExamModal pre-start confirmation
}
```

#### StartExamModal Component Interface

```typescript
interface StartExamModalProps {
  exam: PortalExam;
  open: boolean;
  onClose: () => void;
  onConfirm: () => void;   // triggers session creation mutation
  isLoading: boolean;
}
```

> **Note**: `StartExamModal` reads `shuffle_questions` and `shuffle_options` directly from the `exam` prop (already loaded with the card list — no extra fetch needed).

#### CreateSession Response Interface

```typescript
// SessionQuestion is defined in the exam-taking screen requirement (FR-BB37)
interface CreateSessionResponse {
  session_id: string;
  exam_id: string;
  started_at: string;           // ISO 8601 UTC
  expires_at: string;           // ISO 8601 UTC; "0001-01-01T00:00:00Z" if no time limit
  remaining_seconds: number;    // 0 if no time limit
  questions: SessionQuestion[]; // forward reference — shape defined in FR-BB37
}
```

> **Technical Note**: An `expires_at` of `0001-01-01T00:00:00Z` or `remaining_seconds === 0` indicates the exam has no time limit. Frontend must handle this instead of checking for `null`.

```typescript
```

#### Countdown Timer Hook

```typescript
// hooks/useCountdown.ts
function useCountdown(deadline: string | null): string {
  const [display, setDisplay] = useState<string>(formatDeadline(deadline, Date.now()));

  useEffect(() => {
    if (!deadline) return;
    const interval = setInterval(() => {
      const remaining = new Date(deadline).getTime() - Date.now();
      if (remaining <= 0) {
        setDisplay(t('portal.deadline.expired'));
        clearInterval(interval);
        return;
      }
      setDisplay(formatDeadline(deadline, Date.now()));
    }, 1000);
    return () => clearInterval(interval);
  }, [deadline]);

  return display;
}
```

#### React Query Hook

```typescript
// api/portal.ts
export function usePortalExams() {
  return useQuery({
    queryKey: ['portal', 'exams'],
    queryFn: () => apiFetch<PortalExam[]>('/api/v1/portal/exams'),
    refetchInterval: 30_000,
  });
}

export function useCreateSession(examId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () => apiFetch<CreateSessionResponse>(`/api/v1/portal/exams/${examId}/sessions`, {
      method: 'POST',
    }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['portal', 'exams'] });
    },
  });
}
```

#### i18n Key Examples

```json
{
  "portal.title": "My Exams",
  "portal.empty.title": "No exams assigned",
  "portal.empty.description": "You have no exams assigned to you right now.",
  "portal.card.status.not_started": "Not started",
  "portal.card.status.in_progress": "In progress",
  "portal.card.status.passed": "Passed",
  "portal.card.status.failed": "Failed",
  "portal.card.status.expired": "Expired",
  "portal.card.cta.start": "Start exam",
  "portal.card.cta.continue": "Continue",
  "portal.card.cta.viewResult": "View result",
  "portal.card.timeLimit": "{{minutes}} min",
  "portal.card.passingScore": "Pass: {{pct}}%",
  "portal.card.attempts": "Attempt {{used}} of {{max}}",
  "portal.card.deadline": "Due {{countdown}}",
  "portal.card.noDeadline": "No deadline",
  "portal.modal.title": "Start exam",
  "portal.modal.timeLimit": "You will have {{minutes}} minutes.",
  "portal.modal.maxAttempts": "You have {{remaining}} attempt(s) remaining.",
  "portal.modal.warning": "Once started, do not close or switch tabs — it may be counted against you.",
  "portal.modal.confirm": "Begin exam",
  "portal.modal.cancel": "Cancel",
  "portal.deadline.expired": "Expired"
}
```

## Out of Scope

- Result screen UI (Phase 4 analytics — separate requirement).
- Session answer-saving logic and question rendering (FR-BB37).
- Admin exam management UI (covered in Phase 3 admin requirements).
- Backend `PortalExamItem` changes needed to expose `show_answers`, `shuffle_questions`, `shuffle_options` on the list endpoint (tracked separately as a backend concern; this requirement specifies the frontend contract).

## Test Strategy

- **Unit — `useCountdown` hook**: test that the returned string formats correctly for >24 h, <24 h, and exactly-zero (expired) cases using fake timers (`vi.useFakeTimers`).
- **Component — `ExamCard`**: render one card for each `user_status` value and assert the correct status pill label, pill colour class, and CTA button text/disabled state (using React Testing Library).
- **Component — `StartExamModal`**: assert that `shuffle_questions` / `shuffle_options` labels appear when the respective props are `true`, and that the "Begin exam" button is disabled while `isLoading` is true.
- **Integration — `EmployeePortal` page**: use MSW to mock `GET /api/v1/portal/exams` (returns 3 items) and assert the card grid renders 3 cards; mock an empty response and assert the `EmptyPortal` component is visible.
- **Integration — session creation**: mock `POST /api/v1/portal/exams/:id/sessions` and assert that clicking "Begin exam" in the modal triggers navigation to the exam-taking screen with the returned `session_id`.

## Notes
- The 30-second `refetchInterval` keeps deadline countdowns roughly server-accurate; the `useCountdown` hook handles the second-level ticking on the client between refetches.
- The card grid should use CSS Grid with responsive breakpoints: 1 column on mobile, 2 on tablet, 3 on desktop.
- Loading state should render a `ExamCardSkeleton` grid (same count as last known data, or 3 skeletons if no cached data) rather than a spinner to avoid layout shift.
- The portal page must be protected by the auth guard; unauthenticated users are redirected to `/login`.
- "View result" navigation for passed/failed status goes to `/portal/exams/:id/result` — the result screen is implemented in a separate requirement (Phase 4 analytics).

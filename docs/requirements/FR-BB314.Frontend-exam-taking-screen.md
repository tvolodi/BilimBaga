# FR-BB314 — Frontend: Exam Taking Screen

## Metadata
| Field | Value |
|-------|-------|
| ID | FR-BB314 |
| Phase | 3 — Exam Engine |
| Priority | 2 |
| Status | uat-verified |
| Depends On | FR-BB37, FR-BB38, FR-BB39 |

## Description
Implements the distraction-free exam-taking interface. The screen renders all session questions, handles answer selection with auto-save, tracks elapsed time with a server-authoritative countdown, detects tab-switch and focus-loss events to enforce the anti-cheat policy, provides a question navigator panel, supports flagging questions for review, and guides the user through a final submit confirmation flow.

## Acceptance Criteria
- [ ] AC-1: The exam taking screen renders without the global sidebar or navigation links; it uses a standalone full-focus layout component.
- [ ] AC-2: The top bar displays exam title, a server-authoritative countdown timer (using `remaining_seconds` from the last save response), and a progress indicator `X / Y answered`.
- [ ] AC-3: The timer turns amber (warning colour) when `remaining_seconds <= 20%` of total time and red (critical colour) when `remaining_seconds <= 5%` of total time.
- [ ] AC-4: Selecting an answer automatically triggers `PUT /answers/:questionId` via React Query mutation; the UI shows "Saving…" → "Saved ✓" → on error "Connection lost ✗" indicators.
- [ ] AC-5: Answer auto-save is debounced by 800 ms for text-area (short-text) inputs; radio/checkbox selections trigger save immediately on change.
- [ ] AC-6: The question navigator panel shows a numbered grid; each number is coloured: white = unanswered, blue = answered, yellow = flagged for review.
- [ ] AC-7: Tab-switch events are detected via `document.addEventListener('visibilitychange', ...)` and `window.addEventListener('blur', ...)`; on detection, `POST /sessions/:id/events` is called with `{ type: 'tab_switch' }` or `{ type: 'blur' }`; a warning modal is shown if the server returns `{ warn: true }`.
- [ ] AC-8: Clicking "Finish exam" navigates to a pre-submit review screen listing all unanswered questions and all flagged questions by number; a "Submit anyway" and "Go back" button are shown.
- [ ] AC-9: Confirming submission calls `POST /sessions/:id/submit`; on success, the screen transitions to a result summary showing pass/fail status and score (or a "Pending manual grading" message if applicable).
- [ ] AC-10: Zero hardcoded user-visible strings; all text references `src/locales/{locale}.json` keys.

## Technical Specification

### Frontend Components

```
pages/
  ExamTaking/
    index.tsx                 — route entry, loads session via GET /sessions/:id or
                                 creates session from portal; renders ExamLayout
    ExamLayout.tsx            — full-focus layout (no sidebar), top bar, main area
    ExamTopBar.tsx            — title, CountdownTimer, progress X/Y answered
    CountdownTimer.tsx        — server-sync countdown with colour thresholds
    QuestionNavigator.tsx     — collapsible panel, numbered grid, answered/flagged state
    QuestionDisplay.tsx       — renders question stem + answer input by type
    SingleChoiceInput.tsx     — radio group for single_choice / true_false
    MultipleChoiceInput.tsx   — checkbox group for multiple_choice
    LikertInput.tsx           — button row for likert (1–5 scale)
    ShortTextInput.tsx        — textarea with debounced save
    SaveIndicator.tsx         — "Saving…" / "Saved ✓" / "Connection lost ✗"
    FlagButton.tsx            — toggle flag for review per question
    FinishReviewScreen.tsx    — pre-submit review of unanswered/flagged questions
    SubmitConfirmModal.tsx    — final confirmation modal
    ResultScreen.tsx          — post-submit result (pass/fail/pending)
    TabSwitchWarningModal.tsx — anti-cheat warning modal

api/
  sessions.ts                 — hooks: useSession, useSaveAnswer, useSubmitSession,
                                 useReportEvent
```

#### Component Interfaces

```typescript
interface ExamLayoutProps {
  sessionId: string;
}

interface CountdownTimerProps {
  remainingSeconds: number;
  totalSeconds: number;       // time_limit_minutes * 60
  onExpire: () => void;       // called when reaches 0
}

interface QuestionDisplayProps {
  question: SessionQuestion;
  savedAnswer: SavedAnswer | undefined;
  onAnswer: (optionIds: string[], textAnswer: string | null) => void;
  isFlagged: boolean;
  onToggleFlag: () => void;
}

interface SessionQuestion {
  id: string;
  sort_order: number;
  stem: string;
  type: 'single_choice' | 'multiple_choice' | 'true_false' | 'likert' | 'short_text';
  options: { id: string; text: string }[];
}

interface SavedAnswer {
  selected_option_ids: string[];
  text_answer: string | null;
}
```

#### Tab-Switch Detection

```typescript
// hooks/useTabSwitchDetection.ts
function useTabSwitchDetection(
  sessionId: string,
  onAutoSubmit: () => void
) {
  const reportEvent = useReportEvent(sessionId);

  useEffect(() => {
    const handleVisibilityChange = () => {
      if (document.visibilityState === 'hidden') {
        reportEvent.mutate({ type: 'tab_switch' }, {
          onSuccess: (data) => {
            if (data.status === 'auto_submitted') onAutoSubmit();
          },
        });
      }
    };
    const handleBlur = () => {
      reportEvent.mutate({ type: 'blur' });
    };

    document.addEventListener('visibilitychange', handleVisibilityChange);
    window.addEventListener('blur', handleBlur);
    return () => {
      document.removeEventListener('visibilitychange', handleVisibilityChange);
      window.removeEventListener('blur', handleBlur);
    };
  }, [sessionId]);
}
```

#### Server-Authoritative Timer Sync

```typescript
// CountdownTimer uses remaining_seconds from last answer save response
// to re-sync every time an answer is saved, not just on page load.
const { remainingSeconds, syncFromServer } = useCountdownTimer(
  session.remaining_seconds,
  session.expires_at
);

// After each save mutation succeeds:
onSuccess: (data) => {
  syncFromServer(data.remaining_seconds);
}
```

#### i18n Key Examples

```json
{
  "exam.taking.progress": "{{answered}} / {{total}} answered",
  "exam.taking.flagButton": "Flag for review",
  "exam.taking.unflagButton": "Unflagged",
  "exam.taking.finishButton": "Finish exam",
  "exam.taking.save.saving": "Saving…",
  "exam.taking.save.saved": "Saved ✓",
  "exam.taking.save.error": "Connection lost ✗",
  "exam.taking.navigator.title": "Questions",
  "exam.taking.review.title": "Review before submitting",
  "exam.taking.review.unanswered": "Unanswered questions",
  "exam.taking.review.flagged": "Flagged for review",
  "exam.taking.review.submitAnyway": "Submit anyway",
  "exam.taking.review.goBack": "Go back",
  "exam.taking.submit.confirm.title": "Submit exam",
  "exam.taking.submit.confirm.message": "Once submitted, you cannot change your answers.",
  "exam.taking.submit.confirm.submit": "Submit",
  "exam.taking.result.passed": "Passed",
  "exam.taking.result.failed": "Failed",
  "exam.taking.result.pending": "Your exam is being reviewed by an instructor.",
  "exam.taking.result.score": "Score: {{score}}%",
  "exam.taking.tabswitch.warning": "You left the exam window. Further violations may result in automatic submission.",
  "exam.taking.timer.warning": "Less than 20% of your time remaining.",
  "exam.taking.timer.critical": "Less than 5% of your time remaining."
}
```

## Notes
- The route for this page should be `/portal/sessions/:sessionId` to allow direct navigation from "Continue" links. The `ExamTaking/index.tsx` calls `GET /api/v1/portal/sessions/:id` on mount to restore state.
- The fullscreen API (`document.documentElement.requestFullscreen()`) may optionally be requested on session start; `fullscreen_exit` events are sent if the user exits fullscreen. Fullscreen is not enforced (not all browsers permit it without user gesture).
- The navigator panel should be `sticky` on desktop (fixed right sidebar) and rendered as a bottom drawer on mobile.
- Session expiry at the client: when `remainingSeconds` reaches 0, a "Time's up" modal is displayed and `POST /submit` is called automatically; if the submission fails (network error), the client retries every 5 seconds up to 3 times.
- The result screen should check `exam.certificate_enabled && passed` and show a "Download certificate" CTA if applicable (certificate generation is Phase 4).
- Left-side question navigation: clicking a question number in the navigator scrolls to that question using `scrollIntoView({ behavior: 'smooth' })`.

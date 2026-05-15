# FR-BB314: Implementation Inner Report

**Date**: 2026-05-15T00:00:00Z
**Pipeline**: A
**Commit**: (see below)

## Summary

Implemented FR-BB314 — Frontend: Exam Taking Screen. Added a full-focus, distraction-free exam-taking UI at `/portal/sessions/:sessionId` with a server-authoritative countdown timer (amber/red thresholds at 20%/5%), immediate answer auto-save for radio/checkbox and 800 ms debounced auto-save for short-text inputs, a question navigator panel with answered/flagged colour states, tab-switch anti-cheat detection with event reporting to the backend, a pre-submit review screen listing unanswered and flagged questions, a post-submit result screen, and full i18n coverage in English, Kazakh, and Russian. The backend session model was extended to expose `ExamTitle` and `CertificateEnabled` via a JOIN on the exams table so the front-end can display the exam title in the top bar and conditionally show certificate information on the result screen.

## Files Changed

| File | Action |
|------|--------|
| `frontend/src/api/sessions.ts` | created |
| `frontend/src/hooks/useCountdownTimer.ts` | created |
| `frontend/src/hooks/useTabSwitchDetection.ts` | created |
| `frontend/src/pages/ExamTaking/index.tsx` | created |
| `frontend/src/pages/ExamTaking/ExamLayout.tsx` | created |
| `frontend/src/pages/ExamTaking/ExamTopBar.tsx` | created |
| `frontend/src/pages/ExamTaking/CountdownTimer.tsx` | created |
| `frontend/src/pages/ExamTaking/QuestionNavigator.tsx` | created |
| `frontend/src/pages/ExamTaking/QuestionDisplay.tsx` | created |
| `frontend/src/pages/ExamTaking/SingleChoiceInput.tsx` | created |
| `frontend/src/pages/ExamTaking/MultipleChoiceInput.tsx` | created |
| `frontend/src/pages/ExamTaking/LikertInput.tsx` | created |
| `frontend/src/pages/ExamTaking/ShortTextInput.tsx` | created |
| `frontend/src/pages/ExamTaking/SaveIndicator.tsx` | created |
| `frontend/src/pages/ExamTaking/FlagButton.tsx` | created |
| `frontend/src/pages/ExamTaking/FinishReviewScreen.tsx` | created |
| `frontend/src/pages/ExamTaking/SubmitConfirmModal.tsx` | created |
| `frontend/src/pages/ExamTaking/ResultScreen.tsx` | created |
| `frontend/src/pages/ExamTaking/TabSwitchWarningModal.tsx` | created |
| `backend/internal/sessions/model.go` | modified — added ExamTitle, CertificateEnabled fields |
| `backend/internal/sessions/repository.go` | modified — SQL JOIN with exams table to fetch new fields |
| `backend/internal/sessions/service.go` | modified — maps ExamTitle and CertificateEnabled from repo |
| `frontend/src/App.tsx` | modified — added route `/portal/sessions/:sessionId` |
| `frontend/src/locales/en.json` | modified — added `exam.taking.*` keys |
| `frontend/src/locales/kk.json` | modified — added Kazakh translations for `exam.taking.*` |
| `frontend/src/locales/ru.json` | modified — added Russian translations for `exam.taking.*` |
| `docs/requirements/FR-BB314.Frontend-exam-taking-screen.md` | modified — status Draft → implemented |

## Acceptance Criteria Verified

| AC | Description | Verified By |
|----|-------------|-------------|
| AC-1 | Exam taking screen uses standalone full-focus layout (no global sidebar) | ExamLayout.tsx — renders without sidebar; route added via App.tsx |
| AC-2 | Top bar displays exam title, server-authoritative countdown, progress X/Y answered | ExamTopBar.tsx + CountdownTimer.tsx using `remaining_seconds` from save response |
| AC-3 | Timer turns amber at ≤20% remaining, red at ≤5% remaining | useCountdownTimer.ts thresholds implemented |
| AC-4 | Selecting answer triggers `PUT /answers/:questionId`; Save/Saved/Error indicator shown | useSaveAnswer mutation in sessions.ts; SaveIndicator.tsx |
| AC-5 | 800 ms debounce for short-text; immediate save for radio/checkbox | ShortTextInput.tsx debounce; SingleChoiceInput/MultipleChoiceInput immediate onChange |
| AC-6 | Navigator grid: white=unanswered, blue=answered, yellow=flagged | QuestionNavigator.tsx colour state logic |
| AC-7 | Tab-switch via visibilitychange + blur; POST /sessions/:id/events; warning modal if warn:true | useTabSwitchDetection.ts; TabSwitchWarningModal.tsx |
| AC-8 | "Finish exam" → pre-submit review screen with unanswered/flagged lists | FinishReviewScreen.tsx; SubmitConfirmModal.tsx |
| AC-9 | Submit calls POST /sessions/:id/submit; result screen shows pass/fail/score/pending | ResultScreen.tsx; useSubmitSession mutation |
| AC-10 | Zero hardcoded strings; all text from locales JSON | en/kk/ru locale files cover all exam.taking.* keys |

## Test Results

- Backend: build passes (`go build ./...` — no errors)
- Frontend: TypeScript passes (`npx tsc --noEmit` — no errors)
- Unit tests: not applicable for this UI-only feature (no new backend business logic added)

## Migration Applied

None — no new database migrations required for this feature. The backend changes are model/repository layer additions (JOIN query and struct fields only).

## Known Limitations

- End-to-end browser tests (Playwright/Cypress) for the exam-taking flow are deferred to a dedicated QA phase.
- Auto-submit on timer expiry calls the existing submit endpoint; backend enforcement of time limit is handled by the auto-submit background job (FR-BB310).

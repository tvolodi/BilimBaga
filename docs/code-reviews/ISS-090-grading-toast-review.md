# Code Review: ISS-090 grading success toast (run issue-84)

Scope: frontend only. GradingDetailPage.tsx, GradingQueuePage.tsx, locales en/ru/kk, GradingDetailPage.test.tsx, GradingQueuePage.test.tsx.

Result: PASS

## Findings

- [Medium] GradingQueuePage.tsx (banner) — Dismiss button visible text is `&times;` while the accessible name comes from `aria-label`. Acceptable, but the banner is a hand-rolled div rather than a shadcn/ui primitive. Consistent with the existing inline toast helper in the detail page. No action required.
- [Medium] GradingQueuePage.tsx (clear-state effect) — The effect depends on `[location, navigate]`. After the replace-navigation `location.state` is null, so there is no loop. `showSuccess` is initialised lazily from state, so clearing the state does not hide the banner. Correct. A back or forward navigation to the same history entry cannot re-show the banner because the state was replaced.
- [Low] GradingDetailPage.tsx — `useToast` is still used for the error path, so no dead code. `setIsSubmitting(false)` before navigate is harmless.
- [Low] GradingQueuePage.test.tsx — The new `describe` sits before the existing one and relies on `useGradingQueue` mock reset ordering. It passes. The 5s auto-dismiss timer is not covered by a test (fake timers). Optional.
- [Low] The pre-existing queue refetch/invalidation part of AC-9 is not touched by this diff and is handled by the existing mutation hook. Not in the scope of this change.

No Critical or High findings. No secrets, no raw fetch, no `any` on API data, no `console.log`. Navigation state is typed via a cast, which is acceptable.

## Checklist (frontend)
- i18n: new key `grading.dismiss_toast` added to en, ru and kk. Existing `grading.success_toast` reused. No hardcoded visible strings (`&times;` is a glyph). OK.
- Route guards and routes: unchanged. OK.
- React Query and API layer: unchanged. OK.
- Loading, error and empty states of the queue page: unchanged. The banner renders independently of the data state. OK.
- Accessibility: `role="status"` plus an aria-labelled dismiss button. OK.
- Timer cleanup: the timeout is cleared in the effect cleanup. OK.

## AC Coverage (FR-BB47)
- AC-9: covered. The success notice appears on the queue page after the final submit, the examiner is navigated to the queue, and the notice is dismissible and one-shot. The integration test renders the detail and queue pages together and asserts the notice and its dismissal. The queue invalidation part is pre-existing behaviour.
- Other ACs: not touched by this change.

## Root-cause check
The detail page unmounted on navigate, so its local toast state was lost. Handing the flag over via router navigation state and rendering it on the destination page addresses the cause directly. Clearing the state with a replace-navigation prevents re-display on refresh or back.

Summary: Minimal, correct fix with an end-to-end regression test and complete i18n; no Critical or High issues.

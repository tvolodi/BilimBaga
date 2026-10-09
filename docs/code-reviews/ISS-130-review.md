# Code Review: ISS-130 portal header overflow at 375px

Verdict: PASS (no Critical/High findings). Static review only; live 375px check deferred to UAT (e2e AC6).

## Reasoning
- nav: `flex-wrap` + `px-2` (375 -> 359px content). Tab group `min-w-0 max-w-full` is capped to the row, so when tabs + right cluster do not fit, the cluster wraps to row 2 (`ml-auto` keeps it right-aligned) instead of overflowing. Previously no wrap and no min-w-0, causing 408/457px.
- Tab links are shrinkable flex items with `px-2` and `text-center`; text can wrap at word boundaries, so min-content is the longest word, far below 359px even in kk.
- Right cluster `shrink-0`: select 104px + gap 4px + icon-only ghost button (~36px) is about 144px, well under 359px, so it never overflows by itself even when alone on a row.
- tailwind-merge (`cn`) resolves `w-full` vs `w-[104px] sm:w-[120px]` correctly.
- Desktop (sm+): px-6, tab px-4, gap-3, select 120px restored. Added `gap-x-2`, `ml-auto`, `text-center`, `min-w-0 max-w-full` have no visible effect on a single non-wrapping row with justify-between. Unchanged.
- a11y: nav aria-label, logout aria-label, aria-hidden icon, DOM/tab order all kept. Select 104px still shows the locale label; touch target heights unchanged (h-10).
- Test: class-level assertions are brittle but legitimately guard the fix (jsdom has no layout). Issue report is accurate and consistent with the diff.

## Findings
| Severity | Finding |
|----------|---------|
| Low | At 375 in ru/kk the sticky header likely becomes two rows (taller), consuming vertical space. Acceptable trade-off; consider verifying visually. |
| Low | Unit test only asserts classes, not real overflow; the live e2e AC6 must be run in UAT to confirm scrollWidth <= 375 for en/ru/kk. |
| Info | Tabs at 320px could wrap label text onto two lines; no overflow, cosmetic only. |

No hardcoded strings, no API/Go changes, no secrets.

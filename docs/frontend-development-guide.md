# Frontend development guide

## Visual regression tests (FR-BB320 AC-11 to AC-13)

The design gallery and the page screenshots are compared with committed baselines under `frontend/e2e/*-snapshots/`.

- Run `make visual-test` to compare, and `make visual-update` to regenerate. Both run `scripts/visual/run.sh` in the Playwright image that matches `@playwright/test` in `frontend/package-lock.json`, so the browser and fonts are the same on every run.
- Baselines are generated only in that container, never on Windows or macOS. Commit the PNG files that change, and say in the PR why each changed.
- The tolerance is `maxDiffPixelRatio: 0.01`: up to 1% of the pixels may differ, the value FR-BB320 AC-11 specifies (`frontend/playwright.visual.config.ts`). Zero tolerance is a follow-up, tracked in #588 (see #587), and returns once a fix holds under load.
- The regular `npx playwright test` does not run these specs (`frontend/playwright.config.ts` ignores them), and they skip unless `VISUAL_MODE` is set. The visual suite has its own config, `frontend/playwright.visual.config.ts`.
- A value that changes between runs (a date, a relative time, a timer, a live list, a count from the clock) carries a `data-visual-mask` attribute, so the screenshot masks it. Do not mask a whole page to hide a change you should look at.

/**
 * Playwright config for the visual regression suite (FR-BB320 AC-11 to AC-13).
 *
 * Runs only in the pinned Playwright container, through scripts/visual/run.sh (test or update). Baselines are
 * never made on Windows or macOS. The suite is separate from playwright.config.ts, which CI runs unchanged.
 */
import { defineConfig, devices } from '@playwright/test'

export default defineConfig({
  testDir: './e2e',
  globalSetup: './e2e/global-setup.ts',
  fullyParallel: false,
  forbidOnly: !!process.env.CI,
  retries: 0,
  workers: 1,
  reporter: 'list',
  expect: {
    // FR-BB320 AC-11: up to 1% of the pixels may differ. Zero tolerance is tracked in #588 (see #587).
    toHaveScreenshot: { animations: 'disabled', caret: 'hide', maxDiffPixelRatio: 0.01 },
  },
  use: {
    baseURL: process.env.E2E_BASE_URL || 'http://localhost:4173',
  },
  projects: [
    // Once per run, before the page tests: the fixtures are created here, not in a beforeAll (see the setup file).
    {
      name: 'setup',
      testMatch: 'visual-fixtures.setup.ts',
    },
    {
      name: 'chromium',
      testMatch: ['design-gallery.spec.ts', 'visual-pages.spec.ts'],
      dependencies: ['setup'],
      use: { ...devices['Desktop Chrome'] },
    },
  ],
  webServer: {
    command: 'npm run preview -- --host 127.0.0.1 --port 4173 --strictPort',
    url: 'http://localhost:4173',
    reuseExistingServer: false,
    timeout: 60_000,
  },
})

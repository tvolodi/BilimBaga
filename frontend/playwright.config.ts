import { defineConfig, devices } from '@playwright/test'

export default defineConfig({
  testDir: './e2e',
  // The visual suite runs only in the pinned container, through playwright.visual.config.ts (FR-BB320 AC-11).
  testIgnore: ['**/design-gallery.spec.ts', '**/visual-pages.spec.ts'],
  fullyParallel: false,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: 1,
  reporter: 'list',
  use: {
    baseURL: process.env.E2E_BASE_URL || 'http://localhost:4173',
    trace: 'on-first-retry',
  },
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],
  webServer: {
    command: 'npm run preview',
    url: process.env.E2E_BASE_URL || 'http://localhost:4173',
    reuseExistingServer: true,
    timeout: 30000,
  },
})

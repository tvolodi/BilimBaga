/**
 * Playwright config for full visual E2E walkthrough against a real running stack.
 *
 * Prerequisites (run before executing tests):
 *   make dev          — starts DB, backend (port 8080, or BB_API_PORT), and frontend dev server (port 5173)
 *
 * Run with:
 *   npm run test:e2e:live
 *
 * Free-port run: start the stack with BB_API_PORT=18080 and export the same var for the tests
 * (E2E_API_URL overrides the full API URL).
 */
import { defineConfig, devices } from '@playwright/test'
import path from 'path'
import { fileURLToPath } from 'url'

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const ADMIN_STORAGE_STATE = path.join(__dirname, '.auth', 'admin.json')
const EMPLOYEE_STORAGE_STATE = path.join(__dirname, '.auth', 'employee.json')

export default defineConfig({
  testDir: './e2e',
  globalSetup: './e2e/global-setup.ts',
  fullyParallel: false,
  forbidOnly: !!process.env.CI,
  retries: 0,
  workers: 1,
  reporter: [
    ['list'],
    ['json', { outputFile: '../e2e-results.json' }],
    ['html', { outputFolder: 'playwright-report-live', open: 'never' }],
  ],
  use: {
    baseURL: process.env.E2E_BASE_URL || 'http://localhost:5173',
    trace: 'on',
    screenshot: 'on',
    video: 'on',
    actionTimeout: 15_000,
    navigationTimeout: 20_000,
  },
  projects: [
    {
      name: 'chromium-live-admin',
      testMatch: [
        '**/full-walkthrough.spec.ts',
        '**/admin-grading.spec.ts',
        '**/auth.spec.ts',
        '**/categories.spec.ts',
        '**/tags.spec.ts',
        '**/branding.spec.ts',
        '**/question-bank.spec.ts',
        '**/question-editor.spec.ts',
        '**/question-management.spec.ts',
        '**/exam-wizard.spec.ts',
        '**/exam-lifecycle.spec.ts',
        '**/user-management.spec.ts',
        '**/accessibility.spec.ts',
        '**/loyalty-narrative.spec.ts',
        '**/downloads-bearer.spec.ts',
        '**/grading/ai-grading.spec.ts',
      ],
      use: {
        ...devices['Desktop Chrome'],
        viewport: { width: 1440, height: 900 },
        storageState: ADMIN_STORAGE_STATE,
      },
    },
    {
      name: 'chromium-live-employee',
      testMatch: [
        '**/employee-portal.spec.ts',
        '**/exam-taking.spec.ts',
        '**/exam-result.spec.ts',
        '**/my-results.spec.ts',
      ],
      use: {
        ...devices['Desktop Chrome'],
        viewport: { width: 1440, height: 900 },
        storageState: EMPLOYEE_STORAGE_STATE,
      },
    },
  ],
})

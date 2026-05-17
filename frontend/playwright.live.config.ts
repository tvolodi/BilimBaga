/**
 * Playwright config for full visual E2E walkthrough against a real running stack.
 *
 * Prerequisites (run before executing tests):
 *   make dev          — starts DB, backend (port 8080), and frontend dev server (port 5173)
 *
 * Run with:
 *   npm run test:e2e:live
 */
import { defineConfig, devices } from '@playwright/test'
import path from 'path'
import { fileURLToPath } from 'url'

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const STORAGE_STATE = path.join(__dirname, '.auth', 'admin.json')

export default defineConfig({
  testDir: './e2e',
  testMatch: '**/full-walkthrough.spec.ts',
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
    baseURL: 'http://localhost:5173',
    trace: 'on',
    screenshot: 'on',
    video: 'on',
    actionTimeout: 15_000,
    navigationTimeout: 20_000,
    storageState: STORAGE_STATE,
  },
  projects: [
    {
      name: 'chromium-live',
      use: { ...devices['Desktop Chrome'], viewport: { width: 1440, height: 900 } },
    },
  ],
})

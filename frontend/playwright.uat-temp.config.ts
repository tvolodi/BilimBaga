import { defineConfig, devices } from '@playwright/test';
import path from 'path';
import { fileURLToPath } from 'url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));

export default defineConfig({
  testDir: './e2e/uat-temp',
  fullyParallel: false,
  retries: 0,
  workers: 1,
  reporter: [
    ['list'],
    ['json', { outputFile: '../e2e-uat-rerun-results.json' }],
  ],
  use: {
    baseURL: 'http://localhost',
    trace: 'on',
    screenshot: 'on',
    actionTimeout: 20_000,
    navigationTimeout: 25_000,
  },
  projects: [
    {
      name: 'chromium-uat',
      use: { ...devices['Desktop Chrome'] },
    },
  ],
});

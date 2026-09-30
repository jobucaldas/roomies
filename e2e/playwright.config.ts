import { defineConfig, devices } from '@playwright/test';

export default defineConfig({
  testDir: './tests',
  fullyParallel: false,
  // Flutter web bootstrap + auth config can exceed the default 30s.
  timeout: 90_000,
  expect: { timeout: 15_000 },
  reporter: [['list'], ['html', { outputFolder: 'artifacts/report', open: 'never' }]],
  outputDir: 'artifacts/test-results',
  use: {
    baseURL: process.env.ROOMIES_WEB_URL ?? 'http://localhost',
    launchOptions: { executablePath: process.env.CHROMIUM_PATH },
    screenshot: 'only-on-failure',
    trace: 'retain-on-failure',
  },
  projects: [
    { name: 'desktop', use: { ...devices['Desktop Chrome'], viewport: { width: 1280, height: 900 } } },
    { name: 'narrow', use: { ...devices['iPhone 13'], browserName: 'chromium' } },
  ],
});

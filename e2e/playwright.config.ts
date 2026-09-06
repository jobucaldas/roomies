import { defineConfig, devices } from '@playwright/test';

export default defineConfig({
  testDir: './tests',
  fullyParallel: false,
  reporter: [['list'], ['html', { outputFolder: 'artifacts/report', open: 'never' }]],
  outputDir: 'artifacts/test-results',
  use: {
    baseURL: process.env.ROOMIES_WEB_URL ?? 'http://localhost',
    screenshot: 'only-on-failure',
    trace: 'retain-on-failure',
  },
  projects: [
    { name: 'desktop', use: { ...devices['Desktop Chrome'], viewport: { width: 1280, height: 900 } } },
    { name: 'narrow', use: { ...devices['iPhone 13'] } },
  ],
});

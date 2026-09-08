import { createHash } from 'node:crypto';
import { execFileSync } from 'node:child_process';
import { mkdir, writeFile } from 'node:fs/promises';
import { expect, test, type APIRequestContext, type Page, type TestInfo } from '@playwright/test';

const api = process.env.ROOMIES_API_URL ?? 'http://localhost:8080/api';
const web = process.env.ROOMIES_WEB_URL ?? 'http://localhost';

type Evidence = {
  consoleErrors: string[];
  pageErrors: string[];
  failedRequests: string[];
  unexpectedResponses: string[];
  assetHashes: Record<string, string>;
  assetBodies: Promise<void>[];
};

const evidence = new WeakMap<Page, Evidence>();
const slug = (value: string) => value.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '');
const sanitizedRequest = (method: string, url: string) => `${method} ${new URL(url).pathname
  .replace(/\/houses\/[^/]+/, '/houses/:id')
  .replace(/\/(groceries|chores|calendar|chat)\/[^/]+/, '/$1/:id')}`;

function assetPath(url: string) {
  const path = new URL(url).pathname;
  return path === '/' || path.endsWith('.html') || path.endsWith('.js') || path.endsWith('.wasm') ? path : null;
}

test.beforeEach(async ({ page }) => {
  const value: Evidence = {
    consoleErrors: [], pageErrors: [], failedRequests: [], unexpectedResponses: [], assetHashes: {}, assetBodies: [],
  };
  evidence.set(page, value);
  page.on('console', message => { if (message.type() === 'error') value.consoleErrors.push('console error'); });
  page.on('pageerror', () => value.pageErrors.push('page error'));
  page.on('requestfailed', request => value.failedRequests.push(`${sanitizedRequest(request.method(), request.url())}: ${request.failure()?.errorText ?? 'unknown failure'}`));
  page.on('response', response => {
    const request = response.request();
    if (response.status() >= 400) value.unexpectedResponses.push(`${sanitizedRequest(request.method(), response.url())}: ${response.status()}`);
    const path = assetPath(response.url());
    if (path) value.assetBodies.push((async () => {
      try { value.assetHashes[path] = createHash('sha256').update(await response.body()).digest('hex'); } catch { /* response can be unavailable after navigation */ }
    })());
  });
});

test.afterEach(async ({ page }, info: TestInfo) => {
  const value = evidence.get(page)!;
  await Promise.allSettled(value.assetBodies);
  const head = execFileSync('git', ['rev-parse', 'HEAD'], { encoding: 'utf8' }).trim();
  const configuredCommit = process.env.GIT_COMMIT;
  const horizontalOverflow = await page.evaluate(() => document.documentElement.scrollWidth > innerWidth);
  const assets = Object.keys(value.assetHashes);
  expect(configuredCommit, 'GIT_COMMIT must be configured for exact-head evidence').toBe(head);
  expect(assets.some(path => path === '/' || path.endsWith('.html')), 'served HTML hash is required').toBeTruthy();
  expect(assets.some(path => path.endsWith('.js')), 'served JavaScript hash is required').toBeTruthy();
  expect(assets.some(path => path.endsWith('.wasm')), 'served WASM hash is required').toBeTruthy();
  expect(value.consoleErrors).toEqual([]);
  expect(value.pageErrors).toEqual([]);
  expect(value.failedRequests).toEqual([]);
  expect(value.unexpectedResponses).toEqual([]);
  expect(horizontalOverflow).toBeFalsy();
  await mkdir('artifacts/evidence', { recursive: true });
  const name = `${info.project.name}-household-${slug(info.title)}`;
  // The chat interaction deletes its synthetic message before this screenshot so evidence never contains a chat body.
  await page.screenshot({ path: `artifacts/evidence/${name}.png`, fullPage: true });
  await writeFile(`artifacts/evidence/${name}.json`, JSON.stringify({
    head,
    command: process.env.ROOMIES_E2E_COMMAND ?? 'npx playwright test',
    project: info.project.name,
    test: info.title,
    passCount: info.status === 'passed' ? 1 : 0,
    consoleErrorCount: value.consoleErrors.length,
    pageErrorCount: value.pageErrors.length,
    failedRequestCount: value.failedRequests.length,
    unexpectedResponseCount: value.unexpectedResponses.length,
    horizontalOverflow,
    assetHashes: value.assetHashes,
  }, null, 2) + '\n');
});

async function register(request: APIRequestContext) {
  const email = `household-${Date.now()}-${Math.random().toString(16).slice(2)}@example.test`;
  const response = await request.post(`${api}/auth/register`, { data: { name: 'Household user', email, password: 'synthetic-password-123' } });
  expect(response.ok()).toBeTruthy();
  return { email, ...(await response.json() as { token: string }) };
}

async function login(page: Page, email: string) {
  await page.goto(web);
  await page.getByPlaceholder('Email').fill(email);
  await page.getByPlaceholder('Password').fill('synthetic-password-123');
  await page.getByRole('button', { name: 'Login' }).click();
  await expect(page).toHaveURL(/\/dashboard$/);
}

async function household(page: Page, request: APIRequestContext) {
  const user = await register(request);
  const response = await request.post(`${api}/houses`, { headers: { Authorization: `Bearer ${user.token}` }, data: { name: 'Household validation' } });
  expect(response.ok()).toBeTruthy();
  const { id } = await response.json() as { id: string };
  await login(page, user.email);
  await page.goto(`${web}/house/${id}`);
  await expect(page.getByRole('heading', { name: 'Household validation' })).toBeVisible();
}

async function open(page: Page, tab: string, heading: string) {
  await page.getByRole('tab', { name: tab }).click();
  await expect(page.getByRole('heading', { name: heading })).toBeVisible();
}

test('household groceries chores calendar and chat interactions', async ({ page, request }) => {
  await household(page, request);

  await open(page, 'Groceries', 'Groceries');
  await page.getByLabel('Name').fill('Validation grocery');
  await page.getByRole('button', { name: 'Add grocery' }).click();
  await expect(page.getByText('Grocery saved.')).toBeVisible();
  await page.getByRole('button', { name: 'Check' }).click();
  await expect(page.getByText('Checked')).toBeVisible();
  await page.getByRole('button', { name: 'Delete' }).click();
  await expect(page.getByText('Grocery deleted.')).toBeVisible();

  await open(page, 'Chores', 'Chores');
  await page.getByLabel('Title').fill('Validation chore');
  await page.getByLabel('Due local').fill('2030-01-07T09:00');
  await page.getByRole('button', { name: 'Create chore' }).click();
  await expect(page.getByText('Chore saved.')).toBeVisible();
  await page.getByRole('button', { name: 'Disable' }).click();
  await expect(page.getByText('Chore enabled state saved.')).toBeVisible();
  await page.getByRole('button', { name: 'Delete' }).click();
  await expect(page.getByText('Chore deleted.')).toBeVisible();

  await open(page, 'Calendar', 'Calendar');
  await page.getByLabel('Title').fill('Validation calendar event');
  await page.getByLabel('Start local').fill('2030-01-07T09:00');
  await page.getByLabel('End local').fill('2030-01-07T10:00');
  await page.getByRole('button', { name: 'Create calendar event' }).click();
  await expect(page.getByText('Calendar event saved.')).toBeVisible();
  await page.getByRole('button', { name: 'Delete' }).click();
  await expect(page.getByText('Calendar event deleted.')).toBeVisible();

  await open(page, 'Chat', 'Chat');
  await page.getByLabel('Message').fill('Synthetic message');
  await page.getByRole('button', { name: 'Send message' }).click();
  await expect(page.getByText('Message sent.')).toBeVisible();
  await page.getByRole('button', { name: 'Edit message' }).click();
  await page.getByLabel('Message').fill('Synthetic edited message');
  await page.getByRole('button', { name: 'Save message' }).click();
  await expect(page.getByText('Message edited.')).toBeVisible();
  await expect(page.getByText('Synthetic edited message', { exact: true })).toBeVisible();
  await page.getByRole('button', { name: 'Delete message' }).click();
  await expect(page.getByText('Message deleted.')).toBeVisible();
  await page.getByRole('button', { name: 'Refresh chat' }).click();
  await expect(page.getByText('Chat refreshed.')).toBeVisible();
});

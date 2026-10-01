import { createHash } from 'node:crypto';
import { execFileSync } from 'node:child_process';
import { mkdir, writeFile } from 'node:fs/promises';
import { expect, test, type APIRequestContext, type Page, type TestInfo } from '@playwright/test';
import { fillFlutterText, loginViaUiOrToken, openHouseViaUi } from '../helpers/login';

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
  .replace(/\/(groceries|chores|calendar)\/[^/]+/, '/$1/:id')}`;

function assetPath(url: string) {
  const path = new URL(url).pathname;
  return path === '/' || path.endsWith('.html') || path.endsWith('.js') ? path : null;
}

function hasWebClientBundle(assetHashes: Record<string, string>) {
  return Object.keys(assetHashes).some(path => path.endsWith('.js'));
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
      try {
        const body = await Promise.race([
          response.body(),
          new Promise<Buffer>((_, reject) => setTimeout(() => reject(new Error('body timeout')), 5000)),
        ]);
        value.assetHashes[path] = createHash('sha256').update(body).digest('hex');
      } catch { /* response can be unavailable after navigation */ }
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
  expect(hasWebClientBundle(value.assetHashes), 'served JavaScript hash is required').toBeTruthy();
  expect(value.consoleErrors).toEqual([]);
  expect(value.pageErrors).toEqual([]);
  expect(value.failedRequests).toEqual([]);
  expect(value.unexpectedResponses).toEqual([]);
  expect(horizontalOverflow).toBeFalsy();
  await mkdir('artifacts/evidence', { recursive: true });
  const name = `${info.project.name}-household-${slug(info.title)}`;
  // Household evidence omits free-text message bodies.
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

async function login(page: Page, email: string, token?: string) {
  await loginViaUiOrToken(page, {
    web,
    email,
    password: 'synthetic-password-123',
    token,
  });
}

async function household(page: Page, request: APIRequestContext) {
  const user = await register(request);
  const response = await request.post(`${api}/houses`, { headers: { Authorization: `Bearer ${user.token}` }, data: { name: 'Household validation' } });
  expect(response.ok()).toBeTruthy();
  const { id } = await response.json() as { id: string };
  await login(page, user.email, user.token);
  await openHouseViaUi(page, {
    web,
    houseId: id,
    houseName: 'Household validation',
  });
}

async function open(page: Page, tab: string, heading: string) {
  await page.getByRole('button', { name: tab, exact: true }).click();
  const panel = page.getByRole('group', { name: heading, exact: true });
  await expect(panel).toBeVisible();
  return panel;
}

test('household groceries chores and calendar interactions', async ({ page, request }) => {
  await household(page, request);

  const groceries = await open(page, 'Groceries', 'Groceries');
  await fillFlutterText(groceries.getByLabel('Name'), 'Validation grocery');
  await groceries.getByRole('button', { name: 'Add grocery' }).click();
  await expect(groceries.getByText('Grocery saved.')).toBeVisible();
  await groceries.getByRole('button', { name: 'Check' }).click();
  await expect(groceries.getByText('Checked')).toBeVisible();
  await groceries.getByRole('button', { name: 'Delete' }).click();
  await expect(groceries.getByText('Grocery deleted.')).toBeVisible();

  const chores = await open(page, 'Chores', 'Chores');
  await fillFlutterText(chores.getByLabel('Title'), 'Validation chore');
  await fillFlutterText(chores.getByLabel('Due local'), '2030-01-07T09:00');
  await chores.getByRole('button', { name: 'Create chore' }).click();
  await expect(chores.getByText('Chore saved.')).toBeVisible();
  await chores.getByRole('button', { name: 'Disable' }).click();
  await expect(chores.getByText('Chore enabled state saved.')).toBeVisible();
  await chores.getByRole('button', { name: 'Delete' }).click();
  await expect(chores.getByText('Chore deleted.')).toBeVisible();

  const calendar = await open(page, 'Calendar', 'Calendar');
  await fillFlutterText(calendar.getByLabel('Title'), 'Validation calendar event');
  await fillFlutterText(calendar.getByLabel('Start local'), '2030-01-07T09:00');
  await fillFlutterText(calendar.getByLabel('End local'), '2030-01-07T10:00');
  await calendar.getByRole('button', { name: 'Create calendar event' }).click();
  await expect(calendar.getByText('Calendar event saved.')).toBeVisible();
  await calendar.getByRole('button', { name: 'Delete' }).click();
  await expect(calendar.getByText('Calendar event deleted.')).toBeVisible();
});

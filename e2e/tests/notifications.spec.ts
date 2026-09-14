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
  deleteResponses: string[];
  assetHashes: Record<string, string>;
  assetBodies: Promise<void>[];
};

const evidence = new WeakMap<Page, Evidence>();
const slug = (value: string) => value.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '');
const sanitizedRequest = (method: string, url: string) => `${method} ${new URL(url).pathname
  .replace(/\/houses\/[^/]+/, '/houses/:id')
  .replace(/\/scheduled-events\/[^/]+/, '/scheduled-events/:id')}`;

function assetPath(url: string) {
  const path = new URL(url).pathname;
  return path === '/' || path.endsWith('.html') || path.endsWith('.js') || path.endsWith('.wasm') ? path : null;
}

test.beforeEach(async ({ page }) => {
  const value: Evidence = {
    consoleErrors: [], pageErrors: [], failedRequests: [], unexpectedResponses: [], deleteResponses: [], assetHashes: {}, assetBodies: [],
  };
  evidence.set(page, value);
  page.on('console', message => { if (message.type() === 'error') value.consoleErrors.push('console error'); });
  page.on('pageerror', () => value.pageErrors.push('page error'));
  page.on('requestfailed', request => value.failedRequests.push(`${sanitizedRequest(request.method(), request.url())}: ${request.failure()?.errorText ?? 'unknown failure'}`));
  page.on('response', response => {
    const request = response.request();
    const requestSummary = `${sanitizedRequest(request.method(), response.url())}: ${response.status()}`;
    if (response.status() >= 400) value.unexpectedResponses.push(requestSummary);
    if (request.method() === 'DELETE') value.deleteResponses.push(requestSummary);
    const path = assetPath(response.url());
    if (path) value.assetBodies.push((async () => {
      try { value.assetHashes[path] = createHash('sha256').update(await response.body()).digest('hex'); } catch { /* navigation responses can be unavailable */ }
    })());
  });
});

test.afterEach(async ({ page }, info: TestInfo) => {
  const value = evidence.get(page)!;
  await Promise.allSettled(value.assetBodies);
  const horizontalOverflow = await page.evaluate(() => document.documentElement.scrollWidth > innerWidth);
  await mkdir('artifacts/evidence', { recursive: true });
  const name = `${info.project.name}-notifications-${slug(info.title)}`;
  await page.screenshot({ path: `artifacts/evidence/${name}.png`, fullPage: true });
  await writeFile(`artifacts/evidence/${name}.json`, JSON.stringify({
    head: process.env.GIT_COMMIT ?? execFileSync('git', ['rev-parse', 'HEAD'], { encoding: 'utf8' }).trim(),
    command: process.env.ROOMIES_E2E_COMMAND ?? 'npx playwright test',
    project: info.project.name,
    test: info.title,
    passCount: info.status === 'passed' ? 1 : 0,
    consoleErrorCount: value.consoleErrors.length,
    pageErrorCount: value.pageErrors.length,
    failedRequestCount: value.failedRequests.length,
    unexpectedResponseCount: value.unexpectedResponses.length,
    deleteResponses: value.deleteResponses,
    horizontalOverflow,
    assetHashes: value.assetHashes,
  }, null, 2) + '\n');
  expect(value.consoleErrors).toEqual([]);
  expect(value.pageErrors).toEqual([]);
  expect(value.failedRequests).toEqual([]);
  expect(value.unexpectedResponses).toEqual([]);
  expect(horizontalOverflow).toBeFalsy();
});

async function register(request: APIRequestContext, label: string) {
  const email = `${label}-${Date.now()}-${Math.random().toString(16).slice(2)}@example.test`;
  const response = await request.post(`${api}/auth/register`, { data: { name: label, email, password: 'synthetic-password-123' } });
  expect(response.ok()).toBeTruthy();
  return { email, ...(await response.json() as { token: string; user: { id: string } }) };
}

async function house(request: APIRequestContext) {
  const admin = await register(request, 'Admin');
  const response = await request.post(`${api}/houses`, { headers: { Authorization: `Bearer ${admin.token}` }, data: { name: 'Notification House' } });
  expect(response.ok()).toBeTruthy();
  return { admin, id: (await response.json() as { id: string }).id };
}

async function login(page: Page, email: string) {
  await page.goto(web);
  await page.getByPlaceholder('Email').fill(email);
  await page.getByPlaceholder('Password').fill('synthetic-password-123');
  await page.getByRole('button', { name: 'Login' }).click();
  await expect(page).toHaveURL(/\/dashboard$/);
}

async function notifications(page: Page, id: string) {
  await page.goto(`${web}/house/${id}`);
  await page.getByRole('tab', { name: 'Notifications / Schedule' }).click();
  await expect(page.getByRole('heading', { name: 'Notifications & schedule' })).toBeVisible();
  await expect(page.getByText('Loading notification settings…')).toHaveCount(0);
  await expect(page.getByLabel('Shared expense alerts')).toBeChecked();
}

test('preferences reload and public VAPID unavailable state are explicit', async ({ page, request }) => {
  const data = await house(request);
  await login(page, data.admin.email);
  await notifications(page, data.id);
  await page.getByLabel('Shared expense alerts').click();
  await expect(page.getByLabel('Shared expense alerts')).not.toBeChecked();
  await page.getByLabel('Scheduled reminder alerts').click();
  await expect(page.getByLabel('Scheduled reminder alerts')).not.toBeChecked();
  await page.getByRole('button', { name: 'Save preferences' }).click();
  await expect(page.getByText('Notification preferences saved.')).toBeVisible();
  await page.reload();
  await page.getByRole('tab', { name: 'Notifications / Schedule' }).click();
  await expect(page.getByLabel('Shared expense alerts')).not.toBeChecked();
  await expect(page.getByLabel('Scheduled reminder alerts')).not.toBeChecked();
  const serviceWorker = await page.request.get(`${web}/roomies-sw.js`);
  expect(serviceWorker.ok()).toBeTruthy();
  const serviceWorkerSource = await serviceWorker.text();
  expect(serviceWorkerSource).toContain('`/house/${encodeURIComponent(');
  expect(serviceWorkerSource).not.toContain('/houses/');
  evidence.get(page)!.assetHashes['/roomies-sw.js'] = createHash('sha256').update(serviceWorkerSource).digest('hex');
  const key = await request.get(`${api}/notifications/vapid-public-key`);
  expect(key.ok()).toBeTruthy();
  expect((await key.json()).public_key).toBe('');
  await page.getByRole('button', { name: 'Enable browser push' }).click();
  await expect(page.getByText('Browser push is not configured by this server.')).toBeVisible();
});

test('member creates edits and deletes a scheduled event', async ({ page, request }) => {
  const data = await house(request);
  const member = await register(request, 'Member');
  const joined = await request.post(`${api}/houses/${data.id}/members`, { headers: { Authorization: `Bearer ${data.admin.token}` }, data: { user_id: member.user.id, role: 'member' } });
  expect(joined.ok()).toBeTruthy();
  await login(page, member.email);
  await notifications(page, data.id);
  await page.getByLabel('Title').fill('Bins');
  await page.getByLabel('Local start').fill('2025-02-01T09:00');
  await page.getByRole('button', { name: 'Create scheduled event' }).click();
  await expect(page.getByText('Scheduled event created.')).toBeVisible();
  await page.getByRole('button', { name: 'Edit' }).click();
  await expect(page.getByRole('heading', { name: 'Edit scheduled event' })).toBeVisible();
  await page.getByLabel('Title').fill('Recycling');
  await page.getByRole('button', { name: 'Save scheduled event' }).click();
  await expect(page.getByText('Scheduled event updated.')).toBeVisible();
  await expect(page.getByText('Recycling')).toBeVisible();
  await page.getByRole('button', { name: 'Delete' }).click();
  await expect(page.getByText('Scheduled event deleted.')).toBeVisible();
  await expect(page.getByText('Recycling')).toHaveCount(0);
  expect(evidence.get(page)!.deleteResponses).toContain('DELETE /api/houses/:id/scheduled-events/:id: 204');
});

test('monitor is view-only and recurrence feedback is local', async ({ page, request }) => {
  const data = await house(request);
  const monitor = await register(request, 'Monitor');
  const joined = await request.post(`${api}/houses/${data.id}/members`, { headers: { Authorization: `Bearer ${data.admin.token}` }, data: { user_id: monitor.user.id, role: 'monitor' } });
  expect(joined.ok()).toBeTruthy();
  await login(page, monitor.email);
  await notifications(page, data.id);
  await expect(page.getByText('Monitors can view scheduled events but cannot create, edit, or delete them.')).toBeVisible();
  await expect(page.getByRole('button', { name: 'Create scheduled event' })).toHaveCount(0);
});

test('invalid recurrence and EXDATE receive local feedback', async ({ page, request }) => {
  const data = await house(request);
  await login(page, data.admin.email);
  await notifications(page, data.id);
  await page.getByLabel('Title').fill('Bins');
  await page.getByLabel('Local start').fill('2025-02-01T09:00');
  await page.getByLabel('Interval (1–366)').fill('367');
  await page.getByRole('button', { name: 'Create scheduled event' }).click();
  await expect(page.getByText('Interval must be 1–366.')).toBeVisible();
  await page.getByLabel('Interval (1–366)').fill('1');
  await page.getByLabel('EXDATE local times (comma-separated)').fill(Array.from({ length: 101 }, (_, i) => `2025-02-${String((i % 28) + 1).padStart(2, '0')}T09:00`).join(','));
  await page.getByRole('button', { name: 'Create scheduled event' }).click();
  await expect(page.getByText('At most 100 exception dates are allowed.')).toBeVisible();
});

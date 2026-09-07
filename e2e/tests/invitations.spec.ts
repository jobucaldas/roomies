import { createHash } from 'node:crypto';
import { execFileSync } from 'node:child_process';
import { mkdir, writeFile } from 'node:fs/promises';
import { expect, test as base, type APIRequestContext, type Page, type TestInfo } from '@playwright/test';

const test = base;

type BrowserEvidence = {
  consoleErrors: string[];
  pageErrors: string[];
  failedRequests: string[];
  assetHashes: Record<string, string>;
  assetBodies: Promise<void>[];
  expectedInvitationNotFound: boolean;
};
const evidenceByPage = new WeakMap<Page, BrowserEvidence>();
const slug = (title: string) => title.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '');

// There are currently no intentionally aborted application requests. Keep this explicit so
// adding an exception requires naming the exact request and its reason in review.
function intentionallyAborted(_url: string, _error: string | undefined): boolean { return false; }

async function writeBrowserEvidence(page: Page, testInfo: TestInfo) {
  const evidence = evidenceByPage.get(page);
  if (!evidence) return;
  await Promise.allSettled(evidence.assetBodies);
  const output = testInfo.project.name + '-' + slug(testInfo.title);
  const directory = 'artifacts/evidence';
  await mkdir(directory, { recursive: true });
  await page.screenshot({ path: `${directory}/${output}.png`, fullPage: true });
  const horizontalOverflow = await page.evaluate(
    () => document.documentElement.scrollWidth > window.innerWidth,
  );
  const summary = {
    head: process.env.GIT_COMMIT ?? execFileSync('git', ['rev-parse', 'HEAD'], { encoding: 'utf8' }).trim(),
    command: process.env.ROOMIES_E2E_COMMAND ?? 'npx playwright test',
    project: testInfo.project.name,
    test: testInfo.title,
    passCount: testInfo.status === 'passed' ? 1 : 0,
    consoleErrorCount: evidence.consoleErrors.length,
    pageErrorCount: evidence.pageErrors.length,
    failedRequestCount: evidence.failedRequests.length,
    horizontalOverflow,
    assetHashes: evidence.assetHashes,
  };
  await writeFile(`${directory}/${output}.json`, JSON.stringify(summary, null, 2) + '\n');
  if (
    evidence.consoleErrors.length
    || evidence.pageErrors.length
    || evidence.failedRequests.length
    || horizontalOverflow
  ) {
    throw new Error(`browser evidence failures: console=${evidence.consoleErrors.length}, page=${evidence.pageErrors.length}, network=${evidence.failedRequests.length}, overflow=${horizontalOverflow}`);
  }
}

test.beforeEach(async ({ page }) => {
  const evidence: BrowserEvidence = {
    consoleErrors: [],
    pageErrors: [],
    failedRequests: [],
    assetHashes: {},
    assetBodies: [],
    expectedInvitationNotFound: false,
  };
  evidenceByPage.set(page, evidence);
  page.on('console', (message) => {
    if (message.type() !== 'error') return;
    // The enumeration-safe wrong-account response is intentionally 404. Chromium
    // mirrors that expected API denial to the console even though the app handles it.
    if (
      evidence.expectedInvitationNotFound
      && message.text().includes('Failed to load resource: the server responded with a status of 404')
    ) {
      evidence.expectedInvitationNotFound = false;
      return;
    }
    evidence.consoleErrors.push('console error');
  });
  page.on('pageerror', () => evidence.pageErrors.push('page error'));
  page.on('requestfailed', (request) => {
    const error = request.failure()?.errorText;
    if (!intentionallyAborted(request.url(), error)) evidence.failedRequests.push('request failed');
  });
  page.on('response', (response) => {
    const url = new URL(response.url());
    if (!['.html', '.js', '.wasm'].some((extension) => url.pathname.endsWith(extension)) && url.pathname !== '/') return;
    evidence.assetBodies.push((async () => {
      try {
        evidence.assetHashes[url.pathname] = createHash('sha256').update(await response.body()).digest('hex');
      } catch { /* response may be unavailable after a failed navigation */ }
    })());
  });
});

test.afterEach(async ({ page }, testInfo) => writeBrowserEvidence(page, testInfo));

const apiURL = process.env.ROOMIES_API_URL ?? 'http://localhost:8080/api';
const mailpitURL = process.env.MAILPIT_URL ?? 'http://localhost:8025';

type Fixture = { admin: string; house: string; invitation: string; token: string; email: string };

async function register(request: APIRequestContext, email: string, name: string) {
  const response = await request.post(`${apiURL}/auth/register`, {
    data: { name, email, password: 'synthetic-password-123' },
  });
  expect(response.ok()).toBeTruthy();
  return (await response.json()) as { token: string; user: { id: string } };
}

async function fixture(request: APIRequestContext): Promise<Fixture> {
  const suffix = `${Date.now()}-${Math.random().toString(16).slice(2)}`;
  const email = `invitee-${suffix}@example.test`;
  const admin = await register(request, `admin-${suffix}@example.test`, 'Synthetic Admin');
  const houseResponse = await request.post(`${apiURL}/houses`, {
    headers: { Authorization: `Bearer ${admin.token}` }, data: { name: 'Synthetic House' },
  });
  expect(houseResponse.ok()).toBeTruthy();
  const house = (await houseResponse.json()) as { id: string };
  const invitationResponse = await request.post(`${apiURL}/houses/${house.id}/invites`, {
    headers: { Authorization: `Bearer ${admin.token}`, 'Idempotency-Key': `invite-${suffix}` },
    data: { email, role: 'member' },
  });
  expect(invitationResponse.ok()).toBeTruthy();
  const invitation = (await invitationResponse.json()) as { id: string; manual_acceptance_url: string };
  let message = '';
  await expect.poll(async () => {
    const search = await request.get(`${mailpitURL}/api/v1/search?query=to:${email}`);
    if (!search.ok()) return '';
    const data = await search.json() as { messages?: Array<{ ID: string }> };
    if (!data.messages?.[0]) return '';
    const body = await request.get(`${mailpitURL}/api/v1/message/${data.messages[0].ID}`);
    if (!body.ok()) return '';
    const detail = await body.json() as { Text?: unknown; HTML?: unknown };
    message = typeof detail.Text === 'string' ? detail.Text : typeof detail.HTML === 'string' ? detail.HTML : '';
    return message;
  }, { timeout: 15_000 }).not.toBe('');
  const token = (message.match(/accept-invitation\?token=([^\s]+)/)?.[1] ?? '').replace(/[>.)]+$/, '');
  expect(token).not.toBe('');
  return { admin: admin.token, house: house.id, invitation: invitation.id, token, email };
}

async function login(page: Page, email: string) {
  await expect(page.getByRole('heading', { name: 'Login' })).toBeVisible();
  await page.getByPlaceholder('Email').fill(email);
  await page.getByPlaceholder('Password').fill('synthetic-password-123');
  await page.getByRole('button', { name: 'Login' }).click();
}

test('admin invite is delivered and intended user joins once', async ({ page, request }) => {
  const data = await fixture(request);
  const invitee = await register(request, data.email, 'Synthetic Invitee');
  await page.goto(`${process.env.ROOMIES_WEB_URL ?? 'http://localhost'}/accept-invitation?token=${data.token}`);
  await expect.poll(() => page.evaluate(() => sessionStorage.getItem('roomies.pending.invitation') !== null)).toBe(true);
  await login(page, data.email);
  await expect(page).toHaveURL(new RegExp(`/house/${data.house}`));
  const retry = await request.post(`${apiURL}/invitations/accept`, {
    headers: { Authorization: `Bearer ${invitee.token}` }, data: { token: data.token },
  });
  expect(retry.ok()).toBeTruthy();
  const members = await request.get(`${apiURL}/houses/${data.house}/members`, {
    headers: { Authorization: `Bearer ${invitee.token}` },
  });
  expect(members.ok()).toBeTruthy();
  const joined = (await members.json() as Array<{ user_id: string }>).filter((member) => member.user_id === invitee.user.id);
  expect(joined).toHaveLength(1);
});

test('wrong account, revoked link, and monitor invite are denied', async ({ page, request }) => {
  const data = await fixture(request);
  const wrongEmail = `wrong-${Date.now()}@example.test`;
  const wrong = await register(request, wrongEmail, 'Wrong Account');
  await page.goto(`${process.env.ROOMIES_WEB_URL ?? 'http://localhost'}/accept-invitation?token=${data.token}`);
  evidenceByPage.get(page)!.expectedInvitationNotFound = true;
  await login(page, wrongEmail);
  await expect(page.getByRole('alert')).toContainText('Unable to accept');
  const wrongDenied = await request.post(`${apiURL}/invitations/accept`, {
    headers: { Authorization: `Bearer ${wrong.token}` }, data: { token: data.token },
  });
  expect(wrongDenied.status()).toBe(404);
  const revoke = await request.delete(`${apiURL}/houses/${data.house}/invites/${data.invitation}`, {
    headers: { Authorization: `Bearer ${data.admin}` },
  });
  expect(revoke.ok()).toBeTruthy();
  const denied = await request.post(`${apiURL}/invitations/accept`, {
    headers: { Authorization: `Bearer ${wrong.token}` }, data: { token: data.token },
  });
  expect(denied.status()).toBe(404);
});

test('monitor cannot invite', async ({ request }) => {
  const data = await fixture(request);
  const monitorEmail = `monitor-${Date.now()}@example.test`;
  const monitor = await register(request, monitorEmail, 'Synthetic Monitor');
  const member = await request.post(`${apiURL}/houses/${data.house}/members`, {
    headers: { Authorization: `Bearer ${data.admin}` },
    data: { user_id: monitor.user.id, role: 'monitor' },
  });
  expect(member.ok()).toBeTruthy();
  const response = await request.post(`${apiURL}/houses/${data.house}/invites`, {
    headers: { Authorization: `Bearer ${monitor.token}` },
    data: { email: `blocked-${Date.now()}@example.test`, role: 'member' },
  });
  expect(response.status()).toBe(403);
});

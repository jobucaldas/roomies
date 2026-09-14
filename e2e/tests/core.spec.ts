import { execFileSync } from 'node:child_process';
import { createHash } from 'node:crypto';
import { mkdir, writeFile } from 'node:fs/promises';
import { expect, test, type APIRequestContext, type Page, type TestInfo } from '@playwright/test';

// Core tests deliberately disable Playwright screenshots and traces. The sanitized JSON record
// below is sufficient for the CI gate and cannot retain expense, note, or session contents.
test.use({ screenshot: 'off', trace: 'off' });

const api = process.env.ROOMIES_API_URL ?? 'http://localhost:8080/api';
const web = process.env.ROOMIES_WEB_URL ?? 'http://localhost';
const password = 'synthetic-password-123';

type Account = {
  email: string;
  token: string;
  user: { id: string };
  name: string;
};

type Fixture = {
  admin: Account;
  member: Account;
  unlisted?: Account;
  monitor?: Account;
  houseId: string;
};

type BrowserEvidence = {
  consoleErrors: string[];
  pageErrors: string[];
  failedRequests: string[];
  unexpectedResponses: string[];
  assetHashes: Record<string, string>;
  assetBodies: Promise<void>[];
};

const evidenceByPage = new WeakMap<Page, BrowserEvidence>();

function assetPath(url: string) {
  const path = new URL(url).pathname;
  return path === '/' || path.endsWith('.html') || path.endsWith('.js') || path.endsWith('.wasm')
    ? path
    : null;
}

function requestSummary(method: string, url: string, status?: number) {
  const path = new URL(url).pathname
    .replace(/\/houses\/[^/]+/g, '/houses/:id')
    .replace(/\/(expenses|notes)\/[^/]+/g, '/$1/:id');
  return `${method} ${path}${status === undefined ? '' : `: ${status}`}`;
}

test.beforeEach(async ({ page }) => {
  const evidence: BrowserEvidence = {
    consoleErrors: [],
    pageErrors: [],
    failedRequests: [],
    unexpectedResponses: [],
    assetHashes: {},
    assetBodies: [],
  };
  evidenceByPage.set(page, evidence);
  page.on('console', message => {
    if (message.type() === 'error') evidence.consoleErrors.push('console error');
  });
  page.on('pageerror', () => evidence.pageErrors.push('page error'));
  page.on('requestfailed', () => evidence.failedRequests.push('request failed'));
  page.on('response', response => {
    if (response.status() >= 400) {
      evidence.unexpectedResponses.push(requestSummary(response.request().method(), response.url(), response.status()));
    }
    const path = assetPath(response.url());
    if (!path) return;
    evidence.assetBodies.push((async () => {
      try {
        evidence.assetHashes[path] = createHash('sha256').update(await response.body()).digest('hex');
      } catch {
        // Navigation responses can be unavailable after a later navigation.
      }
    })());
  });
});

test.afterEach(async ({ page }, testInfo: TestInfo) => {
  const evidence = evidenceByPage.get(page)!;
  await Promise.allSettled(evidence.assetBodies);
  const head = execFileSync('git', ['rev-parse', 'HEAD'], { encoding: 'utf8' }).trim();
  const configuredCommit = process.env.GIT_COMMIT;
  let horizontalOverflow = false;
  try {
    horizontalOverflow = await page.evaluate(() => document.documentElement.scrollWidth > innerWidth);
  } catch {
    // Preserve the remaining evidence if the browser closed during a failed test.
  }
  await mkdir('artifacts/evidence', { recursive: true });
  const output = `core-${testInfo.project.name}-${slug(testInfo.title)}`;
  await writeFile(`artifacts/evidence/${output}.json`, JSON.stringify({
    head,
    command: process.env.ROOMIES_E2E_COMMAND ?? 'npx playwright test tests/core.spec.ts',
    project: testInfo.project.name,
    test: testInfo.title,
    passCount: testInfo.status === 'passed' ? 1 : 0,
    consoleErrorCount: evidence.consoleErrors.length,
    pageErrorCount: evidence.pageErrors.length,
    failedRequestCount: evidence.failedRequests.length,
    unexpectedResponseCount: evidence.unexpectedResponses.length,
    horizontalOverflow,
    assetHashes: evidence.assetHashes,
  }, null, 2) + '\n');
  // Clear rendered private data without navigating, which could abort an in-flight app request.
  await page.evaluate(() => document.body.replaceChildren()).catch(() => undefined);
  expect(configuredCommit, 'GIT_COMMIT must be configured for exact-head evidence').toBe(head);
  expect(Object.keys(evidence.assetHashes).some(path => path === '/' || path.endsWith('.html'))).toBeTruthy();
  expect(Object.keys(evidence.assetHashes).some(path => path.endsWith('.js'))).toBeTruthy();
  expect(Object.keys(evidence.assetHashes).some(path => path.endsWith('.wasm'))).toBeTruthy();
  expect(evidence.consoleErrors).toEqual([]);
  expect(evidence.pageErrors).toEqual([]);
  expect(evidence.failedRequests).toEqual([]);
  expect(evidence.unexpectedResponses).toEqual([]);
  expect(horizontalOverflow).toBeFalsy();
});

function slug(value: string) {
  return value.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '');
}

function auth(token: string) {
  return { Authorization: `Bearer ${token}` };
}

async function register(request: APIRequestContext, name: string, suffix: string): Promise<Account> {
  const email = `${name.toLowerCase().replace(/[^a-z0-9]+/g, '-')}-${suffix}@example.test`;
  const response = await request.post(`${api}/auth/register`, {
    data: { name, email, password },
  });
  expect(response.ok()).toBeTruthy();
  const result = await response.json() as { token: string; user: { id: string } };
  return { email, token: result.token, user: result.user, name };
}

async function addMember(request: APIRequestContext, admin: Account, houseId: string, account: Account, role: string) {
  const response = await request.post(`${api}/houses/${houseId}/members`, {
    headers: auth(admin.token),
    data: { user_id: account.user.id, role },
  });
  expect(response.ok()).toBeTruthy();
}

async function fixture(request: APIRequestContext, withMonitor = false): Promise<Fixture> {
  const suffix = `${Date.now()}-${Math.random().toString(16).slice(2)}`;
  const admin = await register(request, 'Core Admin', suffix);
  const member = await register(request, 'Core Member', suffix);
  const houseResponse = await request.post(`${api}/houses`, {
    headers: auth(admin.token),
    data: { name: 'Core Validation House' },
  });
  expect(houseResponse.ok()).toBeTruthy();
  const houseId = (await houseResponse.json() as { id: string }).id;
  await addMember(request, admin, houseId, member, 'member');
  if (!withMonitor) return { admin, member, houseId };
  const unlisted = await register(request, 'Core Unlisted', suffix);
  await addMember(request, admin, houseId, unlisted, 'member');
  const monitor = await register(request, 'Core Monitor', suffix);
  await addMember(request, admin, houseId, monitor, 'monitor');
  return { admin, member, unlisted, monitor, houseId };
}

async function login(page: Page, email: string) {
  await page.goto(web);
  await expect(page.getByRole('heading', { name: 'Login' })).toBeVisible();
  await page.getByPlaceholder('Email').fill(email);
  await page.getByPlaceholder('Password').fill(password);
  await page.getByRole('button', { name: 'Login' }).click();
  await expect(page).toHaveURL(/\/dashboard$/);
}

async function logout(page: Page) {
  // House pages intentionally expose only a Back action; logout is on Dashboard.
  await page.goto(`${web}/dashboard`);
  await page.getByRole('button', { name: 'Logout' }).click();
  await expect(page.getByRole('heading', { name: 'Login' })).toBeVisible();
}

async function openTab(page: Page, tab: string, heading: string) {
  await page.getByRole('tab', { name: tab }).click();
  const panel = page.getByRole('tabpanel', { name: tab });
  await expect(panel.getByRole('heading', { name: heading })).toBeVisible();
  return panel;
}

async function openHouse(page: Page, houseId: string) {
  await page.goto(`${web}/house/${houseId}`);
  await expect(page.getByRole('heading', { name: 'Core Validation House' })).toBeVisible();
}

async function cardContains(page: Page, marker: string) {
  return page.locator('article.card').evaluateAll((cards, value) => (
    cards.some(card => card.textContent?.includes(value as string) ?? false)
  ), marker);
}

async function expectCard(page: Page, marker: string, visible: boolean) {
  await expect.poll(() => cardContains(page, marker)).toBe(visible);
}

async function expectNoOverflow(page: Page) {
  expect(await page.evaluate(() => document.documentElement.scrollWidth > innerWidth)).toBe(false);
}

async function listExpenseId(request: APIRequestContext, account: Account, houseId: string, description: string) {
  const response = await request.get(`${api}/houses/${houseId}/expenses`, { headers: auth(account.token) });
  expect(response.ok()).toBeTruthy();
  const expenses = await response.json() as Array<{ id: string; description: string }>;
  return expenses.find(expense => expense.description === description)?.id ?? '';
}

test('private expense stays hidden from an unlisted member', async ({ page, request }) => {
  const data = await fixture(request, true);
  const description = 'core-private-expense';
  let expenseId = '';
  try {
    await login(page, data.admin.email);
    await openHouse(page, data.houseId);
    await openTab(page, 'Expenses', 'Expenses');
    await page.getByRole('button', { name: 'Add expense' }).click();
    await page.getByLabel('Amount').fill('21.00');
    await page.getByLabel('Description').fill(description);
    await page.getByLabel('Visibility').selectOption('private');
    await page.getByLabel(/Recipient user IDs/).fill(data.member.user.id);
    await page.getByRole('button', { name: 'Save expense' }).click();
    await expectCard(page, description, true);

    expenseId = await listExpenseId(request, data.admin, data.houseId, description);
    expect(expenseId.length > 0).toBe(true);
    const recipientList = await request.get(`${api}/houses/${data.houseId}/expenses`, {
      headers: auth(data.member.token),
    });
    expect(recipientList.ok()).toBeTruthy();
    const recipientExpenses = await recipientList.json() as Array<{ description: string }>;
    expect(recipientExpenses.some(expense => expense.description === description)).toBe(true);
    const recipientDetail = await request.get(`${api}/houses/${data.houseId}/expenses/${expenseId}`, {
      headers: auth(data.member.token),
    });
    expect(recipientDetail.status()).toBe(200);

    const unlistedList = await request.get(`${api}/houses/${data.houseId}/expenses`, {
      headers: auth(data.unlisted!.token),
    });
    expect(unlistedList.ok()).toBeTruthy();
    const unlistedExpenses = await unlistedList.json() as Array<{ description: string }>;
    expect(unlistedExpenses.some(expense => expense.description === description)).toBe(false);
    const unlistedDetail = await request.get(`${api}/houses/${data.houseId}/expenses/${expenseId}`, {
      headers: auth(data.unlisted!.token),
    });
    expect(unlistedDetail.status()).toBe(404);

    await logout(page);
    await login(page, data.unlisted!.email);
    await openHouse(page, data.houseId);
    await openTab(page, 'Expenses', 'Expenses');
    await expectCard(page, description, false);

    await logout(page);
    await login(page, data.member.email);
    await openHouse(page, data.houseId);
    await openTab(page, 'Expenses', 'Expenses');
    await expectCard(page, description, true);
    await expectNoOverflow(page);
  } finally {
    if (expenseId) await request.delete(`${api}/houses/${data.houseId}/expenses/${expenseId}`, { headers: auth(data.admin.token) });
    await page.goto(`${web}/dashboard`).catch(() => undefined);
  }
});

test('shared expense editing, deletion, and settlement suggestions are scoped to the payer', async ({ page, request }) => {
  const data = await fixture(request);
  const description = 'core-shared-expense';
  const updatedDescription = 'core-edited-expense';
  let expenseId = '';
  try {
    await login(page, data.admin.email);
    await openHouse(page, data.houseId);
    await openTab(page, 'Expenses', 'Expenses');
    await page.getByRole('button', { name: 'Add expense' }).click();
    await page.getByLabel('Amount').fill('10.00');
    await page.getByLabel('Description').fill(description);
    await page.getByLabel(/Custom splits/).fill(`${data.admin.user.id}:7.00, ${data.member.user.id}:3.00`);
    await page.getByRole('button', { name: 'Save expense' }).click();
    await expectCard(page, description, true);
    expenseId = await listExpenseId(request, data.admin, data.houseId, description);
    expect(expenseId.length > 0).toBe(true);

    const memberUpdate = await request.put(`${api}/houses/${data.houseId}/expenses/${expenseId}`, {
      headers: auth(data.member.token),
      data: { description: 'unauthorized update' },
    });
    expect(memberUpdate.status()).toBe(403);
    const memberDelete = await request.delete(`${api}/houses/${data.houseId}/expenses/${expenseId}`, {
      headers: auth(data.member.token),
    });
    expect(memberDelete.status()).toBe(403);

    await openTab(page, 'Balances', 'Balances');
    const settlement = `${data.member.name} pays ${data.admin.name} $3.00`;
    await expect.poll(() => page.locator('p').evaluateAll((paragraphs, line) => (
      paragraphs.some(paragraph => paragraph.textContent?.includes(line as string) ?? false)
    ), settlement)).toBe(true);

    await openTab(page, 'Expenses', 'Expenses');
    const card = page.locator('article.card').filter({ hasText: description });
    await card.getByRole('button', { name: 'Details / edit' }).click();
    const editor = page.getByRole('dialog');
    await expect(editor.getByRole('heading', { name: 'Expense details' })).toBeVisible();
    await editor.getByLabel('Description').fill(updatedDescription);
    await editor.getByRole('button', { name: 'Save changes' }).click();
    await expect(editor).toHaveCount(0);
    await expectCard(page, updatedDescription, true);

    const deleteResponse = page.waitForResponse(response => (
      response.request().method() === 'DELETE' && response.url().includes('/expenses/')
    ));
    const updatedCard = page.locator('article.card').filter({ hasText: updatedDescription });
    await updatedCard.getByRole('button', { name: 'Delete' }).click();
    await page.getByRole('alertdialog').getByRole('button', { name: 'Confirm delete' }).click();
    expect((await deleteResponse).status()).toBe(200);
    await expectCard(page, updatedDescription, false);

    const remaining = await request.get(`${api}/houses/${data.houseId}/expenses`, { headers: auth(data.admin.token) });
    expect(remaining.ok()).toBeTruthy();
    const remainingExpenses = await remaining.json() as Array<{ description: string }>;
    expect(remainingExpenses.some(expense => expense.description === updatedDescription)).toBe(false);
    await expectNoOverflow(page);
  } finally {
    await page.goto(`${web}/dashboard`).catch(() => undefined);
  }
});

test('member can create, edit, and delete a note', async ({ page, request }) => {
  const data = await fixture(request);
  const title = 'core-note';
  const content = 'core-note-content';
  const updatedTitle = 'core-edited-note';
  const updatedContent = 'core-edited-note-content';
  await login(page, data.member.email);
  await openHouse(page, data.houseId);
  await openTab(page, 'Notes', 'Notes');
  await expect(page.getByText('Loading notes…')).toHaveCount(0);
  await page.getByLabel('Title').fill(title);
  await page.getByLabel('Content').fill(content);
  await page.getByRole('button', { name: 'Save note' }).click();
  await expectCard(page, title, true);

  const noteCard = page.locator('article.card').filter({ hasText: title });
  await noteCard.getByRole('button', { name: 'Edit' }).click();
  const editor = page.getByRole('dialog');
  await expect(editor.getByRole('heading', { name: 'Edit note' })).toBeVisible();
  await editor.getByLabel('Title').fill(updatedTitle);
  await editor.getByLabel('Content').fill(updatedContent);
  await editor.getByRole('button', { name: 'Save changes' }).click();
  await expect(editor).toHaveCount(0);
  await expectCard(page, updatedTitle, true);

  const updatedCard = page.locator('article.card').filter({ hasText: updatedTitle });
  await updatedCard.getByRole('button', { name: 'Delete' }).click();
  await page.getByRole('alertdialog').getByRole('button', { name: 'Confirm delete' }).click();
  await expectCard(page, updatedTitle, false);
  const notes = await request.get(`${api}/houses/${data.houseId}/notes`, { headers: auth(data.member.token) });
  expect(notes.ok()).toBeTruthy();
  const remainingNotes = await notes.json() as Array<{ title: string }>;
  expect(remainingNotes.some(note => note.title === updatedTitle)).toBe(false);
  await expectNoOverflow(page);
});

test('admin role controls apply and monitor mutations stay unavailable', async ({ page, request }) => {
  const data = await fixture(request, true);
  await login(page, data.admin.email);
  await openHouse(page, data.houseId);
  const members = await openTab(page, 'Members', 'Members');
  const memberCard = members.locator('article.card').filter({ hasText: data.member.name });
  await expect(memberCard).toHaveCount(1);
  await memberCard.getByRole('combobox').selectOption('monitor');
  await memberCard.getByRole('button', { name: 'Change role' }).click();
  await expect.poll(async () => {
    const response = await request.get(`${api}/houses/${data.houseId}/members`, { headers: auth(data.admin.token) });
    if (!response.ok()) return '';
    const current = await response.json() as Array<{ user_id: string; role: string }>;
    return current.find(value => value.user_id === data.member.user.id)?.role ?? '';
  }).toBe('monitor');
  await expect.poll(() => memberCard.locator('p').evaluateAll((paragraphs) => (
    paragraphs.some(paragraph => paragraph.textContent?.includes('monitor') ?? false)
  ))).toBe(true);

  await logout(page);
  await login(page, data.member.email);
  await openHouse(page, data.houseId);
  const memberPanel = await openTab(page, 'Members', 'Members');
  const loadedMonitorCard = memberPanel.locator('article.card').filter({ hasText: data.member.name });
  await expect.poll(() => loadedMonitorCard.evaluate(card => (
    card.textContent?.includes('monitor') ?? false
  ))).toBe(true);
  await expect(memberPanel.getByRole('button', { name: 'Add member' })).toHaveCount(0);
  await expect(memberPanel.getByRole('button', { name: 'Change role' })).toHaveCount(0);
  await expect(memberPanel.getByRole('button', { name: 'Remove' })).toHaveCount(0);

  const expenses = await openTab(page, 'Expenses', 'Expenses');
  await expect(expenses.getByText('Loading expenses…')).toHaveCount(0);
  await expect(expenses.getByRole('button', { name: 'Add expense' })).toHaveCount(0);
  await expect(expenses.getByText('Your monitor role is view-only.')).toBeVisible();
  const notes = await openTab(page, 'Notes', 'Notes');
  await expect(notes.getByText('Loading notes…')).toHaveCount(0);
  await expect(notes.getByText('Your monitor role is view-only.')).toBeVisible();
  await expect(notes.getByRole('button', { name: 'Save note' })).toHaveCount(0);

  const blockedExpense = await request.post(`${api}/houses/${data.houseId}/expenses`, {
    headers: auth(data.member.token),
    data: { amount: 1, description: 'blocked', date: '2030-01-01', visibility: 'shared', visible_to: [], split: [] },
  });
  expect(blockedExpense.status()).toBe(403);
  const blockedNote = await request.post(`${api}/houses/${data.houseId}/notes`, {
    headers: auth(data.member.token),
    data: { title: 'blocked', content: 'blocked' },
  });
  expect(blockedNote.status()).toBe(403);
  const blockedRole = await request.put(`${api}/houses/${data.houseId}/members/${data.monitor!.user.id}`, {
    headers: auth(data.member.token),
    data: { role: 'member' },
  });
  expect(blockedRole.status()).toBe(403);
  await expectNoOverflow(page);
});

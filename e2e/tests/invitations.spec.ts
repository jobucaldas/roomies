import { expect, test, type APIRequestContext, type Page } from '@playwright/test';

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
  const message = await expect.poll(async () => {
    const search = await request.get(`${mailpitURL}/api/v1/search?query=to:${email}`);
    if (!search.ok()) return '';
    const data = await search.json() as { messages?: Array<{ ID: string }> };
    if (!data.messages?.[0]) return '';
    const body = await request.get(`${mailpitURL}/api/v1/message/${data.messages[0].ID}`);
    return body.ok() ? (await body.json() as { Text: string }).Text : '';
  }, { timeout: 15_000 });
  const token = (message.match(/accept-invitation\?token=([^\s]+)/)?.[1] ?? '').replace(/[>.)]+$/, '');
  expect(token).not.toBe('');
  return { admin: admin.token, house: house.id, invitation: invitation.id, token, email };
}

async function login(page: Page, email: string) {
  await page.goto('/');
  await page.getByPlaceholder('Email').fill(email);
  await page.getByPlaceholder('Password').fill('synthetic-password-123');
  await page.getByRole('button', { name: 'Login' }).click();
}

test('admin invite is delivered and intended user joins once', async ({ page, request }) => {
  const data = await fixture(request);
  const invitee = await register(request, data.email, 'Synthetic Invitee');
  await page.goto(`${process.env.ROOMIES_WEB_URL ?? 'http://localhost'}/accept-invitation?token=${data.token}`);
  await login(page, data.email);
  await expect(page.getByText('You joined the house.')).toBeVisible();
  await expect(page).toHaveURL(new RegExp(`/house/${data.house}`));
  await page.screenshot({ path: 'artifacts/invitation-joined.png', fullPage: true });
  const retry = await request.post(`${apiURL}/invitations/accept`, {
    headers: { Authorization: `Bearer ${invitee.token}` }, data: { token: data.token },
  });
  expect(retry.status()).toBe(404);
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

import { expect, test, type APIRequestContext } from '@playwright/test';
import { fillFlutterText, loginViaUiOrToken } from '../helpers/login';

/**
 * Regression: empty-DB first house — house-name field must keep DOM focus on a
 * phone viewport across post-auth dashboard settle and layout churn (the soft
 * keyboard used to open and immediately dismiss).
 */
test.use({ screenshot: 'on', trace: 'retain-on-failure' });

const api = process.env.ROOMIES_API_URL ?? 'http://localhost:8080/api';
const web = process.env.ROOMIES_WEB_URL ?? 'http://localhost';
const password = 'synthetic-password-123';

async function registerEmptyUser(request: APIRequestContext) {
  const suffix = `${Date.now()}-${Math.random().toString(16).slice(2)}`;
  const email = `house-focus-${suffix}@example.test`;
  const response = await request.post(`${api}/auth/register`, {
    data: { name: 'Focus Tester', email, password },
  });
  expect(response.ok()).toBeTruthy();
  const result = (await response.json()) as { token: string; user: { id: string } };
  return { email, token: result.token, user: result.user };
}

test.describe('create-house focus (narrow)', () => {
  test('house name field retains focus after dashboard settle', async ({
    page,
    request,
  }, testInfo) => {
    test.skip(
      testInfo.project.name !== 'narrow',
      'Phone viewport regression — run on the narrow project only',
    );

    const account = await registerEmptyUser(request);
    await loginViaUiOrToken(page, {
      web,
      email: account.email,
      password,
      token: account.token,
    });

    await expect(page).toHaveURL(/\/dashboard$/);
    // Empty membership: create form is the primary job.
    const houseName = page.getByRole('textbox', { name: /House name|Nome da casa/i });
    await expect(houseName).toBeVisible({ timeout: 30_000 });

    await houseName.evaluate((el) => {
      const input = el as HTMLElement;
      input.scrollIntoView({ block: 'center', inline: 'nearest' });
      input.focus();
    });
    await page.waitForTimeout(250);

    const focusedBefore = await page.evaluate(() => {
      const active = document.activeElement as HTMLElement | null;
      return {
        tag: active?.tagName ?? null,
        role: active?.getAttribute('role'),
        label: active?.getAttribute('aria-label') ?? active?.getAttribute('aria-labelledby'),
        isTextbox:
          active?.getAttribute('role') === 'textbox' ||
          active?.tagName === 'INPUT' ||
          active?.tagName === 'TEXTAREA' ||
          active?.hasAttribute('contenteditable') === true,
      };
    });
    expect(focusedBefore.isTextbox, `expected textbox focus, got ${JSON.stringify(focusedBefore)}`).toBeTruthy();

    // Simulate soft-keyboard viewport shrink (Android Chrome visualViewport).
    await page.setViewportSize({ width: 390, height: 520 });
    await page.waitForTimeout(400);

    const focusedAfterResize = await page.evaluate(() => {
      const active = document.activeElement as HTMLElement | null;
      const label = (active?.getAttribute('aria-label') ?? '').toLowerCase();
      return {
        stillTextbox:
          active?.getAttribute('role') === 'textbox' ||
          active?.tagName === 'INPUT' ||
          active?.tagName === 'TEXTAREA',
        label,
      };
    });
    expect(
      focusedAfterResize.stillTextbox,
      `focus lost after viewport shrink: ${JSON.stringify(focusedAfterResize)}`,
    ).toBeTruthy();

    await fillFlutterText(houseName, 'Casa Focus');
    await expect.poll(async () => houseName.inputValue()).toBe('Casa Focus');

    // Focus must still be on the house-name field after typing.
    const focusedAfterType = await page.evaluate(() => {
      const active = document.activeElement as HTMLElement | null;
      return (
        active?.getAttribute('role') === 'textbox' ||
        active?.tagName === 'INPUT' ||
        active?.tagName === 'TEXTAREA'
      );
    });
    expect(focusedAfterType).toBeTruthy();

    await page.screenshot({
      path: `artifacts/evidence/house-name-focus-${testInfo.project.name}.png`,
      fullPage: true,
    });
  });
});

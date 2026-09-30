import { expect, type Locator, type Page } from '@playwright/test';

/** Wait until Flutter web exposes password login or AuthKit CTA. */
export async function waitForLoginReady(page: Page) {
  const emailField = page.getByRole('textbox', { name: 'Email', disabled: false });
  const authkit = page.getByRole('button', { name: 'Sign in with AuthKit' });
  const loginButton = page.getByRole('button', { name: 'Login', exact: true });
  await Promise.race([
    emailField.waitFor({ state: 'visible', timeout: 90_000 }),
    authkit.waitFor({ state: 'visible', timeout: 90_000 }),
    loginButton.waitFor({ state: 'visible', timeout: 90_000 }),
  ]);
  return { emailField, authkit };
}

async function fillEnabledTextbox(page: Page, name: string, value: string) {
  const field = page.getByRole('textbox', { name, disabled: false });
  await fillFlutterText(field, value);
}

/** Flutter web attaches its input listener only after the semantics field is focused. */
export async function fillFlutterText(field: Locator, value: string) {
  await field.waitFor({ state: 'attached', timeout: 15000 });
  // Focus through the semantics node. A pointer click can hit a neighboring
  // control when Flutter's box is still settling, which dismisses the form.
  await field.evaluate((el) => {
    const input = el as HTMLElement;
    input.scrollIntoView({ block: 'center', inline: 'nearest' });
    input.focus();
  });
  await field.page().waitForTimeout(200);
  await field.evaluate((el, next) => {
    const input = el as HTMLInputElement | HTMLTextAreaElement;
    input.focus();
    input.value = next as string;
    input.dispatchEvent(new InputEvent('input', { bubbles: true, inputType: 'insertText', data: next as string }));
  }, value, { timeout: 15000 });
  await expect.poll(async () => field.inputValue()).toBe(value);
}

export async function loginViaUiOrToken(
  page: Page,
  options: {
    web: string;
    email: string;
    password: string;
    token?: string;
    /** When false, assume the page is already on / or another auth surface. */
    navigate?: boolean;
    /** AuthKit token bootstrap destination. */
    authkitFallback?: 'dashboard' | 'reload';
    expectDashboard?: boolean;
  },
) {
  const {
    web,
    email,
    password,
    token,
    navigate = true,
    authkitFallback = 'dashboard',
    expectDashboard = true,
  } = options;
  if (navigate) {
    await page.goto(web);
  }
  const { emailField } = await waitForLoginReady(page);
  if (token) {
    const sessionUrl = new URL(web);
    await page.context().addCookies([
      {
        name: 'roomies_session',
        value: token,
        url: sessionUrl.origin,
        httpOnly: true,
        sameSite: 'Lax',
        secure: sessionUrl.protocol === 'https:',
      },
      {
        name: 'roomies_session_hint',
        value: '1',
        url: sessionUrl.origin,
        httpOnly: false,
        sameSite: 'Lax',
        secure: sessionUrl.protocol === 'https:',
      },
    ]);
    if (authkitFallback === 'reload') {
      await page.reload();
    } else {
      await page.goto(`${web}/dashboard`);
    }
  } else if (await emailField.isVisible()) {
    await fillEnabledTextbox(page, 'Email', email);
    await fillEnabledTextbox(page, 'Password', password);
    await page.getByRole('button', { name: 'Login', exact: true }).click();
  } else {
    expect(token, 'AuthKit UI requires API token for e2e login').toBeTruthy();
  }
  if (expectDashboard) {
    await expect(page).toHaveURL(/\/dashboard$/);
  }
}

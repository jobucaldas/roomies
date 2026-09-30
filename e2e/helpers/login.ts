import { expect, type Page } from '@playwright/test';

/** Wait until Flutter web exposes password login or AuthKit CTA. */
export async function waitForLoginReady(page: Page) {
  await page.getByRole('heading', { name: 'Login' }).waitFor({
    state: 'visible',
    timeout: 90_000,
  });
  // Flutter also renders a disabled semantics input with the same name.
  // The enabled textbox is the field the user can type into.
  const emailField = page.getByRole('textbox', { name: 'Email', disabled: false });
  const authkit = page.getByRole('button', { name: 'Sign in with AuthKit' });
  await Promise.race([
    emailField.waitFor({ state: 'visible', timeout: 90_000 }),
    authkit.waitFor({ state: 'visible', timeout: 90_000 }),
  ]);
  return { emailField, authkit };
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
  if (await emailField.isVisible()) {
    await emailField.fill(email);
    await page.getByRole('textbox', { name: 'Password', disabled: false }).fill(password);
    await page.getByRole('button', { name: 'Login' }).click();
  } else {
    expect(token, 'AuthKit UI requires API token for e2e login').toBeTruthy();
    const sessionUrl = new URL(web);
    await page.context().addCookies([
      {
        name: 'roomies_session',
        value: token as string,
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
  }
  if (expectDashboard) {
    await expect(page).toHaveURL(/\/dashboard$/);
  }
}

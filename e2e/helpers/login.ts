import { expect, type Locator, type Page } from '@playwright/test';

/** Wait until Flutter web exposes password (collapsed or open) or AuthKit CTA. */
export async function waitForLoginReady(page: Page) {
  const emailField = page.getByRole('textbox', { name: 'Email', disabled: false });
  const usePassword = page.getByRole('button', { name: 'Use email and password' });
  // AuthKit-on primary CTA and password-form submit both use "Sign in".
  const signIn = page.getByRole('button', { name: 'Sign in', exact: true });
  await Promise.race([
    emailField.waitFor({ state: 'visible', timeout: 90_000 }),
    usePassword.waitFor({ state: 'visible', timeout: 90_000 }),
    signIn.waitFor({ state: 'visible', timeout: 90_000 }),
  ]);
  return { emailField, usePassword, signIn };
}

/** Reveal the CI/local password fields when AuthKit is unset. */
export async function expandPasswordLogin(page: Page) {
  const emailField = page.getByRole('textbox', { name: 'Email', disabled: false });
  if (await emailField.isVisible().catch(() => false)) {
    return emailField;
  }
  const usePassword = page.getByRole('button', { name: 'Use email and password' });
  await usePassword.waitFor({ state: 'visible', timeout: 15_000 });
  await usePassword.click();
  await emailField.waitFor({ state: 'visible', timeout: 15_000 });
  return emailField;
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
  const { emailField, usePassword, signIn } = await waitForLoginReady(page);
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
  } else if (
    (await usePassword.isVisible().catch(() => false)) ||
    (await emailField.isVisible().catch(() => false))
  ) {
    // AuthKit unset: password path starts collapsed behind "Use email and password".
    await expandPasswordLogin(page);
    await fillEnabledTextbox(page, 'Email', email);
    await fillEnabledTextbox(page, 'Password', password);
    await page.getByRole('button', { name: 'Sign in', exact: true }).click();
  } else if (await signIn.isVisible().catch(() => false)) {
    expect(token, 'AuthKit UI requires API token for e2e login').toBeTruthy();
  } else {
    expect(token, 'AuthKit UI requires API token for e2e login').toBeTruthy();
  }
  if (expectDashboard) {
    await expect(page).toHaveURL(/\/dashboard$/);
  }
}


/** Open a house from Dashboard/shell after login (fresh sessions land on Dashboard). */
export async function openHouseViaUi(
  page: Page,
  options: {
    web: string;
    houseName: string;
    houseId?: string;
  },
) {
  const { web, houseName, houseId } = options;
  if (!/\/dashboard\/?$/.test(new URL(page.url()).pathname)) {
    await page.goto(`${web}/dashboard`);
  }
  await expect(page).toHaveURL(/\/dashboard\/?$/);

  // Wait for Dashboard chrome so house list / shell nav are mounted.
  await expect(page.getByRole('heading', { name: 'Dashboard', exact: true })).toBeVisible({
    timeout: 30_000,
  });

  // Shell nav (sidebar / drawer) exposes each house as a button.
  const shellHouse = page.getByRole('button', { name: houseName, exact: true });
  if ((await shellHouse.count()) === 0) {
    const menu = page.getByRole('button', { name: 'Menu' });
    if (await menu.isVisible().catch(() => false)) {
      await menu.click();
    }
  }
  if ((await shellHouse.count()) > 0) {
    await shellHouse.first().click();
  } else {
    // Dashboard house row semantics: "Open <name>".
    const openRow = page.getByRole('button', { name: `Open ${houseName}`, exact: true });
    await openRow.waitFor({ state: 'visible', timeout: 30_000 });
    await openRow.click();
  }

  if (houseId) {
    await expect(page).toHaveURL(new RegExp(`/house/${houseId}$`));
  } else {
    await expect(page).toHaveURL(/\/house\/[^/]+$/);
  }
  await expect(page.getByRole('heading', { name: houseName, exact: true })).toBeVisible({
    timeout: 30_000,
  });
}

/** Sign out from AppShell (sidebar on wide, drawer on narrow). */
export async function logoutViaUi(page: Page, web: string) {
  await page.goto(`${web}/dashboard`);
  const logOut = page.getByRole('button', { name: 'Log out' });
  if (!(await logOut.isVisible().catch(() => false))) {
    await page.getByRole('button', { name: 'Menu' }).click();
    await logOut.waitFor({ state: 'visible', timeout: 15_000 });
  }
  await logOut.click();
  await waitForLoginReady(page);
}

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


/** Click a Flutter semantics control that Playwright may not consider "visible". */
async function clickFlutterRole(
  page: Page,
  role: string,
  name: string,
  options?: { exact?: boolean; timeout?: number },
) {
  const exact = options?.exact ?? true;
  const timeout = options?.timeout ?? 30_000;
  const locator = page.getByRole(role as 'button', { name, exact });
  await locator.first().waitFor({ state: 'attached', timeout });
  // Flutter web often parks semantics nodes off the hit-test path; prefer DOM activate.
  const handle = await locator.first().elementHandle({ timeout });
  if (!handle) {
    throw new Error(`No element for role=${role} name=${name}`);
  }
  await handle.evaluate((el) => {
    const node = el as HTMLElement;
    node.scrollIntoView({ block: 'center', inline: 'nearest' });
    node.focus();
    node.click();
  });
  await handle.dispose();
}

/** Open a seeded house after login (Dashboard-first UX). Prefer direct route when id known. */
export async function openHouseViaUi(
  page: Page,
  options: {
    web: string;
    houseName: string;
    houseId?: string;
  },
) {
  const { web, houseName, houseId } = options;

  if (houseId) {
    await page.goto(`${web}/house/${houseId}`);
    await expect(page).toHaveURL(new RegExp(`/house/${houseId}$`));
    await expect(page.getByRole('heading', { name: houseName, exact: true })).toBeVisible({
      timeout: 30_000,
    });
    return;
  }

  if (!/\/dashboard\/?$/.test(new URL(page.url()).pathname)) {
    await page.goto(`${web}/dashboard`);
  }
  await expect(page).toHaveURL(/\/dashboard\/?$/);
  await expect(page.getByRole('heading', { name: 'Dashboard', exact: true })).toBeVisible({
    timeout: 30_000,
  });

  const shellHouse = page.getByRole('button', { name: houseName, exact: true });
  try {
    await shellHouse.first().waitFor({ state: 'attached', timeout: 5_000 });
  } catch {
    const menu = page.getByRole('button', { name: 'Menu' });
    if ((await menu.count()) > 0) {
      await clickFlutterRole(page, 'button', 'Menu', { timeout: 10_000 });
    }
  }

  if ((await shellHouse.count()) > 0) {
    await clickFlutterRole(page, 'button', houseName, { timeout: 30_000 });
  } else {
    await clickFlutterRole(page, 'button', `Open ${houseName}`, { timeout: 30_000 });
  }

  await expect(page).toHaveURL(/\/house\/[^/]+$/);
  await expect(page.getByRole('heading', { name: houseName, exact: true })).toBeVisible({
    timeout: 30_000,
  });
}

/** Sign out from AppShell (sidebar on wide, drawer on narrow). */
export async function logoutViaUi(page: Page, web: string) {
  await page.goto(`${web}/dashboard`);
  await expect(page.getByRole('heading', { name: 'Dashboard', exact: true })).toBeVisible({
    timeout: 30_000,
  });
  const logOut = page.getByRole('button', { name: 'Log out' });
  if ((await logOut.count()) === 0) {
    await clickFlutterRole(page, 'button', 'Menu', { timeout: 15_000 });
  }
  await clickFlutterRole(page, 'button', 'Log out', { timeout: 15_000 });
  await waitForLoginReady(page);
}
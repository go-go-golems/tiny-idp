import { expect, test } from '@playwright/test';

// The operator supplies these only to the test process. Do not commit test
// account credentials to this ticket or repository.
const baseURL = 'http://127.0.0.1:8090';
const firstLogin = process.env.TINYIDP_TEST_LOGIN;
const password = process.env.TINYIDP_TEST_PASSWORD;
const secondLogin = process.env.TINYIDP_TEST_SECOND_LOGIN;
const chromiumExecutable = process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE;

if (!firstLogin || !password || !secondLogin) {
  throw new Error('set TINYIDP_TEST_LOGIN, TINYIDP_TEST_PASSWORD, and TINYIDP_TEST_SECOND_LOGIN');
}

if (chromiumExecutable) {
  test.use({ launchOptions: { executablePath: chromiumExecutable } });
}

async function signIn(page, login, password) {
  await page.getByLabel('LOGIN').fill(login);
  await page.getByLabel('PASSWORD').fill(password);
  await page.getByRole('button', { name: 'Approve' }).click();
  await expect(page).toHaveURL(baseURL + '/');
  await expect(page.getByText('SIGNED IN')).toBeVisible();
}

test('Message Desk can switch a browser from one remembered account to another', async ({ page }) => {
  await page.goto(baseURL, { waitUntil: 'networkidle' });
  await page.getByRole('link', { name: 'Sign in' }).click();
  await expect(page.getByRole('heading', { name: /Sign in/ })).toBeVisible();
  await signIn(page, firstLogin, password);
  const firstSession = await page.request.get(baseURL + '/api/session').then((response) => response.json());

  await page.getByRole('link', { name: 'Change account' }).click();
  await expect(page.getByRole('heading', { name: 'Choose an account' })).toBeVisible();
  await expect(page.locator('input[type="radio"][name="account"]')).toHaveCount(1);
  await page.getByRole('button', { name: 'Use another account' }).click();
  await expect(page.getByRole('heading', { name: /Sign in/ })).toBeVisible();
  await signIn(page, secondLogin, password);

  const secondSession = await page.request.get(baseURL + '/api/session').then((response) => response.json());
  expect(secondSession.authenticated).toBe(true);
  expect(secondSession.subject).not.toBe(firstSession.subject);

  await page.getByRole('link', { name: 'Change account' }).click();
  await expect(page.getByRole('heading', { name: 'Choose an account' })).toBeVisible();
  await expect(page.locator('input[type="radio"][name="account"]')).toHaveCount(2);
});

test('local Message Desk logout preserves remembered accounts for the next sign in', async ({ page }) => {
  await page.goto(baseURL, { waitUntil: 'networkidle' });
  await page.getByRole('link', { name: 'Sign in' }).click();
  await expect(page.getByRole('heading', { name: /Sign in/ })).toBeVisible();
  await signIn(page, firstLogin, password);

  await page.getByRole('button', { name: 'Log out of Message Desk' }).click();
  await expect(page.getByRole('link', { name: 'Sign in' })).toBeVisible();

  await page.getByRole('link', { name: 'Sign in' }).click();
  await expect(page.getByRole('heading', { name: 'Choose an account' })).toBeVisible();
  await expect(page.locator('input[type="radio"][name="account"]')).toHaveCount(1);
});

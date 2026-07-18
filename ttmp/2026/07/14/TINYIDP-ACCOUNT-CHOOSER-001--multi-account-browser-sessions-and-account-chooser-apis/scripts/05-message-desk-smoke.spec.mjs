import { expect, test } from '@playwright/test';

// The ticket runbook starts the isolated demo on this fixed loopback origin.
// Keeping this explicit avoids a hidden environment-variable configuration
// channel in an otherwise reproducible browser smoke check.
const baseURL = 'http://127.0.0.1:8090';

test('Message Desk renders its self-service account entry point', async ({ page }) => {
  await page.goto(baseURL, { waitUntil: 'networkidle' });
  await expect(page.getByRole('heading', { name: 'Message Desk' })).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Open an account' })).toBeVisible();
  await expect(page.getByRole('link', { name: 'Sign in' })).toBeVisible();
});

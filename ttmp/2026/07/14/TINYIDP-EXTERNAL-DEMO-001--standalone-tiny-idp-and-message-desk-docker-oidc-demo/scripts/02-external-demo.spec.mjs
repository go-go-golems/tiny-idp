import { expect, test } from "@playwright/test";

const messageDeskURL = process.env.MESSAGE_DESK_URL || "http://localhost:8080";
const idpURL = process.env.TINYIDP_ISSUER_URL || "http://localhost:8081";
const firstAccount = {
  login: process.env.TINYIDP_E2E_FIRST_LOGIN || "amelie",
  name: process.env.TINYIDP_E2E_FIRST_NAME || "Amelie"
};
const secondAccount = {
  login: process.env.TINYIDP_E2E_SECOND_LOGIN || "wesen",
  name: process.env.TINYIDP_E2E_SECOND_NAME || "Wesen"
};
const password = process.env.TINYIDP_E2E_PASSWORD || "dev-only-not-a-secret-12345";

async function expectReady(request, url) {
  const response = await request.get(url, { timeout: 10_000 });
  expect(response.status(), `${url} must be ready before browser assertions`).toBeLessThan(300);
}

async function beginSignIn(page) {
  await page.getByRole("link", { name: "Sign in", exact: true }).click();
  await expect(page).toHaveURL(new RegExp(`^${escapeRegExp(idpURL)}/authorize`));
}

async function approvePasswordLogin(page, account) {
  await expect(page.locator("#tinyidp-login")).toBeVisible();
  await page.locator("#tinyidp-login").fill(account.login);
  await page.locator("#tinyidp-password").fill(password);
  await page.getByRole("button", { name: /Approve|Continue/ }).click();
  await expect(page).toHaveURL(new RegExp(`^${escapeRegExp(messageDeskURL)}/`));
  await expect(page.locator("header .status").getByText(account.name, { exact: true })).toBeVisible();
}

async function useAnotherAccount(page) {
  await expect(page.getByRole("heading", { name: "Choose an account" })).toBeVisible();
  await page.getByRole("button", { name: "Use another account" }).click();
  await expect(page.locator("#tinyidp-login")).toBeVisible();
}

function escapeRegExp(value) {
  return value.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}

test.describe("standalone tiny-idp and external Message Desk", () => {
  test.beforeAll(async ({ request }) => {
    await expectReady(request, `${messageDeskURL}/readyz`);
    await expectReady(request, `${idpURL}/readyz`);
  });

  test("executes two-origin login, consent, message, chooser, and logout contracts", async ({ page }) => {
    await page.goto(messageDeskURL);
    await expect(page.getByRole("heading", { name: "Use a desk account" })).toBeVisible();
    await expect(page.getByRole("button", { name: "Create account" })).toHaveCount(0);
    await expect.poll(async () => (await page.request.get(`${messageDeskURL}/api/registration`)).status()).toBe(404);

    // A callback that was never initiated by this browser cannot establish an RP session.
    const invalidCallback = await page.request.get(`${messageDeskURL}/auth/callback?state=unrecognized-state&code=unrecognized-code`);
    expect(invalidCallback.status()).toBe(502);

    await beginSignIn(page);
    await expect(page.getByText("REQUESTED ACCESS", { exact: true })).toBeVisible();
    await expect(page.getByText("openid", { exact: true })).toBeVisible();
    await expect(page.getByText("profile", { exact: true })).toBeVisible();
    await approvePasswordLogin(page, firstAccount);

    const missingCSRF = await page.evaluate(async () => {
      const response = await fetch("/api/messages", {
        method: "POST",
        headers: { "content-type": "application/json" },
        body: JSON.stringify({ body: "must not be accepted" })
      });
      return response.status;
    });
    expect(missingCSRF).toBe(403);

    const marker = `Playwright two-origin verification ${Date.now()}`;
    await page.getByRole("textbox", { name: "MESSAGE" }).fill(marker);
    await page.getByRole("button", { name: "Place note" }).click();
    await expect(page.getByText(marker, { exact: true })).toBeVisible();

    // Local logout removes only the relying-party session; provider chooser state remains.
    await page.getByRole("button", { name: "Log out of Message Desk" }).click();
    await expect(page.getByRole("link", { name: "Sign in", exact: true })).toBeVisible();
    await beginSignIn(page);
    await expect(page.getByRole("heading", { name: "Choose an account" })).toBeVisible();

    // Account selection is provider-owned. The RP asks for it but cannot name accounts itself.
    await useAnotherAccount(page);
    await approvePasswordLogin(page, secondAccount);

    // The second message proves the relying party derives message attribution
    // from the newly verified OIDC session, rather than retaining the first
    // account's display name across a provider-owned account switch.
    const secondMarker = `Playwright switched-account verification ${Date.now()}`;
    await page.getByRole("textbox", { name: "MESSAGE" }).fill(secondMarker);
    await page.getByRole("button", { name: "Place note" }).click();
    const secondMessage = page.getByRole("article").filter({ hasText: secondMarker });
    await expect(secondMessage.getByText(secondAccount.name, { exact: true })).toBeVisible();
    await expect(secondMessage.getByText(secondMarker, { exact: true })).toBeVisible();

    // Global logout clears both the RP session and provider browser session.
    await page.getByRole("button", { name: "Log out everywhere" }).click();
    await expect(page.getByRole("link", { name: "Sign in", exact: true })).toBeVisible();
    await beginSignIn(page);
    await expect(page.locator("#tinyidp-login")).toBeVisible();
    await expect(page.getByRole("heading", { name: "Choose an account" })).toHaveCount(0);
  });
});

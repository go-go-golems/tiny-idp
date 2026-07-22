import { expect, Page, test } from "@playwright/test";

const messageOrigin = "https://message.localhost:8443";
const idpOrigin = "https://idp.localhost:8443";
const outboxOrigin = "http://127.0.0.1:8025";
const outboxAuthorization = `Basic ${Buffer.from("operator:local-outbox-password-2026!").toString("base64")}`;

async function expectMessageDeskTheme(page: Page): Promise<void> {
  await expect(page.locator('link[rel="stylesheet"]')).toHaveAttribute(
    "href",
    "/static/themes/message-desk.css"
  );
}

async function beginMessageSignup(page: Page): Promise<void> {
  await page.goto(`${messageOrigin}/auth/register?return_to=/`);
  await expect(page).toHaveURL(new RegExp(`^${idpOrigin.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")}/authorize`));
  await expect(page.getByRole("heading", { name: "Create an account" })).toBeVisible();
  await expectMessageDeskTheme(page);
}

async function latestEmailCode(page: Page, recipient: string): Promise<string> {
  const query = encodeURIComponent(`to:"${recipient}"`);
  await expect
    .poll(async () => {
      const response = await page.request.get(`${outboxOrigin}/view/latest.txt?query=${query}`, {
        headers: { Authorization: outboxAuthorization }
      });
      if (!response.ok()) return "";
      return (await response.text()).match(/verification code is:\s*([A-Z2-7]{8})/)?.[1] ?? "";
    })
    .not.toBe("");
  const response = await page.request.get(`${outboxOrigin}/view/latest.txt?query=${query}`, {
    headers: { Authorization: outboxAuthorization }
  });
  const code = (await response.text()).match(/verification code is:\s*([A-Z2-7]{8})/)?.[1];
  if (!code) throw new Error(`Mailpit did not expose a verification code for ${recipient}`);
  return code;
}

async function submitIdentity(page: Page, displayName: string, email: string): Promise<void> {
  await page.getByLabel("Display name").fill(displayName);
  await page.getByLabel("Email").fill(email);
  await page.getByRole("button", { name: "Create account" }).click();
}

async function loginToMessageDesk(page: Page): Promise<void> {
  await page.goto(`${messageOrigin}/auth/login?return_to=/`);
  await page.getByLabel("Login").fill("admin@example.test");
  await page.getByLabel("Password").fill("local-admin-password-2026!");
  await page.getByRole("button", { name: /continue|sign in|approve/i }).first().click();
  if (page.url().startsWith(idpOrigin)) {
    await page.getByRole("button", { name: /approve|continue/i }).first().click();
  }
  await expect(page).toHaveURL(new RegExp(`^${messageOrigin.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")}/`));
  await expect(page.getByText("SIGNED IN")).toBeVisible();
}

test.beforeEach(async ({ page }) => {
  const pageErrors: string[] = [];
  page.on("pageerror", error => pageErrors.push(error.message));
  (page as Page & { productPageErrors?: string[] }).productPageErrors = pageErrors;
});

test.afterEach(async ({ page }) => {
  expect((page as Page & { productPageErrors?: string[] }).productPageErrors).toEqual([]);
});

test("Message Desk presents signup and login as separate vertically stacked actions", async ({ page }) => {
  await page.goto(messageOrigin);
  const signup = page.getByRole("link", { name: "Create an account with Tiny-IDP" });
  const login = page.getByRole("link", { name: "I already have an account" });
  await expect(signup).toBeVisible();
  await expect(login).toBeVisible();
  const signupBox = await signup.boundingBox();
  const loginBox = await login.boundingBox();
  expect(signupBox).not.toBeNull();
  expect(loginBox).not.toBeNull();
  expect(loginBox!.y).toBeGreaterThan(signupBox!.y + signupBox!.height - 1);
});

test("malformed signup email stays on the themed form with native validation", async ({ page }) => {
  await beginMessageSignup(page);
  await page.getByLabel("Display name").fill("Malformed Email");
  await page.getByLabel("Email").fill("not-an-email");
  await page.getByRole("button", { name: "Create account" }).click();
  await expect(page.getByLabel("Email")).toBeFocused();
  expect(await page.getByLabel("Email").evaluate((input: HTMLInputElement) => input.validity.typeMismatch)).toBe(true);
  await expect(page.getByRole("heading", { name: "Create an account" })).toBeVisible();
  await expectMessageDeskTheme(page);
});

test("remembered TinyIDP session can submit the first add-account signup step", async ({ page }) => {
  await loginToMessageDesk(page);
  await page.getByRole("button", { name: "Log out of Message Desk" }).click();
  await expect(page.getByText("GUEST MODE")).toBeVisible();
  await beginMessageSignup(page);
  const email = `playwright-add-account-${Date.now()}@example.test`;
  await submitIdentity(page, "Playwright Add Account", email);
  await expect(page.getByLabel("Email verification code")).toBeVisible();
  await expectMessageDeskTheme(page);
});

test("duplicate email produces a themed actionable signup error", async ({ page }) => {
  const email = "admin@example.test";
  await beginMessageSignup(page);
  await submitIdentity(page, "Duplicate Administrator", email);
  await expect(page.getByLabel("Email verification code")).toBeVisible();
  await page.getByLabel("Email verification code").fill(await latestEmailCode(page, email));
  await page.getByRole("button", { name: /create account|continue/i }).click();
  await expect(page.getByRole("heading", { name: "Create an account" })).toBeVisible();
  await page.getByLabel("Password", { exact: true }).fill("duplicate account password 2026!");
  await page.getByLabel("Confirm password").fill("duplicate account password 2026!");
  await page.getByRole("button", { name: "Create account" }).click();

  await expect(page.getByText("An account already uses this email address.")).toBeVisible();
  await expect(page.getByRole("link", { name: "Return to application" })).toBeVisible();
  await expectMessageDeskTheme(page);
  expect((await page.locator("body").innerText()).toLowerCase()).not.toContain("registration request was not accepted");
});

test("duplicate display name is rejected on the themed identity form before email verification", async ({ page }) => {
  const suffix = Date.now();
  const displayName = `Playwright Unique Name ${suffix}`;
  const firstEmail = `playwright-name-first-${suffix}@example.test`;
  await beginMessageSignup(page);
  await submitIdentity(page, displayName, firstEmail);
  await expect(page.getByLabel("Email verification code")).toBeVisible();
  await page.getByLabel("Email verification code").fill(await latestEmailCode(page, firstEmail));
  await page.getByRole("button", { name: /create account|continue/i }).click();
  await expect(page.getByLabel("Password", { exact: true })).toBeVisible();
  await page.getByLabel("Password", { exact: true }).fill("playwright unique display name password 2026!");
  await page.getByLabel("Confirm password").fill("playwright unique display name password 2026!");
  await page.getByRole("button", { name: "Create account" }).click();
  await expect(page.getByRole("heading", { name: "Approve access" })).toBeVisible();

  await beginMessageSignup(page);
  await submitIdentity(page, displayName, `playwright-name-second-${suffix}@example.test`);
  await expect(page.getByRole("heading", { name: "Create an account" })).toBeVisible();
  await expect(page.getByText("That display name is already in use. Choose another.")).toBeVisible();
  await expect(page.getByLabel("Email verification code")).toHaveCount(0);
  await expectMessageDeskTheme(page);
});

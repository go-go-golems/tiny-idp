import { expect, Page, test } from "@playwright/test";

const messageOrigin = "https://message.localhost:8443";
const idpOrigin = "https://idp.localhost:8443";
const gojaOrigin = "https://goja.localhost:8443";
const outboxOrigin = "http://127.0.0.1:8025";
const outboxAuthorization = `Basic ${Buffer.from("operator:local-outbox-password-2026!").toString("base64")}`;

async function expectMessageDeskTheme(page: Page): Promise<void> {
  await expect(page.locator('link[rel="stylesheet"]')).toHaveAttribute(
    "href",
    "/static/themes/message-desk.css"
  );
}

async function expectGojaAuthTheme(page: Page): Promise<void> {
  await expect(page.locator('link[rel="stylesheet"]')).toHaveAttribute(
    "href",
    "/static/themes/goja-auth-lab.css"
  );
}

async function beginMessageSignup(page: Page): Promise<void> {
  await page.goto(`${messageOrigin}/auth/register?return_to=/`);
  await expect(page).toHaveURL(new RegExp(`^${idpOrigin.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")}/authorize`));
  await expect(page.getByRole("heading", { name: "Create an account" })).toBeVisible();
  await expectMessageDeskTheme(page);
}

async function beginGojaSignup(page: Page): Promise<void> {
  await page.goto(`${gojaOrigin}/auth/register?return_to=/`);
  await expect(page).toHaveURL(new RegExp(`^${idpOrigin.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")}/authorize`));
  await expect(page.getByRole("heading", { name: "Create an account" })).toBeVisible();
  await expectGojaAuthTheme(page);
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

async function loginToMessageDesk(page: Page, login = "admin@example.test", password = "local-admin-password-2026!"): Promise<void> {
  await page.goto(`${messageOrigin}/auth/login?return_to=/`);
  await page.getByLabel("Login").fill(login);
  await page.getByLabel("Password").fill(password);
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

test("signup identity form enforces required and bounded display names before submission", async ({ page }) => {
  await beginMessageSignup(page);
  const displayName = page.getByLabel("Display name");
  await expect(displayName).toHaveAttribute("maxlength", "120");
  await page.getByLabel("Email").fill("display-name-validation@example.test");
  await page.getByRole("button", { name: "Create account" }).click();
  await expect(displayName).toBeFocused();
  expect(await displayName.evaluate((input: HTMLInputElement) => input.validity.valueMissing)).toBe(true);
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

test("logging out everywhere clears Message Desk and TinyIDP browser sessions", async ({ page, context }) => {
  await loginToMessageDesk(page);
  await page.getByRole("button", { name: "Log out everywhere" }).click();
  await expect(page).toHaveURL(new RegExp(`^${messageOrigin.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")}/`));
  await expect(page.getByText("GUEST MODE")).toBeVisible();
  const idpCookies = await context.cookies(idpOrigin);
  expect(idpCookies.some(cookie => cookie.name === "tinyidp_session" && cookie.value !== "")).toBe(false);
});

test("account chooser remembers two password logins and supports switching accounts", async ({ page }) => {
  await loginToMessageDesk(page);
  await page.getByRole("link", { name: "Change account" }).click();
  await expect(page.getByRole("heading", { name: "Choose an account" })).toBeVisible();
  await expect(page.getByLabel("Local Administrator")).toBeVisible();
  await page.getByRole("button", { name: "Use another account" }).click();

  await page.getByLabel("Login").fill("invitee@example.test");
  await page.getByLabel("Password").fill("local-invitee-password-2026!");
  await page.getByRole("button", { name: /continue|sign in|approve/i }).first().click();
  if (page.url().startsWith(idpOrigin)) {
    await page.getByRole("button", { name: /approve|continue/i }).first().click();
  }
  await expect(page).toHaveURL(new RegExp(`^${messageOrigin.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")}/`));
  await expect(page.getByText("Local Invitee", { exact: true })).toBeVisible();

  await page.getByRole("link", { name: "Change account" }).click();
  await expect(page.getByRole("heading", { name: "Choose an account" })).toBeVisible();
  await expect(page.getByLabel("Local Administrator")).toBeVisible();
  await expect(page.getByLabel("Local Invitee")).toBeVisible();
  await page.getByLabel("Local Administrator").check();
  await page.getByRole("button", { name: "Continue" }).click();
  await expect(page).toHaveURL(new RegExp(`^${messageOrigin.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")}/`));
  await expect(page.getByText("Local Administrator", { exact: true })).toBeVisible();

  await page.getByRole("link", { name: "Change account" }).click();
  await page.getByLabel("Local Invitee").check();
  await page.getByRole("button", { name: "Remove account" }).click();
  await expect(page.getByLabel("Local Invitee")).toHaveCount(0);
  await expect(page.getByLabel("Local Administrator")).toBeVisible();
});

test("short password is rejected by native validation before the password workflow posts", async ({ page }) => {
  const email = `playwright-short-password-${Date.now()}@example.test`;
  await beginMessageSignup(page);
  await submitIdentity(page, "Playwright Short Password", email);
  await page.getByLabel("Email verification code").fill(await latestEmailCode(page, email));
  await page.getByRole("button", { name: /create account|continue/i }).click();
  const password = page.getByLabel("Password", { exact: true });
  await password.fill("too-short-2026");
  await page.getByLabel("Confirm password").fill("too-short-2026");
  await page.getByRole("button", { name: "Create account" }).click();
  await expect(password).toBeFocused();
  expect(await password.evaluate((input: HTMLInputElement) => input.validity.tooShort)).toBe(true);
  await expectMessageDeskTheme(page);
});

test("wrong email verification code keeps a themed retry form with resend", async ({ page }) => {
  const email = `playwright-wrong-code-${Date.now()}@example.test`;
  await beginMessageSignup(page);
  await submitIdentity(page, "Playwright Wrong Code", email);
  const code = page.getByLabel("Email verification code");
  await expect(code).toBeVisible();
  await code.fill("AAAAAAAA");
  await page.getByRole("button", { name: "Create account" }).click();

  await expect(code).toBeVisible();
  await expect(code).toHaveValue("");
  await expect(page.getByText("This value could not be accepted.")).toBeVisible();
  await expect(page.getByRole("button", { name: "Send another code" })).toBeVisible();
  await expectMessageDeskTheme(page);
});

test("email-code resend keeps a blank themed retry workflow", async ({ page }) => {
  const email = `playwright-resend-code-${Date.now()}@example.test`;
  await beginMessageSignup(page);
  await submitIdentity(page, "Playwright Resend Code", email);
  await page.getByRole("button", { name: "Send another code" }).click();
  const code = page.getByLabel("Email verification code");
  await expect(code).toBeVisible();
  await expect(code).toHaveValue("");
  await expect(page.getByRole("button", { name: "Send another code" })).toBeVisible();
  await expectMessageDeskTheme(page);
});

test("email-code attempt exhaustion remains themed and explains the recovery", async ({ page }) => {
  const email = `playwright-code-exhaustion-${Date.now()}@example.test`;
  await beginMessageSignup(page);
  await submitIdentity(page, "Playwright Code Exhaustion", email);
  const code = page.getByLabel("Email verification code");
  for (let attempt = 0; attempt < 5; attempt++) {
    await code.fill("AAAAAAAA");
    await page.getByRole("button", { name: "Create account" }).click();
    await expect(code).toHaveValue("");
  }
  await expect(page.getByText("Too many incorrect verification codes were entered. Restart registration to receive a new code.")).toBeVisible();
  await expectMessageDeskTheme(page);
});

test("email-code resend limit remains themed and preserves the verification form", async ({ page }) => {
  const email = `playwright-resend-limit-${Date.now()}@example.test`;
  await beginMessageSignup(page);
  await submitIdentity(page, "Playwright Resend Limit", email);
  const resend = page.getByRole("button", { name: "Send another code" });
  await resend.click();
  await resend.click();
  await resend.click();
  await expect(page.getByText("No more verification codes can be sent for this registration. Enter the most recent code or restart registration.")).toBeVisible();
  await expect(page.getByLabel("Email verification code")).toHaveValue("");
  await expectMessageDeskTheme(page);
});

for (const [name, login, password] of [
  ["unknown login", "not-a-real-account@example.test", "not-the-right-password-2026!"],
  ["wrong password", "admin@example.test", "not-the-right-password-2026!"],
]) {
  test(`${name} retains the login name and presents a themed generic credential error`, async ({ page }) => {
    await page.goto(`${messageOrigin}/auth/login?return_to=/`);
    await page.getByLabel("Login").fill(login);
    await page.getByLabel("Password").fill(password);
    await page.getByRole("button", { name: /continue|sign in|approve/i }).first().click();

    await expect(page.getByText("Invalid login or password.")).toBeVisible();
    await expect(page.getByLabel("Login")).toHaveValue(login);
    await expect(page.getByLabel("Password")).toHaveValue("");
    await expectMessageDeskTheme(page);
  });
}

test("Goja Auth invalid credentials retain login and use the Goja-specific theme", async ({ page }) => {
  await page.goto(`${gojaOrigin}/auth/login?return_to=/`);
  await page.getByLabel("Login").fill("not-a-real-goja-account@example.test");
  await page.getByLabel("Password").fill("not-the-right-password-2026!");
  await page.getByRole("button", { name: /continue|sign in|approve/i }).first().click();

  await expect(page.getByText("Invalid login or password.")).toBeVisible();
  await expect(page.getByLabel("Login")).toHaveValue("not-a-real-goja-account@example.test");
  await expect(page.getByLabel("Password")).toHaveValue("");
  await expectGojaAuthTheme(page);
});

test("Goja signup rejects an unknown invitation with a themed field error", async ({ page }) => {
  await beginGojaSignup(page);
  await page.getByLabel("Display name").fill("Goja Unknown Invitation");
  await page.getByLabel("Email").fill(`playwright-goja-invite-${Date.now()}@example.test`);
  await page.getByLabel("Invite code").fill("not-a-real-invitation-code");
  await page.getByRole("button", { name: "Create account" }).click();

  await expect(page.getByLabel("Invite code")).toHaveValue("not-a-real-invitation-code");
  await expect(page.getByText("This value could not be accepted.")).toBeVisible();
  await expectGojaAuthTheme(page);
});

test("Message Desk OIDC callback error is an application-styled recovery page", async ({ page }) => {
  await page.goto(`${messageOrigin}/auth/callback?error=access_denied&error_description=untrusted-provider-text&state=missing`);
  await expect(page.getByRole("heading", { name: "Sign-in was cancelled" })).toBeVisible();
  await expect(page.locator('link[rel="stylesheet"]')).toHaveAttribute("href", "/static/app/assets/index.css");
  await expect(page.getByRole("link", { name: "Try signing in again" })).toBeVisible();
  await expect(page.getByRole("link", { name: "Return to Message Desk" })).toBeVisible();
  expect(await page.locator("body").innerText()).not.toContain("untrusted-provider-text");
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

import { defineConfig } from "@playwright/test";

export default defineConfig({
  testDir: "./tests",
  outputDir: "./test-results",
  fullyParallel: false,
  workers: 1,
  timeout: 45_000,
  expect: { timeout: 8_000 },
  reporter: [["list"], ["html", { open: "never", outputFolder: "playwright-report" }]],
  use: {
    // The local CA is exported for human browsers and backend TLS clients.
    // Playwright's isolated Chromium profile has no persistent trust store, so
    // this exception is scoped to each disposable test context and these local
    // origins; the deployed applications still perform normal TLS validation.
    ignoreHTTPSErrors: true,
    trace: "retain-on-failure",
    screenshot: "only-on-failure",
    video: "retain-on-failure"
  },
  projects: [{ name: "chromium", use: { browserName: "chromium" } }]
});

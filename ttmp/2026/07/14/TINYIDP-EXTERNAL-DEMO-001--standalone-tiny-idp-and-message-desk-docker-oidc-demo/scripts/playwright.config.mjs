import { defineConfig } from "@playwright/test";

export default defineConfig({
  testDir: ".",
  testMatch: "02-external-demo.spec.mjs",
  fullyParallel: false,
  forbidOnly: Boolean(process.env.CI),
  retries: process.env.CI ? 1 : 0,
  workers: 1,
  reporter: [["list"]],
  timeout: 45_000,
  use: {
    baseURL: process.env.MESSAGE_DESK_URL || "http://localhost:8080",
    trace: "retain-on-failure",
    screenshot: "only-on-failure",
    video: "retain-on-failure"
  }
});

import { defineConfig, devices } from "@playwright/test";

const baseURL = process.env.PLAYWRIGHT_BASE_URL?.trim() || "http://127.0.0.1:5173";
const apiBaseURL = process.env.PLAYWRIGHT_API_BASE_URL?.trim() || "http://127.0.0.1:8080";
const testFilesPath = process.env.PLAYWRIGHT_TEST_FILES_PATH?.trim() || "/tmp";
const loginUsername = process.env.PLAYWRIGHT_USERNAME?.trim() || "admin";
const loginPassword = process.env.PLAYWRIGHT_PASSWORD?.trim() || "BootstrapSmoke1!";
const rotatedPassword = process.env.PLAYWRIGHT_ROTATED_PASSWORD?.trim() || "BootstrapSmoke2!";
const limitedUsername = process.env.PLAYWRIGHT_LIMITED_USERNAME?.trim() || "viewer";
const limitedPassword = process.env.PLAYWRIGHT_LIMITED_PASSWORD?.trim() || "ViewerPanel1!";

export default defineConfig({
  testDir: "./e2e",
  timeout: 60_000,
  expect: {
    timeout: 10_000
  },
  use: {
    baseURL,
    trace: "retain-on-failure"
  },
  reporter: [["list"], ["html", { open: "never" }]],
  projects: [
    {
      name: "chromium",
      use: {
        ...devices["Desktop Chrome"]
      }
    }
  ],
  metadata: {
    baseURL,
    apiBaseURL,
    testFilesPath,
    loginUsername,
    loginPassword,
    rotatedPassword,
    limitedUsername,
    limitedPassword
  }
});

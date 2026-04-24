import { ensureFixtureFile } from "./helpers/api";
import { test, expect, loginAndMaybeRotate } from "./fixtures";

test.describe("files page", () => {
  test("lists files and opens a text file", async ({ page, apiContext, bootstrapSession }, testInfo) => {
    const testFilesPath = String(testInfo.config.metadata.testFilesPath);
    const normalizedPath = testFilesPath.replace(/[\\/]+$/, "") || "/";
    const testFilePath = `${normalizedPath}/snowpanel-e2e.txt`;

    const loginResponse = await apiContext.post("/api/v1/auth/login", {
      data: {
        username: bootstrapSession.username,
        password: bootstrapSession.fallbackPassword ?? bootstrapSession.primaryPassword
      }
    });
    expect(loginResponse.ok()).toBeTruthy();
    const loginPayload = await loginResponse.json();
    expect(loginPayload.code).toBe(0);

    await ensureFixtureFile(apiContext, loginPayload.data.access_token, testFilePath);

    await loginAndMaybeRotate(page, bootstrapSession);
    await page.goto("/files");

    const pathInput = page.getByLabel("Current path");
    await pathInput.fill(testFilesPath);
    await page.getByRole("button", { name: /^load$/i }).click();

    await expect(page.getByRole("button", { name: /open snowpanel-e2e.txt/i })).toBeVisible();
    await page.getByRole("button", { name: /open snowpanel-e2e.txt/i }).click();

    await expect(page.getByText(testFilePath)).toBeVisible();
    await expect(page.getByLabel("File editor")).toHaveValue(/snowpanel e2e fixture/i);
  });
});

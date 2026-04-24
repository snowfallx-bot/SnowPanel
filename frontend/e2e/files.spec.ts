import { ensureFixtureFile } from "./helpers/api";
import { test, expect, loginAndMaybeRotate, loginViaApi } from "./fixtures";

test.describe("files page", () => {
  test("lists files and opens a text file", async ({ page, apiContext, bootstrapSession }, testInfo) => {
    const testFilesPath = String(testInfo.config.metadata.testFilesPath);
    const normalizedPath = testFilesPath.replace(/[\\/]+$/, "") || "/";
    const testFilePath = `${normalizedPath}/snowpanel-e2e.txt`;

    const session = await loginViaApi(apiContext, bootstrapSession);
    await ensureFixtureFile(apiContext, session.accessToken, testFilePath);

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

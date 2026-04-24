import { test, expect, loginAndMaybeRotate } from "./fixtures";

test.describe("authentication and navigation", () => {
  test("logs in and reaches dashboard", async ({ page, bootstrapSession }) => {
    await loginAndMaybeRotate(page, bootstrapSession);

    await expect(page).toHaveURL(/\/dashboard$/);
    await expect(page.getByRole("link", { name: "Dashboard" })).toBeVisible();
  });

  test("hides unauthorized navigation entries for a limited session", async ({ page }) => {
    await page.route("**/api/v1/auth/me", async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          code: 0,
          message: "ok",
          data: {
            id: 999,
            username: "viewer",
            email: "viewer@example.com",
            status: 1,
            roles: ["operator"],
            permissions: ["dashboard.read"],
            must_change_password: false
          }
        })
      });
    });

    await page.addInitScript(() => {
      localStorage.setItem(
        "snowpanel-auth",
        JSON.stringify({
          state: {
            token: "test-token",
            refreshToken: null,
            user: {
              id: 999,
              username: "viewer",
              email: "viewer@example.com",
              status: 1,
              roles: ["operator"],
              permissions: ["dashboard.read"],
              must_change_password: false
            }
          },
          version: 0
        })
      );
    });

    await page.goto("/dashboard");

    await expect(page.getByRole("link", { name: "Dashboard" })).toBeVisible();
    await expect(page.getByRole("link", { name: "Files" })).toHaveCount(0);
    await expect(page.getByRole("link", { name: "Services" })).toHaveCount(0);
    await expect(page.getByRole("link", { name: "Docker" })).toHaveCount(0);
    await expect(page.getByRole("link", { name: "Cron" })).toHaveCount(0);
    await expect(page.getByRole("link", { name: "Tasks" })).toHaveCount(0);
    await expect(page.getByRole("link", { name: "Audit" })).toHaveCount(0);
  });
});

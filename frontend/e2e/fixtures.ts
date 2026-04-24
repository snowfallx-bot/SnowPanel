import { test as base, expect, type APIRequestContext, type Page } from "@playwright/test";

type AuthSession = {
  username: string;
  primaryPassword: string;
  fallbackPassword?: string;
  rotatedPassword?: string;
};

type SnowPanelFixtures = {
  apiContext: APIRequestContext;
  bootstrapSession: AuthSession;
};

async function submitLogin(page: Page, username: string, password: string) {
  await page.getByLabel(/username/i).fill(username);
  await page.getByLabel(/password/i).fill(password);
  await page.getByRole("button", { name: /sign in/i }).click();
}

export async function loginAndMaybeRotate(page: Page, session: AuthSession) {
  await page.goto("/login");
  await expect(page.getByRole("heading", { name: /sign in to snowpanel/i })).toBeVisible();

  await submitLogin(page, session.username, session.primaryPassword);

  const invalidCredential = page.getByText(/invalid credentials|invalid credential/i);
  if (session.fallbackPassword && (await invalidCredential.isVisible().catch(() => false))) {
    await submitLogin(page, session.username, session.fallbackPassword);
  }

  const passwordGate = page.getByRole("heading", { name: /password change required/i });
  if (session.rotatedPassword && (await passwordGate.isVisible().catch(() => false))) {
    const currentPassword = session.fallbackPassword ?? session.primaryPassword;
    await page.getByLabel(/current password/i).fill(currentPassword);
    await page.getByLabel(/^new password$/i).fill(session.rotatedPassword);
    await page.getByLabel(/confirm new password/i).fill(session.rotatedPassword);
    await page.getByRole("button", { name: /update password/i }).click();
    await expect(page.getByText(/password updated/i)).toBeVisible();
  }

  await expect(page.getByText(/linux panel prototype/i)).toBeVisible();
}

export const test = base.extend<SnowPanelFixtures>({
  apiContext: async ({ playwright }, use, testInfo) => {
    const apiBaseURL = String(testInfo.config.metadata.apiBaseURL);
    const context = await playwright.request.newContext({
      baseURL: apiBaseURL
    });
    await use(context);
    await context.dispose();
  },
  bootstrapSession: async ({}, use, testInfo) => {
    await use({
      username: String(testInfo.config.metadata.loginUsername),
      primaryPassword: String(testInfo.config.metadata.loginPassword),
      fallbackPassword: String(testInfo.config.metadata.rotatedPassword),
      rotatedPassword: String(testInfo.config.metadata.rotatedPassword)
    });
  }
});

export { expect };

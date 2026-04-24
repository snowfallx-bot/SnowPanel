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

  const passwordGate = page.getByRole("heading", { name: /password change required/i });
  const shellMarker = page.getByText(/linux panel prototype/i);
  if (session.rotatedPassword && (await passwordGate.isVisible().catch(() => false))) {
    await page.getByLabel(/current password/i).fill(session.primaryPassword);
    await page.getByLabel(/^new password$/i).fill(session.rotatedPassword);
    await page.getByLabel(/confirm new password/i).fill(session.rotatedPassword);
    await page.getByRole("button", { name: /update password/i }).click();
    await expect(page.getByText(/password updated/i)).toBeVisible();
    return;
  }

  if (session.fallbackPassword) {
    const invalidCredential = page.getByText(/invalid credentials|invalid credential/i);
    await Promise.race([
      shellMarker.waitFor({ state: "visible", timeout: 4_000 }).catch(() => undefined),
      invalidCredential.waitFor({ state: "visible", timeout: 4_000 }).catch(() => undefined)
    ]);

    if (await invalidCredential.isVisible().catch(() => false)) {
      await submitLogin(page, session.username, session.fallbackPassword);
    }
  }

  await expect(shellMarker).toBeVisible();
}

export async function loginViaApi(
  apiContext: APIRequestContext,
  session: AuthSession
): Promise<{ accessToken: string; refreshToken: string; rotated: boolean }> {
  const primaryResponse = await apiContext.post("/api/v1/auth/login", {
    data: {
      username: session.username,
      password: session.primaryPassword
    }
  });
  const primaryPayload = await primaryResponse.json();
  if (primaryResponse.ok() && primaryPayload.code === 0) {
    if (primaryPayload.data?.user?.must_change_password === true && session.rotatedPassword) {
      const changeResponse = await apiContext.post("/api/v1/auth/change-password", {
        headers: {
          Authorization: `Bearer ${primaryPayload.data.access_token}`
        },
        data: {
          current_password: session.primaryPassword,
          new_password: session.rotatedPassword
        }
      });
      const changePayload = await changeResponse.json();
      expect(changeResponse.ok()).toBeTruthy();
      expect(changePayload.code).toBe(0);
      return {
        accessToken: changePayload.data.access_token,
        refreshToken: changePayload.data.refresh_token,
        rotated: true
      };
    }

    return {
      accessToken: primaryPayload.data.access_token,
      refreshToken: primaryPayload.data.refresh_token,
      rotated: false
    };
  }

  if (session.fallbackPassword) {
    const fallbackResponse = await apiContext.post("/api/v1/auth/login", {
      data: {
        username: session.username,
        password: session.fallbackPassword
      }
    });
    const fallbackPayload = await fallbackResponse.json();
    expect(fallbackResponse.ok()).toBeTruthy();
    expect(fallbackPayload.code).toBe(0);
    return {
      accessToken: fallbackPayload.data.access_token,
      refreshToken: fallbackPayload.data.refresh_token,
      rotated: false
    };
  }

  throw new Error(`API login failed for ${session.username}`);
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

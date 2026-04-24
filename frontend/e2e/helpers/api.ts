import { APIRequestContext, expect } from "@playwright/test";

export async function ensureViewerUser(apiContext: APIRequestContext, adminToken: string) {
  const response = await apiContext.get("/api/v1/auth/me", {
    headers: {
      Authorization: `Bearer ${adminToken}`
    }
  });

  await expect(response.ok()).toBeTruthy();
}

export async function ensureFixtureFile(apiContext: APIRequestContext, adminToken: string, path: string) {
  const response = await apiContext.post("/api/v1/files/write", {
    headers: {
      Authorization: `Bearer ${adminToken}`
    },
    data: {
      path,
      content: "snowpanel e2e fixture",
      create_if_not_exists: true,
      truncate: true,
      encoding: "utf-8"
    }
  });

  await expect(response.ok()).toBeTruthy();
  const payload = await response.json();
  expect(payload.code).toBe(0);
}

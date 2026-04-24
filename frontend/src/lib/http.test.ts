import { describe, expect, it } from "vitest";
import { ApiError, describeApiError } from "@/lib/http";

describe("describeApiError", () => {
  it("maps agent unavailable errors to actionable copy", () => {
    const result = describeApiError(
      new ApiError("core agent unavailable: context deadline exceeded", { code: 3001, status: 503 }),
      "fallback"
    );

    expect(result.message).toBe("core-agent is unavailable.");
    expect(result.hint).toContain("core-agent systemd service");
  });

  it("maps missing routes to deployment mismatch guidance", () => {
    const result = describeApiError(
      new ApiError("Request failed with status code 404", { status: 404 }),
      "fallback"
    );

    expect(result.message).toBe("Requested API route was not found.");
    expect(result.hint).toContain("Frontend and backend versions may be mismatched");
  });
});

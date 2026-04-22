import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it } from "vitest";
import { LoginPage } from "@/pages/LoginPage";
import { useAuthStore } from "@/store/auth-store";

describe("LoginPage", () => {
  beforeEach(() => {
    localStorage.clear();
    useAuthStore.getState().clearAuth();
  });

  it("renders the login form", () => {
    render(
      <MemoryRouter>
        <LoginPage />
      </MemoryRouter>
    );

    expect(screen.getByText("Sign In To SnowPanel")).toBeInTheDocument();
    expect(screen.queryByDisplayValue("admin")).not.toBeInTheDocument();
    expect(screen.queryByDisplayValue("admin123456")).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Sign In" })).toBeInTheDocument();
  });
});

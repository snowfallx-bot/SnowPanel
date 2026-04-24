import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { changePassword, logout } from "@/api/auth";
import { AppLayout } from "@/layouts/AppLayout";
import { useAuthStore } from "@/store/auth-store";
import { LoginResult, UserProfile } from "@/types/auth";

vi.mock("@/api/auth", () => ({
  changePassword: vi.fn(),
  logout: vi.fn()
}));

const operatorProfile: UserProfile = {
  id: 1,
  username: "operator",
  email: "operator@example.com",
  status: 1,
  roles: ["operator"],
  permissions: ["dashboard.read", "docker.read"],
  must_change_password: false
};

function renderLayout(initialEntry = "/dashboard") {
  render(
    <MemoryRouter initialEntries={[initialEntry]}>
      <Routes>
        <Route element={<AppLayout />}>
          <Route path="/dashboard" element={<div>Dashboard page</div>} />
          <Route path="/docker" element={<div>Docker page</div>} />
        </Route>
        <Route path="/login" element={<div>Login page</div>} />
      </Routes>
    </MemoryRouter>
  );
}

describe("AppLayout", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    localStorage.clear();
    useAuthStore.setState({
      hydrated: true,
      token: "token-1",
      refreshToken: "refresh-1",
      user: operatorProfile
    });
  });

  it("shows only navigation items granted by permissions", () => {
    renderLayout("/dashboard");

    expect(screen.getByRole("link", { name: "Dashboard" })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Docker" })).toBeInTheDocument();
    expect(screen.queryByRole("link", { name: "Files" })).not.toBeInTheDocument();
    expect(screen.queryByRole("link", { name: "Audit" })).not.toBeInTheDocument();
  });

  it("forces bootstrap users to change password and refreshes local session", async () => {
    const bootstrapProfile: UserProfile = {
      ...operatorProfile,
      must_change_password: true
    };
    const rotatedSession: LoginResult = {
      access_token: "token-2",
      refresh_token: "refresh-2",
      token_type: "Bearer",
      expires_in: 3600,
      refresh_expires_in: 86400,
      user: {
        ...bootstrapProfile,
        must_change_password: false
      }
    };
    vi.mocked(changePassword).mockResolvedValueOnce(rotatedSession);
    useAuthStore.setState({
      token: "token-1",
      refreshToken: "refresh-1",
      user: bootstrapProfile
    });

    renderLayout("/dashboard");

    expect(screen.getByText("Password Change Required")).toBeInTheDocument();

    fireEvent.change(screen.getByLabelText("Current Password"), {
      target: { value: "OldPassw0rd!!" }
    });
    fireEvent.change(screen.getByLabelText("New Password"), {
      target: { value: "NewStrongPassw0rd!!" }
    });
    fireEvent.change(screen.getByLabelText("Confirm New Password"), {
      target: { value: "NewStrongPassw0rd!!" }
    });

    fireEvent.click(screen.getByRole("button", { name: "Update Password" }));

    await waitFor(() => {
      expect(changePassword).toHaveBeenCalledWith({
        current_password: "OldPassw0rd!!",
        new_password: "NewStrongPassw0rd!!"
      });
    });
    await waitFor(() => {
      expect(useAuthStore.getState().token).toBe("token-2");
    });
    expect(useAuthStore.getState().user?.must_change_password).toBe(false);
    await waitFor(() => {
      expect(screen.queryByText("Password Change Required")).not.toBeInTheDocument();
    });
  });

  it("clears local credentials even when backend logout fails", async () => {
    vi.mocked(logout).mockRejectedValueOnce(new Error("network error"));

    renderLayout("/dashboard");

    fireEvent.click(screen.getByRole("button", { name: "Logout" }));

    await waitFor(() => {
      expect(logout).toHaveBeenCalledTimes(1);
      expect(useAuthStore.getState().token).toBeNull();
      expect(useAuthStore.getState().user).toBeNull();
    });
  });
});

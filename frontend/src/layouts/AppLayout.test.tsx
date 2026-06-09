import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { changePassword, logout } from "@/api/auth";
import { listHosts } from "@/api/hosts";
import { AppLayout } from "@/layouts/AppLayout";
import { useAuthStore } from "@/store/auth-store";
import { useHostStore } from "@/store/host-store";
import { LoginResult, UserProfile } from "@/types/auth";

vi.mock("@/api/auth", () => ({
  changePassword: vi.fn(),
  logout: vi.fn()
}));

vi.mock("@/api/hosts", () => ({
  listHosts: vi.fn()
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
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: {
        retry: false
      }
    }
  });

  render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={[initialEntry]}>
        <Routes>
          <Route element={<AppLayout />}>
            <Route path="/dashboard" element={<div>Dashboard page</div>} />
            <Route path="/docker" element={<div>Docker page</div>} />
          </Route>
          <Route path="/login" element={<div>Login page</div>} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>
  );
}

describe("AppLayout", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    localStorage.clear();
    useHostStore.setState({ selectedHostId: null });
    vi.mocked(listHosts).mockResolvedValue({
      items: []
    });
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
      expect(useHostStore.getState().selectedHostId).toBeNull();
    });
  });

  it("shows host selector for users with host read permission and persists selected host", async () => {
    vi.mocked(listHosts).mockResolvedValue({
      items: [
        {
          id: 7,
          name: "edge-node",
          address: "10.0.0.7",
          port: 50051,
          status: "online",
          agent_version: "0.1.0",
          created_at: "2026-04-27T00:00:00Z",
          updated_at: "2026-04-27T00:00:00Z"
        }
      ]
    });
    useAuthStore.setState({
      hydrated: true,
      token: "token-1",
      refreshToken: "refresh-1",
      user: {
        ...operatorProfile,
        permissions: [...operatorProfile.permissions, "hosts.read"]
      }
    });

    renderLayout("/dashboard");

    const selector = await screen.findByLabelText("Target Host");
    fireEvent.change(selector, { target: { value: "7" } });

    await waitFor(() => {
      expect(useHostStore.getState().selectedHostId).toBe(7);
    });
    expect(screen.getByText(/Target:/)).toHaveTextContent("edge-node");
  });
});

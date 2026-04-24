import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { getMe } from "@/api/auth";
import { ApiError } from "@/lib/http";
import { ProtectedRoute } from "@/routes/ProtectedRoute";
import { useAuthStore } from "@/store/auth-store";
import { UserProfile } from "@/types/auth";

vi.mock("@/api/auth", () => ({
  getMe: vi.fn()
}));

const dashboardProfile: UserProfile = {
  id: 1,
  username: "operator",
  email: "operator@example.com",
  status: 1,
  roles: ["operator"],
  permissions: ["dashboard.read"],
  must_change_password: false
};

function createDeferred<T>() {
  let resolve!: (value: T | PromiseLike<T>) => void;
  let reject!: (reason?: unknown) => void;
  const promise = new Promise<T>((res, rej) => {
    resolve = res;
    reject = rej;
  });
  return { promise, resolve, reject };
}

function renderProtectedRoute(initialEntry = "/dashboard") {
  render(
    <MemoryRouter initialEntries={[initialEntry]}>
      <Routes>
        <Route element={<ProtectedRoute />}>
          <Route path="/dashboard" element={<div>Dashboard content</div>} />
          <Route path="/files" element={<div>Files content</div>} />
        </Route>
        <Route path="/login" element={<div>Login page</div>} />
      </Routes>
    </MemoryRouter>
  );
}

describe("ProtectedRoute", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    localStorage.clear();
    useAuthStore.setState({
      hydrated: true,
      token: null,
      refreshToken: null,
      user: null
    });
  });

  it("redirects to login when no token is present", async () => {
    renderProtectedRoute("/dashboard");

    expect(await screen.findByText("Login page")).toBeInTheDocument();
  });

  it("waits for getMe before rendering protected content", async () => {
    const deferred = createDeferred<UserProfile>();
    vi.mocked(getMe).mockImplementationOnce(() => deferred.promise);
    useAuthStore.setState({
      token: "token-1",
      refreshToken: "refresh-1",
      user: dashboardProfile
    });

    renderProtectedRoute("/dashboard");

    expect(screen.getByText("Validating session...")).toBeInTheDocument();
    expect(screen.queryByText("Dashboard content")).not.toBeInTheDocument();

    deferred.resolve(dashboardProfile);

    expect(await screen.findByText("Dashboard content")).toBeInTheDocument();
    expect(getMe).toHaveBeenCalledTimes(1);
  });

  it("clears auth and redirects to login when getMe returns 401", async () => {
    vi.mocked(getMe).mockRejectedValueOnce(
      new ApiError("session expired", { status: 401 })
    );
    useAuthStore.setState({
      token: "token-1",
      refreshToken: "refresh-1",
      user: dashboardProfile
    });

    renderProtectedRoute("/dashboard");

    expect(await screen.findByText("Login page")).toBeInTheDocument();
    await waitFor(() => {
      expect(useAuthStore.getState().token).toBeNull();
    });
  });

  it("shows retryable session validation error for non-auth failures", async () => {
    vi.mocked(getMe)
      .mockRejectedValueOnce(new Error("Network Error"))
      .mockResolvedValueOnce(dashboardProfile);
    useAuthStore.setState({
      token: "token-1",
      refreshToken: "refresh-1",
      user: dashboardProfile
    });

    renderProtectedRoute("/dashboard");

    expect(await screen.findByText("Unable to validate session")).toBeInTheDocument();
    expect(screen.getByText("Unable to reach the SnowPanel API.")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Retry validation" }));

    expect(await screen.findByText("Dashboard content")).toBeInTheDocument();
    expect(getMe).toHaveBeenCalledTimes(2);
  });

  it("redirects to fallback route when user lacks page permission", async () => {
    vi.mocked(getMe).mockResolvedValueOnce(dashboardProfile);
    useAuthStore.setState({
      token: "token-1",
      refreshToken: "refresh-1",
      user: dashboardProfile
    });

    renderProtectedRoute("/files");

    expect(await screen.findByText("Dashboard content")).toBeInTheDocument();
    expect(screen.queryByText("Files content")).not.toBeInTheDocument();
  });
});

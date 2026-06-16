import { ChangeEvent, FormEvent, useEffect, useMemo, useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { NavLink, Outlet } from "react-router-dom";
import { changePassword, logout } from "@/api/auth";
import { listHosts } from "@/api/hosts";
import { useAuthStore } from "@/store/auth-store";
import { hostScopeKey, useHostStore } from "@/store/host-store";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { describeApiError } from "@/lib/http";

const navItems: Array<{ to: string; label: string; permission: string }> = [
  { to: "/dashboard", label: "Dashboard", permission: "dashboard.read" },
  { to: "/files", label: "Files", permission: "files.read" },
  { to: "/services", label: "Services", permission: "services.read" },
  { to: "/docker", label: "Docker", permission: "docker.read" },
  { to: "/cron", label: "Cron", permission: "cron.read" },
  { to: "/hosts", label: "Hosts", permission: "hosts.read" },
  { to: "/tasks", label: "Tasks", permission: "tasks.read" },
  { to: "/audit", label: "Audit", permission: "audit.read" },
  { to: "/websites", label: "Websites", permission: "websites.read" },
  { to: "/settings", label: "Settings", permission: "settings.read" }
];

export function AppLayout() {
  const queryClient = useQueryClient();
  const token = useAuthStore((state) => state.token);
  const user = useAuthStore((state) => state.user);
  const setAuth = useAuthStore((state) => state.setAuth);
  const clearAuth = useAuthStore((state) => state.clearAuth);
  const selectedHostId = useHostStore((state) => state.selectedHostId);
  const setSelectedHostId = useHostStore((state) => state.setSelectedHostId);
  const clearSelectedHost = useHostStore((state) => state.clearSelectedHost);
  const permissionSet = new Set(user?.permissions ?? []);
  const visibleNavItems = navItems.filter((item) => permissionSet.has(item.permission));
  const mustChangePassword = user?.must_change_password === true;
  const canReadHosts = permissionSet.has("hosts.read");
  const hostsQuery = useQuery({
    queryKey: ["hosts", "registry"],
    queryFn: listHosts,
    enabled: canReadHosts && Boolean(token)
  });

  const [currentPassword, setCurrentPassword] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [changingPassword, setChangingPassword] = useState(false);
  const [loggingOut, setLoggingOut] = useState(false);
  const [passwordError, setPasswordError] = useState("");
  const [passwordSuccess, setPasswordSuccess] = useState("");

  const selectedHost = useMemo(
    () => hostsQuery.data?.items.find((item) => item.id === selectedHostId) ?? null,
    [hostsQuery.data?.items, selectedHostId]
  );
  const selectedHostScope = hostScopeKey(selectedHostId);
  const hostsLoadError = hostsQuery.isError
    ? describeApiError(hostsQuery.error, "Failed to load hosts.")
    : null;

  useEffect(() => {
    if (!selectedHostId || hostsQuery.isLoading || hostsQuery.isError) {
      return;
    }
    if (!selectedHost) {
      setSelectedHostId(null);
    }
  }, [hostsQuery.isError, hostsQuery.isLoading, selectedHost, selectedHostId, setSelectedHostId]);

  async function handlePasswordChange(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setPasswordError("");
    setPasswordSuccess("");

    if (!currentPassword || !newPassword) {
      setPasswordError("Current password and new password are required.");
      return;
    }
    if (newPassword !== confirmPassword) {
      setPasswordError("New password and confirm password do not match.");
      return;
    }
    if (!token) {
      setPasswordError("Session is missing. Please log in again.");
      return;
    }

    setChangingPassword(true);
    try {
      const result = await changePassword({
        current_password: currentPassword,
        new_password: newPassword
      });
      setAuth(result.access_token, result.user, result.refresh_token ?? null);
      setCurrentPassword("");
      setNewPassword("");
      setConfirmPassword("");
      setPasswordSuccess("Password updated.");
    } catch (error) {
      setPasswordError(error instanceof Error ? error.message : "Failed to change password");
    } finally {
      setChangingPassword(false);
    }
  }

  async function handleLogout() {
    setLoggingOut(true);
    try {
      if (token) {
        await logout();
      }
    } catch {
      // Even if backend logout fails, clear local credentials for user safety.
    } finally {
      clearAuth();
      clearSelectedHost();
      queryClient.clear();
      setLoggingOut(false);
    }
  }

  function handleHostChange(event: ChangeEvent<HTMLSelectElement>) {
    const nextValue = event.target.value.trim();
    const nextHostId = nextValue ? Number(nextValue) : null;
    setSelectedHostId(Number.isFinite(nextHostId) ? nextHostId : null);
    queryClient.removeQueries({
      predicate(query) {
        return query.queryKey[0] !== "hosts";
      }
    });
  }

  return (
    <div className="min-h-screen bg-slate-100 text-slate-900 md:grid md:grid-cols-[240px_1fr]">
      <aside className="border-r border-slate-200 bg-panel-900 px-5 py-6 text-panel-50">
        <h1 className="mb-5 text-xl font-semibold">SnowPanel</h1>
        <nav aria-label="Primary">
          {visibleNavItems.map((item) => (
            <NavLink
              key={item.to}
              to={item.to}
              className={({ isActive }) =>
                [
                  "mb-2 block rounded-md px-3 py-2 text-sm transition",
                  isActive ? "bg-panel-700 text-white" : "text-panel-100 hover:bg-panel-800"
                ].join(" ")
              }
            >
              {item.label}
            </NavLink>
          ))}
        </nav>
      </aside>
      <main className="flex flex-col">
        <header className="flex items-center justify-between border-b border-slate-200 bg-white px-6 py-4">
          <div>
            <p className="text-sm text-slate-500">SnowPanel Operations Console</p>
            <p className="text-base font-medium">{user?.username ?? "unknown"}</p>
            <p className="text-xs text-slate-500">
              Target:{" "}
              {selectedHost
                ? `${selectedHost.name} (${selectedHost.address}:${selectedHost.port})`
                : "Primary agent"}
            </p>
          </div>
          <div className="flex items-center gap-3">
            {canReadHosts && (
              <div className="min-w-[260px]">
                <label className="block text-xs font-medium uppercase tracking-wide text-slate-500" htmlFor="target-host">
                  Target Host
                </label>
                <select
                  id="target-host"
                  className="mt-1 h-9 w-full rounded-md border border-slate-200 bg-white px-3 text-sm text-slate-700"
                  value={selectedHostId ?? ""}
                  onChange={handleHostChange}
                >
                  <option value="">Primary agent (default)</option>
                  {(hostsQuery.data?.items || []).map((item) => (
                    <option key={item.id} value={item.id}>
                      {item.name} [{item.status}] - {item.address}:{item.port}
                    </option>
                  ))}
                </select>
                {hostsQuery.isLoading && <p className="mt-1 text-xs text-slate-500">Loading hosts...</p>}
                {hostsLoadError && <p className="mt-1 text-xs text-rose-600">{hostsLoadError.message}</p>}
              </div>
            )}
            <Button variant="ghost" onClick={handleLogout} disabled={loggingOut}>
              {loggingOut ? "Logging out..." : "Logout"}
            </Button>
          </div>
        </header>
        <section className="p-6">
          <div key={selectedHostScope}>
            <Outlet />
          </div>
        </section>
      </main>
      {mustChangePassword && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/70 p-4">
          <Card className="w-full max-w-md border-slate-300 bg-white">
            <CardHeader>
              <CardTitle>Password Change Required</CardTitle>
              <p className="text-sm text-slate-600">
                For security reasons, update your bootstrap password before continuing.
              </p>
              <p className="text-xs text-slate-500">
                Password policy: at least 14 characters with upper/lower letters, digits, and symbols.
              </p>
            </CardHeader>
            <CardContent>
              <form className="space-y-3" onSubmit={handlePasswordChange}>
                <div className="space-y-1">
                  <label
                    className="text-sm font-medium text-slate-700"
                    htmlFor="current-password"
                  >
                    Current Password
                  </label>
                  <Input
                    id="current-password"
                    type="password"
                    value={currentPassword}
                    onChange={(event) => setCurrentPassword(event.target.value)}
                  />
                </div>
                <div className="space-y-1">
                  <label className="text-sm font-medium text-slate-700" htmlFor="new-password">
                    New Password
                  </label>
                  <Input
                    id="new-password"
                    type="password"
                    value={newPassword}
                    onChange={(event) => setNewPassword(event.target.value)}
                  />
                </div>
                <div className="space-y-1">
                  <label
                    className="text-sm font-medium text-slate-700"
                    htmlFor="confirm-new-password"
                  >
                    Confirm New Password
                  </label>
                  <Input
                    id="confirm-new-password"
                    type="password"
                    value={confirmPassword}
                    onChange={(event) => setConfirmPassword(event.target.value)}
                  />
                </div>
                {passwordError && <p className="text-sm text-rose-600">{passwordError}</p>}
                {passwordSuccess && <p className="text-sm text-emerald-700">{passwordSuccess}</p>}
                <Button className="w-full" disabled={changingPassword} type="submit">
                  {changingPassword ? "Updating..." : "Update Password"}
                </Button>
              </form>
            </CardContent>
          </Card>
        </div>
      )}
    </div>
  );
}

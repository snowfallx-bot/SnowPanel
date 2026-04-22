import { FormEvent, useState } from "react";
import { NavLink, Outlet } from "react-router-dom";
import { changePassword } from "@/api/auth";
import { useAuthStore } from "@/store/auth-store";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";

const navItems: Array<{ to: string; label: string; permission: string }> = [
  { to: "/dashboard", label: "Dashboard", permission: "dashboard.read" },
  { to: "/files", label: "Files", permission: "files.read" },
  { to: "/services", label: "Services", permission: "services.read" },
  { to: "/docker", label: "Docker", permission: "docker.read" },
  { to: "/cron", label: "Cron", permission: "cron.read" },
  { to: "/tasks", label: "Tasks", permission: "tasks.read" },
  { to: "/audit", label: "Audit", permission: "audit.read" }
];

export function AppLayout() {
  const token = useAuthStore((state) => state.token);
  const user = useAuthStore((state) => state.user);
  const setAuth = useAuthStore((state) => state.setAuth);
  const clearAuth = useAuthStore((state) => state.clearAuth);
  const permissionSet = new Set(user?.permissions ?? []);
  const visibleNavItems = navItems.filter((item) => permissionSet.has(item.permission));
  const mustChangePassword = user?.must_change_password === true;

  const [currentPassword, setCurrentPassword] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [changingPassword, setChangingPassword] = useState(false);
  const [passwordError, setPasswordError] = useState("");
  const [passwordSuccess, setPasswordSuccess] = useState("");

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
      setAuth(result.access_token, result.user);
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

  return (
    <div className="min-h-screen bg-slate-100 text-slate-900 md:grid md:grid-cols-[240px_1fr]">
      <aside className="border-r border-slate-200 bg-panel-900 px-5 py-6 text-panel-50">
        <h1 className="mb-5 text-xl font-semibold">SnowPanel</h1>
        <nav>
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
            <p className="text-sm text-slate-500">Linux Panel Prototype</p>
            <p className="text-base font-medium">{user?.username ?? "unknown"}</p>
          </div>
          <Button variant="ghost" onClick={clearAuth}>
            Logout
          </Button>
        </header>
        <section className="p-6">
          <Outlet />
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
                  <label className="text-sm font-medium text-slate-700">Current Password</label>
                  <Input
                    type="password"
                    value={currentPassword}
                    onChange={(event) => setCurrentPassword(event.target.value)}
                  />
                </div>
                <div className="space-y-1">
                  <label className="text-sm font-medium text-slate-700">New Password</label>
                  <Input
                    type="password"
                    value={newPassword}
                    onChange={(event) => setNewPassword(event.target.value)}
                  />
                </div>
                <div className="space-y-1">
                  <label className="text-sm font-medium text-slate-700">Confirm New Password</label>
                  <Input
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

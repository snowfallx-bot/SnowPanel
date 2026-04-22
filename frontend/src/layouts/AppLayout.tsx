import { NavLink, Outlet } from "react-router-dom";
import { useAuthStore } from "@/store/auth-store";
import { Button } from "@/components/ui/button";

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
  const user = useAuthStore((state) => state.user);
  const clearAuth = useAuthStore((state) => state.clearAuth);
  const permissionSet = new Set(user?.permissions ?? []);
  const visibleNavItems = navItems.filter((item) => permissionSet.has(item.permission));

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
    </div>
  );
}

import { NavLink, Outlet } from "react-router-dom";
import { useAuthStore } from "@/store/auth-store";
import { Button } from "@/components/ui/button";

export function AppLayout() {
  const user = useAuthStore((state) => state.user);
  const clearAuth = useAuthStore((state) => state.clearAuth);

  return (
    <div className="min-h-screen bg-slate-100 text-slate-900 md:grid md:grid-cols-[240px_1fr]">
      <aside className="border-r border-slate-200 bg-panel-900 px-5 py-6 text-panel-50">
        <h1 className="mb-5 text-xl font-semibold">SnowPanel</h1>
        <nav>
          <NavLink
            to="/dashboard"
            className={({ isActive }) =>
              [
                "mb-2 block rounded-md px-3 py-2 text-sm transition",
                isActive ? "bg-panel-700 text-white" : "text-panel-100 hover:bg-panel-800"
              ].join(" ")
            }
          >
            Dashboard
          </NavLink>
          <NavLink
            to="/files"
            className={({ isActive }) =>
              [
                "mb-2 block rounded-md px-3 py-2 text-sm transition",
                isActive ? "bg-panel-700 text-white" : "text-panel-100 hover:bg-panel-800"
              ].join(" ")
            }
          >
            Files
          </NavLink>
          <NavLink
            to="/services"
            className={({ isActive }) =>
              [
                "mb-2 block rounded-md px-3 py-2 text-sm transition",
                isActive ? "bg-panel-700 text-white" : "text-panel-100 hover:bg-panel-800"
              ].join(" ")
            }
          >
            Services
          </NavLink>
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

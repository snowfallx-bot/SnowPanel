import { NavLink, Outlet } from "react-router-dom";

export function AppLayout() {
  return (
    <div className="app-shell">
      <aside className="sidebar">
        <h1 className="logo">SnowPanel</h1>
        <nav>
          <NavLink to="/" end className="nav-link">
            Dashboard
          </NavLink>
          <NavLink to="/overview" className="nav-link">
            Overview
          </NavLink>
        </nav>
      </aside>
      <main className="content">
        <header className="topbar">
          <span>Linux Panel Prototype</span>
        </header>
        <section className="page-body">
          <Outlet />
        </section>
      </main>
    </div>
  );
}

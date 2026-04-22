import { Navigate, createBrowserRouter } from "react-router-dom";
import { AppLayout } from "@/layouts/AppLayout";
import { DashboardPage } from "@/pages/DashboardPage";
import { DockerPage } from "@/pages/DockerPage";
import { CronPage } from "@/pages/CronPage";
import { AuditLogsPage } from "@/pages/AuditLogsPage";
import { TasksPage } from "@/pages/TasksPage";
import { FilesPage } from "@/pages/FilesPage";
import { LoginPage } from "@/pages/LoginPage";
import { ServicesPage } from "@/pages/ServicesPage";
import { ProtectedRoute } from "@/routes/ProtectedRoute";

export const router = createBrowserRouter([
  {
    path: "/login",
    element: <LoginPage />
  },
  {
    path: "/",
    element: <ProtectedRoute />,
    children: [
      {
        path: "/",
        element: <AppLayout />,
        children: [
          {
            index: true,
            element: <Navigate to="/dashboard" replace />
          },
          {
            path: "/dashboard",
            element: <DashboardPage />
          },
          {
            path: "/files",
            element: <FilesPage />
          },
          {
            path: "/services",
            element: <ServicesPage />
          },
          {
            path: "/docker",
            element: <DockerPage />
          },
          {
            path: "/cron",
            element: <CronPage />
          },
          {
            path: "/tasks",
            element: <TasksPage />
          },
          {
            path: "/audit",
            element: <AuditLogsPage />
          }
        ]
      }
    ]
  },
  {
    path: "*",
    element: <Navigate to="/dashboard" replace />
  }
]);

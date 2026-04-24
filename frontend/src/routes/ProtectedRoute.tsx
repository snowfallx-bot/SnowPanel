import { useEffect, useMemo, useState } from "react";
import { Navigate, Outlet, useLocation } from "react-router-dom";
import { getMe } from "@/api/auth";
import { QueryErrorCard } from "@/components/ui/query-error-card";
import { ApiError, ApiErrorDisplay, describeApiError } from "@/lib/http";
import { useAuthStore } from "@/store/auth-store";

const routePermissionRules: Array<{ prefix: string; permission: string }> = [
  { prefix: "/dashboard", permission: "dashboard.read" },
  { prefix: "/files", permission: "files.read" },
  { prefix: "/services", permission: "services.read" },
  { prefix: "/docker", permission: "docker.read" },
  { prefix: "/cron", permission: "cron.read" },
  { prefix: "/tasks", permission: "tasks.read" },
  { prefix: "/audit", permission: "audit.read" }
];

function requiredPermissionForPath(pathname: string): string | null {
  for (const rule of routePermissionRules) {
    if (pathname === rule.prefix || pathname.startsWith(`${rule.prefix}/`)) {
      return rule.permission;
    }
  }
  return null;
}

export function ProtectedRoute() {
  const hydrated = useAuthStore((state) => state.hydrated);
  const token = useAuthStore((state) => state.token);
  const refreshToken = useAuthStore((state) => state.refreshToken);
  const user = useAuthStore((state) => state.user);
  const setAuth = useAuthStore((state) => state.setAuth);
  const clearAuth = useAuthStore((state) => state.clearAuth);
  const location = useLocation();
  const [checkingSession, setCheckingSession] = useState(false);
  const [validatedToken, setValidatedToken] = useState<string | null>(null);
  const [sessionError, setSessionError] = useState<ApiErrorDisplay | null>(null);
  const [sessionRetryKey, setSessionRetryKey] = useState(0);

  useEffect(() => {
    let alive = true;
    if (!hydrated || !token) {
      setCheckingSession(false);
      setValidatedToken(null);
      setSessionError(null);
      return () => {
        alive = false;
      };
    }

    setCheckingSession(true);
    setSessionError(null);
    getMe()
      .then((profile) => {
        if (!alive) {
          return;
        }
        setAuth(token, profile, refreshToken);
        setValidatedToken(token);
      })
      .catch((error: unknown) => {
        if (!alive) {
          return;
        }
        const status = error instanceof ApiError ? error.status : undefined;
        if (status === 401 || status === 403) {
          clearAuth();
          setValidatedToken(null);
          return;
        }
        setSessionError(describeApiError(error, "Failed to validate session."));
      })
      .finally(() => {
        if (alive) {
          setCheckingSession(false);
        }
      });

    return () => {
      alive = false;
    };
  }, [hydrated, token, refreshToken, setAuth, clearAuth, sessionRetryKey]);

  const requiredPermission = useMemo(
    () => requiredPermissionForPath(location.pathname),
    [location.pathname]
  );
  const needsSessionValidation = Boolean(token) && validatedToken !== token;

  const hasPermission = useMemo(() => {
    if (!requiredPermission) {
      return true;
    }
    if (!user) {
      return true;
    }
    return user?.permissions?.includes(requiredPermission) ?? false;
  }, [requiredPermission, user?.permissions]);

  if (!hydrated) {
    return null;
  }

  if (checkingSession || (needsSessionValidation && !sessionError)) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-slate-100 text-slate-600">
        Validating session...
      </div>
    );
  }

  if (needsSessionValidation && sessionError) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-slate-100 p-4">
        <div className="w-full max-w-lg">
          <QueryErrorCard
            title="Unable to validate session"
            message={sessionError.message}
            hint={sessionError.hint}
            onRetry={() => setSessionRetryKey((current) => current + 1)}
            retryLabel="Retry validation"
          />
        </div>
      </div>
    );
  }

  if (!token) {
    return <Navigate to="/login" replace state={{ from: location.pathname }} />;
  }

  if (!hasPermission) {
    const fallback =
      user?.permissions?.includes("dashboard.read") === true ? "/dashboard" : "/login";
    return (
      <Navigate
        to={fallback}
        replace
        state={{ from: location.pathname, message: "You do not have permission to access this page." }}
      />
    );
  }

  return <Outlet />;
}

import axios, { AxiosRequestConfig } from "axios";
import { useAuthStore } from "@/store/auth-store";
import { LoginResult } from "@/types/auth";
import { ApiEnvelope } from "@/types/api";

const AUTH_REDIRECT_MESSAGE_KEY = "snowpanel-auth-redirect-message";
const RAW_API_BASE_URL = import.meta.env.VITE_API_BASE_URL?.trim();
const API_BASE_URL =
  !RAW_API_BASE_URL || RAW_API_BASE_URL === "/"
    ? ""
    : RAW_API_BASE_URL.replace(/\/+$/, "");

const http = axios.create({
  baseURL: API_BASE_URL,
  timeout: 10000
});

const refreshClient = axios.create({
  baseURL: API_BASE_URL,
  timeout: 10000
});

let refreshPromise: Promise<string | null> | null = null;

export class ApiError extends Error {
  code?: number;
  status?: number;
  cause?: unknown;

  constructor(message: string, options?: { code?: number; status?: number; cause?: unknown }) {
    super(message);
    this.name = "ApiError";
    this.code = options?.code;
    this.status = options?.status;
    this.cause = options?.cause;
  }
}

export interface ApiErrorDisplay {
  message: string;
  hint?: string;
}

function normalizeErrorMessage(message: string | undefined, fallback: string) {
  const trimmed = message?.trim();
  return trimmed ? trimmed : fallback;
}

export function describeApiError(error: unknown, fallback: string): ApiErrorDisplay {
  if (error instanceof ApiError) {
    const message = normalizeErrorMessage(error.message, fallback);

    if (error.code === 3001 || error.status === 503) {
      return {
        message: "core-agent is unavailable.",
        hint: "Check the host-agent compose override, the core-agent systemd service, and backend-to-agent connectivity."
      };
    }

    if (error.status === 401) {
      return {
        message: "Session expired or invalid.",
        hint: "Please sign in again."
      };
    }

    if (error.status === 403) {
      return {
        message: "You do not have permission to access this resource.",
        hint: "Ask an administrator to grant the required permission."
      };
    }

    if (error.status === 404) {
      return {
        message: "Requested API route was not found.",
        hint: "Frontend and backend versions may be mismatched, or the proxy target may be incorrect."
      };
    }

    if (error.status && error.status >= 500) {
      return {
        message: "SnowPanel backend returned an internal error.",
        hint: "Check backend logs, verify database and agent health, then retry."
      };
    }

    if (/timeout/i.test(message)) {
      return {
        message: "Request timed out while waiting for the SnowPanel API.",
        hint: "Check backend health, reverse proxy configuration, and server load."
      };
    }

    if (/network error|failed to fetch|load failed|request failed/i.test(message)) {
      return {
        message: "Unable to reach the SnowPanel API.",
        hint: "Check frontend proxy settings, backend port exposure, and browser network access."
      };
    }

    return { message };
  }

  if (error instanceof Error) {
    const message = normalizeErrorMessage(error.message, fallback);
    if (/timeout/i.test(message)) {
      return {
        message: "Request timed out while waiting for the SnowPanel API.",
        hint: "Check backend health, reverse proxy configuration, and server load."
      };
    }
    return { message };
  }

  return { message: fallback };
}

http.interceptors.request.use((config) => {
  const token = useAuthStore.getState().token;
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

http.interceptors.response.use(
  (response) => response,
  async (error) => {
    const status = error.response?.status;
    const originalConfig = error.config as (AxiosRequestConfig & { _retry?: boolean }) | undefined;
    const requestURL = typeof originalConfig?.url === "string" ? originalConfig.url : "";
    const isAuthBootstrapEndpoint =
      requestURL.includes("/api/v1/auth/login") || requestURL.includes("/api/v1/auth/refresh");

    if (status === 401 && originalConfig && !originalConfig._retry && !isAuthBootstrapEndpoint) {
      originalConfig._retry = true;
      const nextAccessToken = await queueRefreshToken();
      if (nextAccessToken) {
        originalConfig.headers = originalConfig.headers ?? {};
        (originalConfig.headers as Record<string, string>).Authorization = `Bearer ${nextAccessToken}`;
        return http(originalConfig);
      }
    }

    if (status === 401) {
      useAuthStore.getState().clearAuth();
      if (typeof window !== "undefined" && window.location.pathname !== "/login") {
        window.sessionStorage.setItem(
          AUTH_REDIRECT_MESSAGE_KEY,
          "Session expired or invalid. Please log in again."
        );
        window.location.assign("/login");
      }
    }
    return Promise.reject(error);
  }
);

async function queueRefreshToken(): Promise<string | null> {
  if (!refreshPromise) {
    refreshPromise = refreshAccessToken().finally(() => {
      refreshPromise = null;
    });
  }
  return refreshPromise;
}

async function refreshAccessToken(): Promise<string | null> {
  const { refreshToken, setAuth, clearAuth } = useAuthStore.getState();
  if (!refreshToken) {
    return null;
  }

  try {
    const { data } = await refreshClient.post<ApiEnvelope<LoginResult>>("/api/v1/auth/refresh", {
      refresh_token: refreshToken
    });

    if (data.code !== 0) {
      throw new ApiError(data.message, { code: data.code, status: 401 });
    }

    const result = data.data;
    setAuth(result.access_token, result.user, result.refresh_token ?? null);
    return result.access_token;
  } catch {
    clearAuth();
    return null;
  }
}

export async function unwrap<T>(promise: Promise<{ data: ApiEnvelope<T> }>): Promise<T> {
  try {
    const { data } = await promise;
    if (data.code !== 0) {
      throw new ApiError(data.message, { code: data.code });
    }
    return data.data;
  } catch (error) {
    if (axios.isAxiosError(error)) {
      const status = error.response?.status;
      const payload = error.response?.data as Partial<ApiEnvelope<unknown>> | undefined;
      if (payload && typeof payload.message === "string") {
        throw new ApiError(payload.message, {
          code: typeof payload.code === "number" ? payload.code : undefined,
          status,
          cause: error
        });
      }
      throw new ApiError(error.message || "Request failed", {
        status,
        cause: error
      });
    }

    if (error instanceof ApiError) {
      throw error;
    }
    if (error instanceof Error) {
      throw new ApiError(error.message, { cause: error });
    }

    throw new ApiError("Request failed");
  }
}

export { AUTH_REDIRECT_MESSAGE_KEY };
export { http };

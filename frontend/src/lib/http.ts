import axios from "axios";
import { useAuthStore } from "@/store/auth-store";
import { ApiEnvelope } from "@/types/api";

const AUTH_REDIRECT_MESSAGE_KEY = "snowpanel-auth-redirect-message";

const http = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL ?? "http://127.0.0.1:8080",
  timeout: 10000
});

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

http.interceptors.request.use((config) => {
  const token = useAuthStore.getState().token;
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

http.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
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

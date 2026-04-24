import React from "react";
import ReactDOM from "react-dom/client";
import { RouterProvider } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ApiError } from "@/lib/http";
import { router } from "./routes";
import "./index.css";

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry(failureCount, error) {
        if (error instanceof ApiError) {
          if (error.status && error.status >= 400 && error.status < 600) {
            return false;
          }
          if (typeof error.code === "number") {
            return false;
          }
        }
        return failureCount < 1;
      }
    },
    mutations: {
      retry: false
    }
  }
});

ReactDOM.createRoot(document.getElementById("root") as HTMLElement).render(
  <React.StrictMode>
    <QueryClientProvider client={queryClient}>
      <RouterProvider router={router} />
    </QueryClientProvider>
  </React.StrictMode>
);

import { http, unwrap } from "@/lib/http";
import { DashboardSummary } from "@/types/dashboard";

export function getDashboardSummary() {
  return unwrap<DashboardSummary>(http.get("/api/v1/dashboard/summary"));
}

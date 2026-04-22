import { http, unwrap } from "@/lib/http";
import { ListAuditLogsResult } from "@/types/audit";

export function listAuditLogs(params: { page: number; size: number; module?: string; action?: string }) {
  return unwrap<ListAuditLogsResult>(http.get("/api/v1/audit/logs", { params }));
}

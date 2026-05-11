import { http, unwrap } from "@/lib/http";
import { ListAuditLogsResult } from "@/types/audit";

export interface ListAuditLogsParams {
  page: number;
  size: number;
  start_time?: string;
  end_time?: string;
  user_id?: number;
  username?: string;
  module?: string;
  action?: string;
  target_type?: string;
  target_id?: string;
  success?: boolean;
  result_code?: string;
  request_id?: string;
  trace_id?: string;
}

export function listAuditLogs(params: ListAuditLogsParams) {
  return unwrap<ListAuditLogsResult>(http.get("/api/v1/audit/logs", { params }));
}

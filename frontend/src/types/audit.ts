export interface AuditLog {
  id: number;
  user_id: number | null;
  username: string;
  ip: string;
  module: string;
  action: string;
  target_type: string;
  target_id: string;
  request_summary: string;
  success: boolean;
  result_code: string;
  result_message: string;
  created_at: string;
}

export interface ListAuditLogsResult {
  page: number;
  size: number;
  total: number;
  items: AuditLog[];
}
